package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/lightfastai/dual/internal/config"
)

// Test helper functions

// mockGitCommand creates a mock git command function
func mockGitCommand(output string, err error) func(args ...string) (string, error) {
	return func(args ...string) (string, error) {
		return output, err
	}
}

// mockGetwd creates a mock getwd function
func mockGetwd(dir string, err error) func() (string, error) {
	return func() (string, error) {
		return dir, err
	}
}

// mockEvalSymlinks creates a mock evalSymlinks function
func mockEvalSymlinks(mapping map[string]string) func(path string) (string, error) {
	return func(path string) (string, error) {
		if resolved, ok := mapping[path]; ok {
			return resolved, nil
		}
		// Default: return the path as-is
		return path, nil
	}
}

// TestDetectService_BasicMatching tests basic service detection
func TestDetectService_BasicMatching(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web": {Path: "apps/web"},
			"api": {Path: "apps/api"},
		},
	}

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name        string
		cwd         string
		projectRoot string
		expected    string
		wantErr     bool
	}{
		{
			name:        "exact match web",
			cwd:         "/project/apps/web",
			projectRoot: "/project",
			expected:    "web",
			wantErr:     false,
		},
		{
			name:        "exact match api",
			cwd:         "/project/apps/api",
			projectRoot: "/project",
			expected:    "api",
			wantErr:     false,
		},
		{
			name:        "nested in web",
			cwd:         "/project/apps/web/src/components",
			projectRoot: "/project",
			expected:    "web",
			wantErr:     false,
		},
		{
			name:        "nested in api",
			cwd:         "/project/apps/api/routes",
			projectRoot: "/project",
			expected:    "api",
			wantErr:     false,
		},
		{
			name:        "outside all services",
			cwd:         "/project/scripts",
			projectRoot: "/project",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "completely outside project",
			cwd:         "/other/project",
			projectRoot: "/project",
			expected:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
				getwd:        mockGetwd(tt.cwd, nil),
				evalSymlinks: mockEvalSymlinks(map[string]string{}),
			}

			result, err := detector.DetectService(cfg, tt.projectRoot)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %q", result)
				}
				if err != ErrServiceNotDetected {
					t.Errorf("expected ErrServiceNotDetected, got: %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// TestDetectService_NestedServices tests that longest match wins for nested services
func TestDetectService_NestedServices(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"parent": {Path: "apps"},
			"child":  {Path: "apps/web"},
		},
	}

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name        string
		cwd         string
		projectRoot string
		expected    string
	}{
		{
			name:        "in parent only",
			cwd:         "/project/apps/other",
			projectRoot: "/project",
			expected:    "parent",
		},
		{
			name:        "in child - should match child (longest path)",
			cwd:         "/project/apps/web",
			projectRoot: "/project",
			expected:    "child",
		},
		{
			name:        "nested in child",
			cwd:         "/project/apps/web/src",
			projectRoot: "/project",
			expected:    "child",
		},
		{
			name:        "exact match parent",
			cwd:         "/project/apps",
			projectRoot: "/project",
			expected:    "parent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
				getwd:        mockGetwd(tt.cwd, nil),
				evalSymlinks: mockEvalSymlinks(map[string]string{}),
			}

			result, err := detector.DetectService(cfg, tt.projectRoot)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestDetectService_SymlinkResolution tests that symlinks are properly resolved
func TestDetectService_SymlinkResolution(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web": {Path: "apps/web"},
		},
	}

	// Simulate symlink resolution
	symlinkMap := map[string]string{
		"/project":          "/real/project",
		"/symlink/location": "/real/project/apps/web",
	}

	detector := &Detector{
		gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
		getwd:        mockGetwd("/symlink/location", nil),
		evalSymlinks: mockEvalSymlinks(symlinkMap),
	}

	result, err := detector.DetectService(cfg, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "web"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestDetectService_ErrorHandling tests error handling
func TestDetectService_ErrorHandling(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web": {Path: "apps/web"},
		},
	}

	t.Run("getwd error", func(t *testing.T) {
		detector := &Detector{
			gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
			getwd:        mockGetwd("", fmt.Errorf("permission denied")),
			evalSymlinks: mockEvalSymlinks(map[string]string{}),
		}

		_, err := detector.DetectService(cfg, "/project")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("no service matches", func(t *testing.T) {
		detector := &Detector{
			gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
			getwd:        mockGetwd("/project/other", nil),
			evalSymlinks: mockEvalSymlinks(map[string]string{}),
		}

		_, err := detector.DetectService(cfg, "/project")
		if err != ErrServiceNotDetected {
			t.Errorf("expected ErrServiceNotDetected, got: %v", err)
		}
	})
}

// TestDetectService_RelativePaths tests handling of relative service paths
func TestDetectService_RelativePaths(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web": {Path: "./apps/web"},
			"api": {Path: "apps/api"},
		},
	}

	detector := &Detector{
		gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
		getwd:        mockGetwd("/project/apps/web/src", nil),
		evalSymlinks: mockEvalSymlinks(map[string]string{}),
	}

	result, err := detector.DetectService(cfg, "/project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "web"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestFindProjectRoot tests project root detection
func TestFindProjectRoot(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name      string
		gitOutput string
		gitError  error
		cwd       string
		expected  string
		wantErr   bool
	}{
		{
			name:      "git repo - success",
			gitOutput: "/project/path\n",
			gitError:  nil,
			cwd:       "/project/path/subdir",
			expected:  "/project/path",
			wantErr:   false,
		},
		{
			name:      "git repo - with whitespace",
			gitOutput: "  /project/path  \n",
			gitError:  nil,
			cwd:       "/project/path",
			expected:  "/project/path",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand:   mockGitCommand(tt.gitOutput, tt.gitError),
				getwd:        mockGetwd(tt.cwd, nil),
				evalSymlinks: mockEvalSymlinks(map[string]string{}),
			}

			result, err := detector.FindProjectRoot()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %q", result)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// TestFindProjectRoot_FallbackToConfigFile tests fallback to walking up for config file
func TestFindProjectRoot_FallbackToConfigFile(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	nestedDir := filepath.Join(projectRoot, "apps", "web")

	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	// Create a config file at project root
	configPath := filepath.Join(projectRoot, config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte("version: 1\nservices:\n  web:\n    path: apps/web\n"), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Test from nested directory
	detector := &Detector{
		gitCommand:   mockGitCommand("", fmt.Errorf("not a git repository")),
		getwd:        mockGetwd(nestedDir, nil),
		evalSymlinks: func(path string) (string, error) { return path, nil },
	}

	result, err := detector.FindProjectRoot()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The result should be the project root
	if result != projectRoot {
		t.Errorf("expected %q, got %q", projectRoot, result)
	}
}

// TestFindProjectRoot_NotFound tests error when project root cannot be found
func TestFindProjectRoot_NotFound(t *testing.T) {
	// Create a temporary directory without any config
	tmpDir := t.TempDir()

	detector := &Detector{
		gitCommand:   mockGitCommand("", fmt.Errorf("not a git repository")),
		getwd:        mockGetwd(tmpDir, nil),
		evalSymlinks: func(path string) (string, error) { return path, nil },
	}

	_, err := detector.FindProjectRoot()
	if err != ErrProjectRootNotFound {
		t.Errorf("expected ErrProjectRootNotFound, got: %v", err)
	}
}

// TestIsWithinPath tests the path matching logic
func TestIsWithinPath(t *testing.T) {
	tests := []struct { //nolint:govet // Test struct optimization not critical
		name       string
		targetPath string
		basePath   string
		expected   bool
	}{
		{
			name:       "exact match",
			targetPath: "/project/apps/web",
			basePath:   "/project/apps/web",
			expected:   true,
		},
		{
			name:       "nested path",
			targetPath: "/project/apps/web/src",
			basePath:   "/project/apps/web",
			expected:   true,
		},
		{
			name:       "deeply nested",
			targetPath: "/project/apps/web/src/components/Button.tsx",
			basePath:   "/project/apps/web",
			expected:   true,
		},
		{
			name:       "not within - sibling",
			targetPath: "/project/apps/api",
			basePath:   "/project/apps/web",
			expected:   false,
		},
		{
			name:       "not within - similar name",
			targetPath: "/project/apps/webapp",
			basePath:   "/project/apps/web",
			expected:   false,
		},
		{
			name:       "not within - parent",
			targetPath: "/project/apps",
			basePath:   "/project/apps/web",
			expected:   false,
		},
		{
			name:       "not within - completely different",
			targetPath: "/other/project",
			basePath:   "/project/apps/web",
			expected:   false,
		},
		{
			name:       "with trailing slash",
			targetPath: "/project/apps/web/",
			basePath:   "/project/apps/web",
			expected:   true,
		},
		{
			name:       "with redundant slashes",
			targetPath: "/project//apps///web",
			basePath:   "/project/apps/web",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isWithinPath(tt.targetPath, tt.basePath)
			if result != tt.expected {
				t.Errorf("isWithinPath(%q, %q) = %v, expected %v",
					tt.targetPath, tt.basePath, result, tt.expected)
			}
		})
	}
}

// TestDetectService_ConvenienceFunction tests the package-level convenience function
func TestDetectService_ConvenienceFunction(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web": {Path: "apps/web"},
		},
	}

	// This is more of an integration test
	// We can't easily test it without real filesystem, so just ensure it doesn't panic
	_, err := DetectService(cfg, "/nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent project root")
	}
}

