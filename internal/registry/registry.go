package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

// Registry represents the global registry structure stored in ~/.dual/registry.json
type Registry struct {
	Projects map[string]Project `json:"projects"`
	mu       sync.RWMutex       `json:"-"`
	flock    *flock.Flock       `json:"-"` // File lock for atomic operations
}

// Project represents a single project in the registry
type Project struct {
	Contexts map[string]Context `json:"contexts"`
}

// Context represents a development context (branch, worktree, etc.)
type Context struct {
	Created      time.Time         `json:"created"`
	Path         string            `json:"path,omitempty"`
	BasePort     int               `json:"basePort"`
	EnvOverrides map[string]string `json:"envOverrides,omitempty"` // Optional: environment variable overrides
}

var (
	// ErrProjectNotFound is returned when a project doesn't exist in the registry
	ErrProjectNotFound = errors.New("project not found in registry")
	// ErrContextNotFound is returned when a context doesn't exist in a project
	ErrContextNotFound = errors.New("context not found in project")
	// ErrLockTimeout is returned when file lock acquisition times out
	ErrLockTimeout = errors.New("timeout waiting for registry lock")
	// DefaultBasePort is the starting port for new contexts
	DefaultBasePort = 4100
	// PortIncrement is the increment between base ports
	PortIncrement = 100
	// LockTimeout is the timeout for acquiring the registry lock
	LockTimeout = 5 * time.Second
)

// GetRegistryPath returns the path to the registry file
func GetRegistryPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".dual", "registry.json"), nil
}

// GetLockPath returns the path to the registry lock file
func GetLockPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".dual", "registry.json.lock"), nil
}

// LoadRegistry reads the registry from ~/.dual/registry.json with file locking
// If the file doesn't exist or is corrupt, it returns a new empty registry
// The caller MUST call Close() on the returned registry to release the lock
func LoadRegistry() (*Registry, error) {
	registryPath, err := GetRegistryPath()
	if err != nil {
		return nil, err
	}

	lockPath, err := GetLockPath()
	if err != nil {
		return nil, err
	}

	// Ensure directory exists before creating lock file
	registryDir := filepath.Dir(registryPath)
	if err := os.MkdirAll(registryDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create registry directory: %w", err)
	}

	// Create file lock
	fileLock := flock.New(lockPath)

	// Try to acquire lock with timeout
	ctx, cancel := context.WithTimeout(context.Background(), LockTimeout)
	defer cancel()

	locked, err := fileLock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire registry lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("%w: another dual command may be running (waited %v)", ErrLockTimeout, LockTimeout)
	}

	// Initialize registry
	registry := &Registry{
		Projects: make(map[string]Project),
		mu:       sync.RWMutex{},
		flock:    fileLock,
	}

	// If file doesn't exist, return empty registry (but keep the lock)
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		return registry, nil
	}

	// Read the file
	// #nosec G304 - registryPath is from trusted GetRegistryPath() function
	data, err := os.ReadFile(registryPath)
	if err != nil {
		// Release lock before returning error
		_ = fileLock.Unlock()
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	// Parse JSON
	var loadedData struct {
		Projects map[string]Project `json:"projects"`
	}
	if err := json.Unmarshal(data, &loadedData); err != nil {
		// If corrupt, log warning but continue with empty registry (keep the lock)
		fmt.Fprintf(os.Stderr, "[dual] Warning: corrupt registry file, creating new one: %v\n", err)
		return registry, nil
	}

	// Load projects into registry
	if loadedData.Projects != nil {
		registry.Projects = loadedData.Projects
	}

	return registry, nil
}

