package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/worktree"
	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the name of the configuration file
	ConfigFileName = "dual.config.yml"
	// SupportedVersion is the currently supported config schema version
	SupportedVersion = 1
)

// Config represents the dual.config.yml structure
type Config struct {
	Services  map[string]Service  `yaml:"services"`
	Version   int                 `yaml:"version"`
	Env       EnvConfig           `yaml:"env,omitempty"`
	Worktrees WorktreeConfig      `yaml:"worktrees,omitempty"`
	Hooks     map[string][]string `yaml:"hooks,omitempty"`
}

// EnvConfig contains environment-related configuration
type EnvConfig struct {
	// BaseFile is the path to the base environment file (relative to project root)
	BaseFile string `yaml:"baseFile,omitempty"`
}

// WorktreeConfig contains worktree-related configuration
type WorktreeConfig struct {
	// Path is the base directory for worktrees (relative to project root)
	// Example: "../worktrees" or "worktrees"
	Path string `yaml:"path,omitempty"`

	// Naming is the pattern for worktree directory names
	// Supports: "branch" (use branch name as-is), "prefix-{branch}", etc.
	// Default: "branch"
	Naming string `yaml:"naming,omitempty"`
}

// Service represents a single service configuration
type Service struct {
	Path    string `yaml:"path"`
	EnvFile string `yaml:"envFile"`
}

// LoadConfig searches for dual.config.yml starting from the current directory
// and walking up the directory tree until it finds the file or reaches the root.
// It returns the parsed config and the absolute path of the project root.
// For worktrees, the project root is the directory where the config was found
// (which will be the worktree directory for worktrees sharing the config).
// Use GetProjectIdentifier() to get the normalized identifier for the registry.
func LoadConfig() (*Config, string, error) {
	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	searchDir := currentDir
	var configDir string
	for {
		configPath := filepath.Join(searchDir, ConfigFileName)

		// Check if config file exists
		if _, err := os.Stat(configPath); err == nil {
			configDir = searchDir
			break
		}

		// Move up one directory
		parentDir := filepath.Dir(searchDir)

		// Check if we've reached the root
		if parentDir == searchDir {
			return nil, "", fmt.Errorf("no %s found in current directory or any parent directory", ConfigFileName)
		}

		searchDir = parentDir
	}

	// Found the config file at configDir
	configPath := filepath.Join(configDir, ConfigFileName)

	// Parse the config
	config, err := parseConfig(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	// The project root is the directory where the config was found
	// This allows service paths to be resolved correctly in both main repo and worktrees
	projectRoot := configDir

	// Validate the config against the project root
	if err := validateConfig(config, projectRoot); err != nil {
		return nil, "", fmt.Errorf("invalid config in %s: %w", configPath, err)
	}

	return config, projectRoot, nil
}

// parseConfig reads and parses a YAML config file
func parseConfig(path string) (*Config, error) {
	// #nosec G304 - path is from trusted source (config file search)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// validateConfig checks that the config has valid structure and values
func validateConfig(config *Config, projectRoot string) error {
	// Check version
	if config.Version == 0 {
		return fmt.Errorf("version field is required")
	}
	if config.Version != SupportedVersion {
		return fmt.Errorf("unsupported config version %d (expected %d)", config.Version, SupportedVersion)
	}

	// Services can be empty (for initial setup), but if present, validate them
	for name, service := range config.Services {
		if err := validateService(name, service, projectRoot); err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
	}

	// Validate worktree configuration if present
	if config.Worktrees.Path != "" {
		if filepath.IsAbs(config.Worktrees.Path) {
			return fmt.Errorf("worktrees.path must be relative to project root, got absolute path: %s", config.Worktrees.Path)
		}
		// Note: We don't check if the worktrees directory exists because it may not exist yet
		// It will be created by the 'dual create' command
	}

	// Validate hooks if present
	if len(config.Hooks) > 0 {
		if err := validateHooks(config.Hooks, projectRoot); err != nil {
			return fmt.Errorf("hooks: %w", err)
		}
	}

	return nil
}

// validateService checks that a service configuration is valid
func validateService(name string, service Service, projectRoot string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if service.Path == "" {
		return fmt.Errorf("path is required")
	}

	// Check if path is absolute (it shouldn't be - should be relative to project root)
	if filepath.IsAbs(service.Path) {
		return fmt.Errorf("path must be relative to project root, got absolute path: %s", service.Path)
	}

	// Validate that the path exists
	fullPath := filepath.Join(projectRoot, service.Path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", service.Path)
		}
		return fmt.Errorf("failed to check path: %w", err)
	}

	// Path should be a directory
	if !info.IsDir() {
		return fmt.Errorf("path must be a directory, got file: %s", service.Path)
	}

	// EnvFile is optional, but if provided, validate the directory exists
	if service.EnvFile != "" {
		if filepath.IsAbs(service.EnvFile) {
			return fmt.Errorf("envFile must be relative to project root, got absolute path: %s", service.EnvFile)
		}

		// Check if the directory containing the env file exists
		envFileFullPath := filepath.Join(projectRoot, service.EnvFile)
		envFileDir := filepath.Dir(envFileFullPath)
		if _, err := os.Stat(envFileDir); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("envFile directory does not exist: %s", filepath.Dir(service.EnvFile))
			}
			return fmt.Errorf("failed to check envFile directory: %w", err)
		}
	}

	return nil
}

