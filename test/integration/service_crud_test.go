package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TestServiceList tests the dual service list command
func TestServiceList(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize config
	h.WriteFile("dual.config.yml", `version: 1
services:
  www:
    path: apps/www
    envFile: .vercel/.env.development.local
  deus:
    path: apps/deus
    envFile: .vercel/.env.development.local
  auth:
    path: apps/auth
    envFile: .vercel/.env.development.local
`)

	// Create service directories
	h.CreateDirectory("apps/www")
	h.CreateDirectory("apps/deus")
	h.CreateDirectory("apps/auth")
	h.CreateDirectory(".vercel")

	t.Run("list services with default output", func(t *testing.T) {
		stdout, stderr, exitCode := h.RunDual("service", "list")
		h.AssertExitCode(exitCode, 0, stderr)
		h.AssertOutputContains(stdout, "Services in dual.config.yml:")
		h.AssertOutputContains(stdout, "auth")
		h.AssertOutputContains(stdout, "deus")
		h.AssertOutputContains(stdout, "www")
		h.AssertOutputContains(stdout, "Total: 3 services")
	})

	t.Run("list services with JSON output", func(t *testing.T) {
		stdout, stderr, exitCode := h.RunDual("service", "list", "--json")
		h.AssertExitCode(exitCode, 0, stderr)

		// Parse JSON
		var result struct {
			Services []struct {
				Name    string `json:"name"`
				Path    string `json:"path"`
				EnvFile string `json:"envFile"`
			} `json:"services"`
		}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
		}

		// Verify services are in alphabetical order
		if len(result.Services) != 3 {
			t.Fatalf("expected 3 services, got %d", len(result.Services))
		}
		if result.Services[0].Name != "auth" {
			t.Errorf("expected first service to be 'auth', got %s", result.Services[0].Name)
		}
		if result.Services[1].Name != "deus" {
			t.Errorf("expected second service to be 'deus', got %s", result.Services[1].Name)
		}
		if result.Services[2].Name != "www" {
			t.Errorf("expected third service to be 'www', got %s", result.Services[2].Name)
		}
	})

	// REMOVED: test "list services with ports" - dual no longer manages ports
	// Port management has been removed from dual. Users can implement custom
	// port logic in hooks if needed.

	t.Run("list services with absolute paths", func(t *testing.T) {
		stdout, stderr, exitCode := h.RunDual("service", "list", "--paths")
		h.AssertExitCode(exitCode, 0, stderr)
		// Should contain absolute paths
		h.AssertOutputContains(stdout, h.ProjectDir)
	})

	t.Run("list services with JSON and paths", func(t *testing.T) {
		stdout, stderr, exitCode := h.RunDual("service", "list", "--json", "--paths")
		h.AssertExitCode(exitCode, 0, stderr)

		// Parse JSON
		var result struct {
			Services []struct {
				Name         string `json:"name"`
				Path         string `json:"path"`
				AbsolutePath string `json:"absolutePath"`
			} `json:"services"`
		}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
		}

		// Verify absolute paths are present
		// Note: On macOS, /var is a symlink to /private/var, so we need to resolve both paths
		projectDirResolved, _ := filepath.EvalSymlinks(h.ProjectDir)
		if projectDirResolved == "" {
			projectDirResolved = h.ProjectDir
		}

		for _, svc := range result.Services {
			if svc.AbsolutePath == "" {
				t.Errorf("service %s missing absolutePath", svc.Name)
			}
			absPathResolved, _ := filepath.EvalSymlinks(svc.AbsolutePath)
			if absPathResolved == "" {
				absPathResolved = svc.AbsolutePath
			}

			if !strings.HasPrefix(absPathResolved, projectDirResolved) {
				t.Errorf("service %s absolutePath %s (resolved: %s) does not start with project dir %s (resolved: %s)",
					svc.Name, svc.AbsolutePath, absPathResolved, h.ProjectDir, projectDirResolved)
			}
		}
	})

	t.Run("list with no services", func(t *testing.T) {
		// Create empty config
		h.WriteFile("dual.config.yml", `version: 1
services: {}
`)

		stdout, stderr, exitCode := h.RunDual("service", "list")
		h.AssertExitCode(exitCode, 0, stderr)
		h.AssertOutputContains(stdout, "No services configured")
	})

	t.Run("list with no services JSON", func(t *testing.T) {
		stdout, stderr, exitCode := h.RunDual("service", "list", "--json")
		h.AssertExitCode(exitCode, 0, stderr)

		// Parse JSON
		var result struct {
			Services []interface{} `json:"services"`
		}
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
		}

		if len(result.Services) != 0 {
			t.Errorf("expected empty services array, got %d services", len(result.Services))
		}
	})
}

