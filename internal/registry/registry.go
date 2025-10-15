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

// Registry represents the project-local registry structure stored in $PROJECT_ROOT/.dual/.local/registry.json
type Registry struct {
	Projects    map[string]Project `json:"projects"`
	mu          sync.RWMutex       `json:"-"`
	flock       *flock.Flock       `json:"-"` // File lock for atomic operations
	projectRoot string             `json:"-"` // Project root path for SaveRegistry
}

// Project represents a single project in the registry
type Project struct {
	Contexts map[string]Context `json:"contexts"`
}

// ContextEnvOverrides represents environment overrides at different levels
type ContextEnvOverrides struct {
	Global   map[string]string            `json:"global,omitempty"`   // Global overrides for all services
	Services map[string]map[string]string `json:"services,omitempty"` // Service-specific overrides
}

// Context represents a development context (branch, worktree, etc.)
type Context struct {
	Created        time.Time            `json:"created"`
	Path           string               `json:"path,omitempty"`
	EnvOverridesV2 *ContextEnvOverrides `json:"envOverridesV2,omitempty"` // Layered overrides
}

var (
	// ErrProjectNotFound is returned when a project doesn't exist in the registry
	ErrProjectNotFound = errors.New("project not found in registry")
	// ErrContextNotFound is returned when a context doesn't exist in a project
	ErrContextNotFound = errors.New("context not found in project")
	// ErrLockTimeout is returned when file lock acquisition times out
	ErrLockTimeout = errors.New("timeout waiting for registry lock")
	// LockTimeout is the timeout for acquiring the registry lock
	LockTimeout = 5 * time.Second
)

// GetRegistryPath returns the path to the project-local registry file
func GetRegistryPath(projectRoot string) (string, error) {
	return filepath.Join(projectRoot, ".dual", ".local", "registry.json"), nil
}

// GetLockPath returns the path to the project-local registry lock file
func GetLockPath(projectRoot string) (string, error) {
	return filepath.Join(projectRoot, ".dual", ".local", "registry.json.lock"), nil
}

// LoadRegistry reads the registry from $PROJECT_ROOT/.dual/.local/registry.json with file locking
// If the file doesn't exist or is corrupt, it returns a new empty registry
// The caller MUST call Close() on the returned registry to release the lock
func LoadRegistry(projectRoot string) (*Registry, error) {
	registryPath, err := GetRegistryPath(projectRoot)
	if err != nil {
		return nil, err
	}

	lockPath, err := GetLockPath(projectRoot)
	if err != nil {
		return nil, err
	}

	// Ensure directory exists before creating lock file
	registryDir := filepath.Dir(registryPath)
	if err := os.MkdirAll(registryDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create project-local registry directory: %w", err)
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
		// Provide detailed guidance for lock timeout
		return nil, fmt.Errorf("%w\n\n"+
			"DETAILS:\n"+
			"  Lock file:    %s\n"+
			"  Waited:       %v\n"+
			"\n"+
			"POSSIBLE CAUSES:\n"+
			"  • Another dual command is currently running\n"+
			"  • A previous dual command crashed without releasing the lock\n"+
			"  • File permissions issue on the lock file\n"+
			"\n"+
			"SOLUTIONS:\n"+
			"  1. Wait for other dual commands to complete\n"+
			"\n"+
			"  2. Check for running dual processes:\n"+
			"     ps aux | grep dual\n"+
			"\n"+
			"  3. If no dual commands are running, remove stale lock:\n"+
			"     rm %s\n"+
			"\n"+
			"  ⚠️  Only remove the lock file if you're certain no dual\n"+
			"     commands are currently running!",
			ErrLockTimeout, lockPath, LockTimeout, lockPath)
	}

	// Initialize registry
	registry := &Registry{
		Projects:    make(map[string]Project),
		mu:          sync.RWMutex{},
		flock:       fileLock,
		projectRoot: projectRoot,
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
		// Create backup of corrupted registry
		backupPath := registryPath + ".corrupt." + time.Now().Format("20060102-150405")
		_ = os.WriteFile(backupPath, data, 0o600) // Best effort backup

		// Provide detailed error recovery information
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "ERROR: Registry file is corrupted\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  Registry file: %s\n", registryPath)
		fmt.Fprintf(os.Stderr, "  Backup saved:  %s\n", backupPath)
		fmt.Fprintf(os.Stderr, "  Parse error:   %v\n", err)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "IMPACT:\n")
		fmt.Fprintf(os.Stderr, "  • A new empty registry will be created\n")
		fmt.Fprintf(os.Stderr, "  • Your worktrees still exist but aren't registered\n")
		fmt.Fprintf(os.Stderr, "  • Environment overrides have been lost\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "TO RECOVER:\n")
		fmt.Fprintf(os.Stderr, "  1. Re-register existing worktrees:\n")
		fmt.Fprintf(os.Stderr, "     dual create <branch-name> for each worktree\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  2. Or try to fix the backup file:\n")
		fmt.Fprintf(os.Stderr, "     cat %s | jq . > %s\n", backupPath, registryPath)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  3. Run 'dual doctor' to diagnose issues\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")

		return registry, nil
	}

	// Load projects into registry
	if loadedData.Projects != nil {
		registry.Projects = loadedData.Projects
	}

	return registry, nil
}