// validateHooks checks that hook definitions are valid
func validateHooks(hooks map[string][]string, projectRoot string) error {
	validEvents := map[string]bool{
		"postWorktreeCreate": true,
		"postPortAssign":     true,
		"preWorktreeDelete":  true,
		"postWorktreeDelete": true,
	}

	for event, scripts := range hooks {
		if !validEvents[event] {
			return fmt.Errorf("invalid hook event: %s (valid events: postWorktreeCreate, postPortAssign, preWorktreeDelete, postWorktreeDelete)", event)
		}

		for _, script := range scripts {
			// Hook scripts are relative to .dual/hooks/ directory
			hookPath := filepath.Join(projectRoot, ".dual", "hooks", script)

			// Check if hook script exists (warning if missing, not error)
			if _, err := os.Stat(hookPath); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "[dual] Warning: hook script not found: %s\n", hookPath)
			}
		}
	}

	return nil
}

// SaveConfig writes a config to the specified path atomically
func SaveConfig(config *Config, path string) error {
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to temporary file
	tempFile := path + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temporary config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, path); err != nil {
		_ = os.Remove(tempFile) // Clean up temp file on error
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// LoadConfigFrom loads a config from a specific path (useful for testing)
func LoadConfigFrom(path string) (*Config, error) {
	config, err := parseConfig(path)
	if err != nil {
		return nil, err
	}

	projectRoot := filepath.Dir(path)
	if err := validateConfig(config, projectRoot); err != nil {
		return nil, err
	}

	return config, nil
}

// GetProjectIdentifier returns the normalized project identifier for the registry.
// For worktrees, this returns the parent repository path so all worktrees share
// the same project entry in the registry. For normal repos, returns the projectRoot.
func GetProjectIdentifier(projectRoot string) (string, error) {
	wtDetector := worktree.NewDetector()

	// Try to detect if we're in a worktree
	gitRoot, err := wtDetector.FindGitRoot(projectRoot)
	if err != nil {
		// Not in a git repo, use projectRoot as-is
		return projectRoot, nil
	}

	// Get the normalized project root (parent repo for worktrees)
	normalizedRoot, err := wtDetector.GetProjectRoot(gitRoot)
	if err != nil {
		// If detection fails, use projectRoot as-is
		return projectRoot, nil
	}

	return normalizedRoot, nil
}

// GetWorktreePath returns the absolute path to the worktrees directory
func (c *Config) GetWorktreePath(projectRoot string) string {
	if c.Worktrees.Path == "" {
		// Default to ../worktrees if not specified
		return filepath.Join(filepath.Dir(projectRoot), "worktrees")
	}
	return filepath.Join(projectRoot, c.Worktrees.Path)
}

// GetWorktreeName returns the worktree directory name for a given branch
func (c *Config) GetWorktreeName(branchName string) string {
	if c.Worktrees.Naming == "" {
		// Default to branch name as-is
		return branchName
	}
	// Support simple replacement (future: could support more complex patterns)
	return strings.ReplaceAll(c.Worktrees.Naming, "{branch}", branchName)
}

// GetHookScripts returns the list of hook scripts for a given event
func (c *Config) GetHookScripts(event string) []string {
	if scripts, exists := c.Hooks[event]; exists {
		return scripts
	}
	return nil
}
