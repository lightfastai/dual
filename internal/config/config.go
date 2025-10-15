package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	dualerrors "github.com/lightfastai/dual/internal/errors"
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
			err := dualerrors.New(dualerrors.ErrConfigNotFound, fmt.Sprintf("No %s found", ConfigFileName))
			err = err.WithContext("Started from", currentDir)
			err = err.WithContext("Searched up to", searchDir)
			err = err.WithFixes(
				"Initialize dual in your project root: cd <project-root> && dual init",
				fmt.Sprintf("Or create %s manually with this structure:", ConfigFileName),
				"",
				"  version: 1",
				"  services:",
				"    web:",
				"      path: ./apps/web",
				"    api:",
				"      path: ./apps/api",
			)
			return nil, "", err
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
		// Enhanced error handling for common YAML issues
		errStr := err.Error()

		// Check for hooks type mismatch (string vs array)
		if strings.Contains(errStr, "cannot unmarshal !!str") && strings.Contains(errStr, "into []string") {
			// Extract the problematic line if possible
			lineInfo := ""
			if strings.Contains(errStr, "line") {
				parts := strings.Split(errStr, ":")
				for _, part := range parts {
					if strings.Contains(part, "line") {
						lineInfo = strings.TrimSpace(part)
						break
					}
				}
			}

			dualErr := dualerrors.New(dualerrors.ErrConfigInvalid, "YAML parsing failed: hooks must be arrays, not strings")
			if lineInfo != "" {
				dualErr = dualErr.WithContext("Location", lineInfo)
			}
			dualErr = dualErr.WithContext("File", path)
			dualErr = dualErr.WithCause(err)
			dualErr = dualErr.WithFixes(
				"Change hook configuration to use array format:",
				"",
				"  Incorrect:  postWorktreeCreate: setup.sh",
				"  Correct:    postWorktreeCreate:",
				"                - setup.sh",
				"",
				"  Multiple scripts example:",
				"  postWorktreeCreate:",
				"    - setup-database.sh",
				"    - install-deps.sh",
			)
			return nil, dualErr
		}

		// Check for indentation errors
		if strings.Contains(errStr, "found character that cannot start any token") ||
			strings.Contains(errStr, "did not find expected") ||
			strings.Contains(errStr, "could not find expected") {
			dualErr := dualerrors.New(dualerrors.ErrConfigInvalid, "YAML parsing failed: indentation or syntax error")
			dualErr = dualErr.WithContext("File", path)
			dualErr = dualErr.WithCause(err)
			dualErr = dualErr.WithFixes(
				"Check YAML indentation and syntax:",
				"• Use consistent spacing (2 or 4 spaces)",
				"• Never use tabs for indentation",
				"• Ensure proper structure with colons and hyphens",
				"• Validate YAML syntax at: https://www.yamllint.com/",
			)
			return nil, dualErr
		}

		// Check for other type mismatches
		if strings.Contains(errStr, "cannot unmarshal") {
			dualErr := dualerrors.New(dualerrors.ErrConfigInvalid, "YAML parsing failed: type mismatch")
			dualErr = dualErr.WithContext("File", path)
			dualErr = dualErr.WithCause(err)

			// Try to provide specific guidance based on the error
			switch {
			case strings.Contains(errStr, "!!map") && strings.Contains(errStr, "string"):
				dualErr = dualErr.WithFixes(
					"A value is expected to be a string but found a map/object",
					"Check that you haven't accidentally nested configuration",
				)
			case strings.Contains(errStr, "!!seq"):
				dualErr = dualErr.WithFixes(
					"A value is expected to be a single item but found an array",
					"Remove the hyphen (-) if you meant to provide a single value",
				)
			default:
				dualErr = dualErr.WithFixes(
					"Check the data types in your configuration:",
					"• version: must be a number (1)",
					"• services: must be a map of service configurations",
					"• hooks: must be arrays of script names",
					"• worktrees.path: must be a string path",
				)
			}
			return nil, dualErr
		}

		// Generic YAML error
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}