// SaveRegistry writes the registry to $PROJECT_ROOT/.dual/.local/registry.json atomically
// Uses the stored projectRoot field from LoadRegistry
func (r *Registry) SaveRegistry() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registryPath, err := GetRegistryPath(r.projectRoot)
	if err != nil {
		return err
	}

	// Ensure directory exists
	registryDir := filepath.Dir(registryPath)
	if err := os.MkdirAll(registryDir, 0o750); err != nil {
		return fmt.Errorf("failed to create project-local registry directory: %w", err)
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
func (r *Registry) SetContext(projectPath, contextName string, contextPath string) error {
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
		Created: time.Now(),
		Path:    contextPath,
	}

	// Preserve existing overrides if context already exists
	if exists && existingContext.EnvOverridesV2 != nil {
		newContext.EnvOverridesV2 = existingContext.EnvOverridesV2
	}

	project.Contexts[contextName] = newContext

	return nil
}

// SetEnvOverrideGlobal sets a global environment variable override for a context
func (r *Registry) SetEnvOverride(projectPath, contextName, key, value string) error {
	return r.SetEnvOverrideForService(projectPath, contextName, key, value, "")
}

// SetEnvOverrideForService sets an environment variable override for a context and optional service
// If serviceName is empty, sets a global override
func (r *Registry) SetEnvOverrideForService(projectPath, contextName, key, value, serviceName string) error {
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

	// Use context method to set override
	context.SetEnvOverride(key, value, serviceName)
	project.Contexts[contextName] = context

	return nil
}

// UnsetEnvOverride removes a global environment variable override for a context
func (r *Registry) UnsetEnvOverride(projectPath, contextName, key string) error {
	return r.UnsetEnvOverrideForService(projectPath, contextName, key, "")
}

// UnsetEnvOverrideForService removes an environment variable override for a context and optional service
// If serviceName is empty, removes from global overrides
func (r *Registry) UnsetEnvOverrideForService(projectPath, contextName, key, serviceName string) error {
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

	// Use context method to unset override
	context.UnsetEnvOverride(key, serviceName)
	project.Contexts[contextName] = context

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

// GetEnvOverrides returns environment overrides for a context
// serviceName can be empty string for global overrides
func (c *Context) GetEnvOverrides(serviceName string) map[string]string {
	// If nil, return empty map
	if c.EnvOverridesV2 == nil {
		return make(map[string]string)
	}

	// Merge global and service-specific overrides
	result := make(map[string]string)

	// Start with global overrides
	for k, v := range c.EnvOverridesV2.Global {
		result[k] = v
	}

	// Apply service-specific overrides if service is specified
	if serviceName != "" && c.EnvOverridesV2.Services != nil {
		if serviceOverrides, exists := c.EnvOverridesV2.Services[serviceName]; exists {
			for k, v := range serviceOverrides {
				result[k] = v
			}
		}
	}

	return result
}

// SetEnvOverride sets an environment override for a context
// serviceName can be empty string for global overrides
func (c *Context) SetEnvOverride(key, value, serviceName string) {
	// Ensure EnvOverridesV2 is initialized
	if c.EnvOverridesV2 == nil {
		c.EnvOverridesV2 = &ContextEnvOverrides{
			Global:   make(map[string]string),
			Services: make(map[string]map[string]string),
		}
	}

	if serviceName == "" {
		// Global override
		if c.EnvOverridesV2.Global == nil {
			c.EnvOverridesV2.Global = make(map[string]string)
		}
		c.EnvOverridesV2.Global[key] = value
	} else {
		// Service-specific override
		if c.EnvOverridesV2.Services == nil {
			c.EnvOverridesV2.Services = make(map[string]map[string]string)
		}
		if c.EnvOverridesV2.Services[serviceName] == nil {
			c.EnvOverridesV2.Services[serviceName] = make(map[string]string)
		}
		c.EnvOverridesV2.Services[serviceName][key] = value
	}
}

// UnsetEnvOverride removes an environment override for a context
// serviceName can be empty string for global overrides
func (c *Context) UnsetEnvOverride(key, serviceName string) {
	if c.EnvOverridesV2 == nil {
		return
	}

	if serviceName == "" {
		// Remove from global
		if c.EnvOverridesV2.Global != nil {
			delete(c.EnvOverridesV2.Global, key)
		}
	} else {
		// Remove from service-specific
		if c.EnvOverridesV2.Services != nil && c.EnvOverridesV2.Services[serviceName] != nil {
			delete(c.EnvOverridesV2.Services[serviceName], key)
		}
	}
}

// GetEnvOverrideValue returns the value of a specific override
// Returns empty string if not found
func (c *Context) GetEnvOverrideValue(key, serviceName string) string {
	overrides := c.GetEnvOverrides(serviceName)
	return overrides[key]
}

// HasEnvOverride checks if an override exists
func (c *Context) HasEnvOverride(key, serviceName string) bool {
	overrides := c.GetEnvOverrides(serviceName)
	_, exists := overrides[key]
	return exists
}