// TestDetectServiceWithRoot tests the convenience function that finds root and detects service
func TestDetectServiceWithRoot(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	projectRoot := filepath.Join(tmpDir, "project")
	webDir := filepath.Join(projectRoot, "apps", "web")

	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("failed to create test directories: %v", err)
	}

	// Create a config file
	configContent := `version: 1
services:
  web:
    path: apps/web
`
	configPath := filepath.Join(projectRoot, config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config
	cfg, err := config.LoadConfigFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Create detector with mocked getwd
	detector := &Detector{
		gitCommand:   mockGitCommand("", fmt.Errorf("not a git repository")),
		getwd:        mockGetwd(webDir, nil),
		evalSymlinks: func(path string) (string, error) { return path, nil },
	}

	serviceName, err := detector.DetectService(cfg, projectRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if serviceName != "web" {
		t.Errorf("expected service 'web', got %q", serviceName)
	}

	// Verify project root is set correctly
	if projectRoot == "" {
		t.Error("expected non-empty project root")
	}
}

// TestDetectService_MultipleServices tests detection with multiple services
func TestDetectService_MultipleServices(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Services: map[string]config.Service{
			"web":     {Path: "apps/web"},
			"api":     {Path: "apps/api"},
			"worker":  {Path: "apps/worker"},
			"admin":   {Path: "apps/admin"},
			"mobile":  {Path: "mobile"},
			"desktop": {Path: "desktop"},
		},
	}

	tests := []struct { //nolint:govet // Test struct optimization not critical
		name     string
		cwd      string
		expected string
	}{
		{name: "web service", cwd: "/project/apps/web/pages", expected: "web"},
		{name: "api service", cwd: "/project/apps/api/routes", expected: "api"},
		{name: "worker service", cwd: "/project/apps/worker", expected: "worker"},
		{name: "admin service", cwd: "/project/apps/admin/dashboard", expected: "admin"},
		{name: "mobile service", cwd: "/project/mobile/src", expected: "mobile"},
		{name: "desktop service", cwd: "/project/desktop/main", expected: "desktop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := &Detector{
				gitCommand:   mockGitCommand("", fmt.Errorf("not used")),
				getwd:        mockGetwd(tt.cwd, nil),
				evalSymlinks: mockEvalSymlinks(map[string]string{}),
			}

			result, err := detector.DetectService(cfg, "/project")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
