package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	Services map[string]Service `yaml:"services"`
	Version  int                `yaml:"version"`
	Env      EnvConfig          `yaml:"env,omitempty"`
}

// EnvConfig contains environment-related configuration
type EnvConfig struct {
	// BaseFile is the path to the base environment file (relative to project root)
	BaseFile string `yaml:"baseFile,omitempty"`
}

// Service represents a single service configuration
type Service struct {
	Path    string `yaml:"path"`
	EnvFile string `yaml:"envFile"`
}

// LoadConfig searches for dual.config.yml starting from the current directory
// and walking up the directory tree until it finds the file or reaches the root.
// It returns the parsed config and the absolute path of the project root (where the config was found).
func LoadConfig() (*Config, string, error) {
	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	searchDir := currentDir
	for {
		configPath := filepath.Join(searchDir, ConfigFileName)

		// Check if config file exists
		if _, err := os.Stat(configPath); err == nil {
			// Found the config file, parse it
			config, err := parseConfig(configPath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to parse %s: %w", configPath, err)
			}

			// Validate the config
			if err := validateConfig(config, searchDir); err != nil {
				return nil, "", fmt.Errorf("invalid config in %s: %w", configPath, err)
			}

			return config, searchDir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(searchDir)

		// Check if we've reached the root
		if parentDir == searchDir {
			return nil, "", fmt.Errorf("no %s found in current directory or any parent directory", ConfigFileName)
		}

		searchDir = parentDir
	}
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