// validateConfig checks that the config has valid structure and values
func validateConfig(config *Config, projectRoot string) error {
	// Check version
	if config.Version == 0 {
		err := dualerrors.New(dualerrors.ErrConfigInvalid, "Missing required 'version' field in configuration")
		err = err.WithContext("File", filepath.Join(projectRoot, ConfigFileName))
		err = err.WithFixes(
			fmt.Sprintf("Add 'version: %d' at the top of your %s file", SupportedVersion, ConfigFileName),
			"",
			"Example configuration:",
			fmt.Sprintf("  version: %d", SupportedVersion),
			"  services:",
			"    web:",
			"      path: ./web",
		)
		return err
	}
	if config.Version != SupportedVersion {
		err := dualerrors.New(dualerrors.ErrConfigInvalid, fmt.Sprintf("Unsupported config version %d", config.Version))
		err = err.WithContext("Current version", fmt.Sprintf("%d", config.Version))
		err = err.WithContext("Required version", fmt.Sprintf("%d", SupportedVersion))
		err = err.WithFixes(
			fmt.Sprintf("Update the version field to %d", SupportedVersion),
			"This version of dual only supports config version 1",
			"Check if you need to update dual: dual --version",
		)
		return err
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
		err := dualerrors.New(dualerrors.ErrConfigInvalid, fmt.Sprintf("Service '%s' missing required 'path' field", name))
		err = err.WithContext("Service", name)
		err = err.WithFixes(
			"Add a path field to the service configuration:",
			"",
			"  services:",
			fmt.Sprintf("    %s:", name),
			fmt.Sprintf("      path: ./path/to/%s", name),
		)
		return err
	}

	// Check if path is absolute (it shouldn't be - should be relative to project root)
	if filepath.IsAbs(service.Path) {
		err := dualerrors.New(dualerrors.ErrConfigInvalid, "Service path must be relative to project root")
		err = err.WithContext("Service", name)
		err = err.WithContext("Absolute path", service.Path)
		err = err.WithContext("Project root", projectRoot)
		err = err.WithFixes(
			"Convert to a relative path from the project root:",
			fmt.Sprintf("  Instead of: %s", service.Path),
			fmt.Sprintf("  Use: ./%s", filepath.Base(service.Path)),
			"",
			"Paths should be relative to where dual.config.yml is located",
		)
		return err
	}

	// Validate that the path exists
	fullPath := filepath.Join(projectRoot, service.Path)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			dualErr := dualerrors.New(dualerrors.ErrConfigInvalid, "Service path does not exist")
			dualErr = dualErr.WithContext("Service", name)
			dualErr = dualErr.WithContext("Configured path", service.Path)
			dualErr = dualErr.WithContext("Resolved to", fullPath)
			dualErr = dualErr.WithContext("Config location", filepath.Join(projectRoot, ConfigFileName))
			dualErr = dualErr.WithFixes(
				fmt.Sprintf("Create the directory: mkdir -p %s", fullPath),
				fmt.Sprintf("Or update the path in %s", ConfigFileName),
				"",
				"Note: Paths are relative to the config file location",
			)
			return dualErr
		}
		return fmt.Errorf("failed to check path: %w", err)
	}

	// Path should be a directory
	if !info.IsDir() {
		dualErr := dualerrors.New(dualerrors.ErrConfigInvalid, "Service path must be a directory, not a file")
		dualErr = dualErr.WithContext("Service", name)
		dualErr = dualErr.WithContext("Path", service.Path)
		dualErr = dualErr.WithContext("Resolved to", fullPath)
		dualErr = dualErr.WithContext("File type", "regular file")
		dualErr = dualErr.WithFixes(
			"Service paths must point to directories containing your service code",
			fmt.Sprintf("Check if you meant the parent directory: %s", filepath.Dir(service.Path)),
			fmt.Sprintf("Or create a directory at: mkdir -p %s", fullPath+"_dir"),
		)
		return dualErr
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
		"preWorktreeDelete":  true,
		"postWorktreeDelete": true,
	}

	for event, scripts := range hooks {
		if !validEvents[event] {
			return fmt.Errorf("invalid hook event: %s (valid events: postWorktreeCreate, preWorktreeDelete, postWorktreeDelete)", event)
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