// TestServiceRemove tests the dual service remove command
func TestServiceRemove(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	t.Run("remove service with force flag", func(t *testing.T) {
		// Initialize config
		h.WriteFile("dual.config.yml", `version: 1
services:
  www:
    path: apps/www
    envFile: .vercel/.env.development.local
  deus:
    path: apps/deus
    envFile: .vercel/.env.development.local
  auth:
    path: apps/auth
    envFile: .vercel/.env.development.local
`)

		// Create service directories
		h.CreateDirectory("apps/www")
		h.CreateDirectory("apps/deus")
		h.CreateDirectory("apps/auth")
		h.CreateDirectory(".vercel")

		// Remove service with --force
		stdout, stderr, exitCode := h.RunDual("service", "remove", "deus", "--force")
		h.AssertExitCode(exitCode, 0, stderr)
		h.AssertOutputContains(stdout, "removed from config")

		// Verify service was removed from config
		config := h.ReadFile("dual.config.yml")
		h.AssertOutputNotContains(config, "deus")
		h.AssertOutputContains(config, "www")
		h.AssertOutputContains(config, "auth")

		// Verify list command shows only 2 services
		stdout, stderr, exitCode = h.RunDual("service", "list")
		h.AssertExitCode(exitCode, 0, stderr)
		h.AssertOutputContains(stdout, "Total: 2 services")
		h.AssertOutputNotContains(stdout, "deus")
	})

	// REMOVED: test "remove service shows port impact warning" - dual no longer manages ports

	t.Run("remove non-existent service fails", func(t *testing.T) {
		// Initialize config
		h.WriteFile("dual.config.yml", `version: 1
services:
  www:
    path: apps/www
    envFile: .vercel/.env.development.local
`)
		h.CreateDirectory("apps/www")

		// Try to remove non-existent service
		_, stderr, exitCode := h.RunDual("service", "remove", "nonexistent", "--force")
		h.AssertExitCode(exitCode, 1, stderr)
		h.AssertOutputContains(stderr, "not found")
	})

	t.Run("remove last service leaves empty services map", func(t *testing.T) {
		// Initialize config with one service
		h.WriteFile("dual.config.yml", `version: 1
services:
  www:
    path: apps/www
    envFile: .vercel/.env.development.local
`)
		h.CreateDirectory("apps/www")

		// Remove the last service
		stdout, stderr, exitCode := h.RunDual("service", "remove", "www", "--force")
		h.AssertExitCode(exitCode, 0, stderr)
		h.AssertOutputContains(stdout, "removed from config")

		// List should show no services
		stdout, stderr, exitCode = h.RunDual("service", "list")
		h.AssertExitCode(exitCode, 0, stderr)
		h.AssertOutputContains(stdout, "No services configured")
	})

	// REMOVED: test "remove service from middle affects subsequent ports" - dual no longer manages ports
}

// TestServiceFullCRUD tests complete CRUD operations
func TestServiceFullCRUD(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize empty config
	h.WriteFile("dual.config.yml", `version: 1
services: {}
`)

	// Add first service
	h.CreateDirectory("apps/api")
	_, stderr, exitCode := h.RunDual("service", "add", "api", "--path", "apps/api")
	h.AssertExitCode(exitCode, 0, stderr)

	// Add second service
	h.CreateDirectory("apps/web")
	_, stderr, exitCode = h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 0, stderr)

	// List services
	stdout, _, _ := h.RunDual("service", "list")
	h.AssertOutputContains(stdout, "api")
	h.AssertOutputContains(stdout, "web")
	h.AssertOutputContains(stdout, "Total: 2 services")

	// Remove first service
	_, stderr, exitCode = h.RunDual("service", "remove", "api", "--force")
	h.AssertExitCode(exitCode, 0, stderr)

	// List should show only one service
	stdout, _, _ = h.RunDual("service", "list")
	h.AssertOutputNotContains(stdout, "api")
	h.AssertOutputContains(stdout, "web")
	h.AssertOutputContains(stdout, "Total: 1 service")

	// Remove last service
	_, stderr, exitCode = h.RunDual("service", "remove", "web", "--force")
	h.AssertExitCode(exitCode, 0, stderr)

	// List should show no services
	stdout, _, _ = h.RunDual("service", "list")
	h.AssertOutputContains(stdout, "No services configured")
}
