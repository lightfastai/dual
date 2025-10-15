package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseConfig(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid config",
			content: `version: 1
services:
  web:
    path: ./apps/web
    envFile: .env.local
  api:
    path: ./apps/api
`,
			wantErr: false,
		},
		{
			name: "invalid YAML",
			content: `version: 1
services:
  web:
    path: ./apps/web
  - invalid
`,
			wantErr: true,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: false, // YAML parsing succeeds, validation will catch empty config
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "dual.config.yml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			_, err := parseConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	// Create a temp directory structure for testing
	tmpDir := t.TempDir()
	webDir := filepath.Join(tmpDir, "apps", "web")
	apiDir := filepath.Join(tmpDir, "apps", "api")
	envDir := filepath.Join(tmpDir, "apps", "web")

	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(envDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Version: 1,
				Services: map[string]Service{
					"web": {
						Path:    "apps/web",
						EnvFile: "apps/web/.env.local",
					},
					"api": {
						Path: "apps/api",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: &Config{
				Version: 0,
				Services: map[string]Service{
					"web": {Path: "apps/web"},
				},
			},
			wantErr: true,
			errMsg:  "Missing required 'version' field",
		},
		{
			name: "unsupported version",
			config: &Config{
				Version: 2,
				Services: map[string]Service{
					"web": {Path: "apps/web"},
				},
			},
			wantErr: true,
			errMsg:  "Unsupported config version",
		},
		{
			name: "no services (allowed for init)",
			config: &Config{
				Version:  1,
				Services: map[string]Service{},
			},
			wantErr: false,
		},
		{
			name: "nil services (allowed for init)",
			config: &Config{
				Version:  1,
				Services: nil,
			},
			wantErr: false,
		},
		{
			name: "service with absolute path",
			config: &Config{
				Version: 1,
				Services: map[string]Service{
					"web": {Path: "/absolute/path"},
				},
			},
			wantErr: true,
			errMsg:  "path must be relative",
		},
		{
			name: "service with non-existent path",
			config: &Config{
				Version: 1,
				Services: map[string]Service{
					"web": {Path: "nonexistent"},
				},
			},
			wantErr: true,
			errMsg:  "path does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateConfig() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	// Create test directory structure
	tmpDir := t.TempDir()
	validDir := filepath.Join(tmpDir, "valid")
	envFileDir := filepath.Join(tmpDir, "with-env")
	testFile := filepath.Join(tmpDir, "file.txt")

	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.MkdirAll(envFileDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name    string
		service Service
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service",
			service: Service{
				Path: "valid",
			},
			wantErr: false,
		},
		{
			name: "valid service with envFile",
			service: Service{
				Path:    "with-env",
				EnvFile: "with-env/.env.local",
			},
			wantErr: false,
		},
		{
			name: "empty path",
			service: Service{
				Path: "",
			},
			wantErr: true,
			errMsg:  "missing required 'path' field",
		},
		{
			name: "absolute path",
			service: Service{
				Path: "/absolute/path",
			},
			wantErr: true,
			errMsg:  "path must be relative",
		},
		{
			name: "path to file instead of directory",
			service: Service{
				Path: "file.txt",
			},
			wantErr: true,
			errMsg:  "path must be a directory",
		},
		{
			name: "non-existent path",
			service: Service{
				Path: "nonexistent",
			},
			wantErr: true,
			errMsg:  "path does not exist",
		},
		{
			name: "absolute envFile",
			service: Service{
				Path:    "valid",
				EnvFile: "/absolute/.env",
			},
			wantErr: true,
			errMsg:  "envFile must be relative",
		},
		{
			name: "envFile with non-existent directory",
			service: Service{
				Path:    "valid",
				EnvFile: "nonexistent/.env",
			},
			wantErr: true,
			errMsg:  "envFile directory does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateService("test", tt.service, tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("validateService() error = %v, want error containing %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	nestedDir := filepath.Join(projectRoot, "apps", "web")

	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Create a valid config file at project root
	configContent := `version: 1
services:
  web:
    path: apps/web
`
	configPath := filepath.Join(projectRoot, "dual.config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Test loading from project root
	t.Run("load from project root", func(t *testing.T) {
		// Change to project root
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		if err := os.Chdir(projectRoot); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		config, root, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Version != 1 {
			t.Errorf("config.Version = %d, want 1", config.Version)
		}

		if len(config.Services) != 1 {
			t.Errorf("len(config.Services) = %d, want 1", len(config.Services))
		}

		// Resolve both paths to handle symlinks (like /var -> /private/var on macOS)
		expectedRoot, _ := filepath.EvalSymlinks(projectRoot)
		actualRoot, _ := filepath.EvalSymlinks(root)
		if actualRoot != expectedRoot {
			t.Errorf("root = %s, want %s", actualRoot, expectedRoot)
		}
	})

	// Test loading from nested directory
	t.Run("load from nested directory", func(t *testing.T) {
		// Change to nested directory
		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		if err := os.Chdir(nestedDir); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		config, root, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() error = %v", err)
		}

		if config.Version != 1 {
			t.Errorf("config.Version = %d, want 1", config.Version)
		}

		// Resolve both paths to handle symlinks (like /var -> /private/var on macOS)
		expectedRoot, _ := filepath.EvalSymlinks(projectRoot)
		actualRoot, _ := filepath.EvalSymlinks(root)
		if actualRoot != expectedRoot {
			t.Errorf("root = %s, want %s", actualRoot, expectedRoot)
		}
	})

	// Test loading from directory without config
	t.Run("no config found", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		if err := os.MkdirAll(emptyDir, 0o755); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		oldWd, _ := os.Getwd()
		defer os.Chdir(oldWd)

		if err := os.Chdir(emptyDir); err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}

		_, _, err := LoadConfig()
		if err == nil {
			t.Error("LoadConfig() expected error, got nil")
		}
		if !contains(err.Error(), "No dual.config.yml found") {
			t.Errorf("LoadConfig() error = %v, want error containing 'No dual.config.yml found'", err)
		}
	})
}

func TestLoadConfigFrom(t *testing.T) {
	tmpDir := t.TempDir()
	webDir := filepath.Join(tmpDir, "apps", "web")

	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name    string
		content string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			content: `version: 1
services:
  web:
    path: apps/web
`,
			wantErr: false,
		},
		{
			name: "invalid version",
			content: `version: 2
services:
  web:
    path: apps/web
`,
			wantErr: true,
			errMsg:  "Unsupported config version",
		},
		{
			name: "missing services (allowed for init)",
			content: `version: 1
services: {}
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, "dual.config.yml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("failed to write test config: %v", err)
			}

			config, err := LoadConfigFrom(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfigFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("LoadConfigFrom() error = %v, want error containing %q", err, tt.errMsg)
				}
			}

			if !tt.wantErr && config == nil {
				t.Error("LoadConfigFrom() returned nil config without error")
			}
		})
	}
}

func TestLoadConfigFrom_NonExistentFile(t *testing.T) {
	_, err := LoadConfigFrom("/nonexistent/path/dual.config.yml")
	if err == nil {
		t.Error("LoadConfigFrom() expected error for non-existent file, got nil")
	}
}

func TestConfigConstants(t *testing.T) {
	if ConfigFileName != "dual.config.yml" {
		t.Errorf("ConfigFileName = %q, want %q", ConfigFileName, "dual.config.yml")
	}

	if SupportedVersion != 1 {
		t.Errorf("SupportedVersion = %d, want 1", SupportedVersion)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