// SaveRegistry writes the registry to ~/.dual/registry.json atomically
func (r *Registry) SaveRegistry() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registryPath, err := GetRegistryPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	registryDir := filepath.Dir(registryPath)
	if err := os.MkdirAll(registryDir, 0o750); err != nil {
		return fmt.Errorf("failed to create registry directory: %w", err)
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}

	// Write to temporary file
	tempFile := registryPath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temporary registry: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, registryPath); err != nil {
		_ = os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// GetContext retrieves a context for a given project
func (r *Registry) GetContext(projectPath, contextName string) (*Context, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, exists := r.Projects[projectPath]
	if !exists {
		return nil, ErrProjectNotFound
	}

	context, exists := project.Contexts[contextName]
	if !exists {
		return nil, ErrContextNotFound
	}

	return &context, nil
}

// SetContext creates or updates a context for a given project
func (r *Registry) SetContext(projectPath, contextName string, basePort int, contextPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure project exists
	project, exists := r.Projects[projectPath]
	if !exists {
		project = Project{
			Contexts: make(map[string]Context),
		}
		r.Projects[projectPath] = project
	}

	// Set or update context
	// Preserve existing env overrides if updating
	existingContext, exists := project.Contexts[contextName]
	newContext := Context{
		Created:      time.Now(),
		Path:         contextPath,
		BasePort:     basePort,
		EnvOverrides: make(map[string]string),
	}

	// Preserve existing overrides if context already exists
	if exists && existingContext.EnvOverrides != nil {
		newContext.EnvOverrides = existingContext.EnvOverrides
	}

	project.Contexts[contextName] = newContext

	return nil
}

// SetEnvOverride sets an environment variable override for a context
func (r *Registry) SetEnvOverride(projectPath, contextName, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, exists := r.Projects[projectPath]
	if !exists {
		return ErrProjectNotFound
	}

	context, exists := project.Contexts[contextName]
	if !exists {
		return ErrContextNotFound
	}

	// Initialize EnvOverrides map if nil
	if context.EnvOverrides == nil {
		context.EnvOverrides = make(map[string]string)
	}

	context.EnvOverrides[key] = value
	project.Contexts[contextName] = context

	return nil
}

// UnsetEnvOverride removes an environment variable override for a context
func (r *Registry) UnsetEnvOverride(projectPath, contextName, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, exists := r.Projects[projectPath]
	if !exists {
		return ErrProjectNotFound
	}

	context, exists := project.Contexts[contextName]
	if !exists {
		return ErrContextNotFound
	}

	if context.EnvOverrides != nil {
		delete(context.EnvOverrides, key)
		project.Contexts[contextName] = context
	}

	return nil
}

// DeleteContext removes a context from a project
func (r *Registry) DeleteContext(projectPath, contextName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	project, exists := r.Projects[projectPath]
	if !exists {
		return ErrProjectNotFound
	}

	if _, exists := project.Contexts[contextName]; !exists {
		return ErrContextNotFound
	}

	delete(project.Contexts, contextName)

	// Clean up empty project
	if len(project.Contexts) == 0 {
		delete(r.Projects, projectPath)
	}

	return nil
}

// ListContexts returns all contexts for a given project
func (r *Registry) ListContexts(projectPath string) (map[string]Context, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, exists := r.Projects[projectPath]
	if !exists {
		return nil, ErrProjectNotFound
	}

	// Return a copy to prevent external modifications
	contexts := make(map[string]Context)
	for name, ctx := range project.Contexts {
		contexts[name] = ctx
	}

	return contexts, nil
}

// FindNextAvailablePort scans all existing base ports and returns the next available one
// It increments by PortIncrement (default 100)
func (r *Registry) FindNextAvailablePort() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	usedPorts := make(map[int]bool)

	// Collect all used base ports
	for _, project := range r.Projects {
		for _, context := range project.Contexts {
			usedPorts[context.BasePort] = true
		}
	}

	// Find next available port starting from DefaultBasePort
	nextPort := DefaultBasePort
	for {
		if !usedPorts[nextPort] {
			return nextPort
		}
		nextPort += PortIncrement
	}
}

// GetAllProjects returns a list of all project paths in the registry
func (r *Registry) GetAllProjects() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]string, 0, len(r.Projects))
	for projectPath := range r.Projects {
		projects = append(projects, projectPath)
	}

	sort.Strings(projects)
	return projects
}

// ContextExists checks if a context exists for a given project
func (r *Registry) ContextExists(projectPath, contextName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, exists := r.Projects[projectPath]
	if !exists {
		return false
	}

	_, exists = project.Contexts[contextName]
	return exists
}

// Close releases the file lock on the registry
// This MUST be called after LoadRegistry() to prevent lock leaks
func (r *Registry) Close() error {
	if r.flock != nil {
		if err := r.flock.Unlock(); err != nil {
			return fmt.Errorf("failed to release registry lock: %w", err)
		}
	}
	return nil
}
