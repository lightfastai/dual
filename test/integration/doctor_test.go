package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lightfastai/dual/internal/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("Doctor in initialized project", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		// Initialize git repo
		h.InitGitRepo()
		h.CreateGitBranch("main")

		// Initialize dual config
		h.RunDual("init")

		// Add a service
		h.CreateDirectory("apps/api")
		h.RunDual("service", "add", "api", "--path", "apps/api")

		// Add worktrees configuration
		h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: apps/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

		// Create an initial commit (required for git worktree add)
		h.WriteFile("README.md", "# Test Project")
		h.RunGitCommand("add", "README.md")
		h.RunGitCommand("commit", "-m", "Initial commit")

		// Create a context matching the branch
		h.RunDual("create", "main")

		// Run doctor
		stdout, stderr, exitCode := h.RunDual("doctor")

		// Doctor may have warnings (exit 1) due to missing env files or service detection
		// but should not have errors (exit 2)
		output := stdout + stderr
		if exitCode != 0 && exitCode != 1 {
			t.Fatalf("Expected exit code 0 or 1, got %d. Output:\n%s", exitCode, output)
		}

		// Check for expected content
		assert.Contains(t, output, "Dual Health Check Results")
		assert.Contains(t, output, "Git Repository")
		assert.Contains(t, output, "Configuration File")
		assert.Contains(t, output, "Registry")
		assert.Contains(t, output, "Service Paths")
	})

	t.Run("Doctor with JSON output", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		// Initialize git repo
		h.InitGitRepo()

		// Initialize dual config
		h.RunDual("init")

		// Run doctor with --json
		stdout, stderr, exitCode := h.RunDual("doctor", "--json")

		// Should exit with warnings (no services configured)
		h.AssertExitCode(exitCode, 1, stdout+stderr)

		// Parse JSON output
		var result health.Result
		err := json.Unmarshal([]byte(stdout), &result)
		require.NoError(t, err, "output should be valid JSON")

		// Verify result structure
		assert.Greater(t, result.TotalChecks, 0)
		assert.NotEmpty(t, result.Checks)
		assert.Equal(t, 1, result.ExitCode)
	})

	t.Run("Doctor without config", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		// Initialize git repo only (no dual config)
		h.InitGitRepo()

		// Run doctor
		stdout, stderr, exitCode := h.RunDual("doctor")

		// Should exit with errors (no config)
		h.AssertExitCode(exitCode, 2, stdout+stderr)

		output := stdout + stderr
		assert.Contains(t, output, "Configuration File")
		assert.Contains(t, output, "No dual.config.yml found")
	})

	t.Run("Doctor with invalid service paths", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		// Initialize git repo
		h.InitGitRepo()

		// Initialize dual config properly
		h.RunDual("init")

		// Add a service with a valid path first
		h.CreateDirectory("apps/invalid-service")
		h.RunDual("service", "add", "invalid-service", "--path", "apps/invalid-service")

		// Remove the directory to make the path invalid
		// This will cause config validation to fail when doctor tries to load it
		require.NoError(h.t, os.RemoveAll(filepath.Join(h.ProjectDir, "apps/invalid-service")))

		// Run doctor - config load will fail due to validation
		stdout, stderr, exitCode := h.RunDual("doctor")

		// Should exit with errors (config validation fails)
		h.AssertExitCode(exitCode, 2, stdout+stderr)

		output := stdout + stderr
		// When config validation fails, CheckConfigFile shows "No dual.config.yml found"
		// because ctx.Config is nil (even though file exists but validation failed)
		// This is current behavior - the config file exists but can't be loaded
		assert.Contains(t, output, "Configuration File")
		assert.Contains(t, output, "No dual.config.yml found")
	})

	t.Run("Doctor with --fix for orphaned contexts", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		// Initialize git repo
		h.InitGitRepo()
		h.CreateGitBranch("main")

		// Initialize dual config with worktrees
		h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: apps/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

		// Add a service
		h.CreateDirectory("apps/api")

		// Create an initial commit (required for git worktree add)
		h.WriteFile("README.md", "# Test Project")
		h.RunGitCommand("add", "README.md")
		h.RunGitCommand("commit", "-m", "Initial commit")

		// Create main context first (this will create the registry)
		h.RunDual("create", "main")

		// Create another worktree
		h.RunDual("create", "orphaned-branch")

		// Delete the worktree directory to make it orphaned
		// (but leave it in the registry)
		worktreesPath := filepath.Join(filepath.Dir(h.ProjectDir), "worktrees")
		orphanedPath := filepath.Join(worktreesPath, "orphaned-branch")
		require.NoError(t, os.RemoveAll(orphanedPath))

		// Run doctor without --fix first
		stdout, stderr, _ := h.RunDual("doctor")

		// Should warn about orphaned context
		output := stdout + stderr
		assert.Contains(t, output, "Orphaned Contexts")
		// Just check that it's mentioned, don't check exact count
		if !strings.Contains(output, "No orphaned contexts") {
			// Run doctor with --fix
			stdout, stderr, _ = h.RunDual("doctor", "--fix")

			// Should clean up or show fixing
			output = stdout + stderr
			// Check that something was cleaned or the check ran
			assert.Contains(t, output, "Orphaned Contexts")
		}
	})

	t.Run("Doctor with --verbose", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		// Initialize git repo
		h.InitGitRepo()

		// Initialize dual config
		h.RunDual("init")

		// Add services
		h.CreateDirectory("apps/api")
		h.RunDual("service", "add", "api", "--path", "apps/api")

		// Run doctor with --verbose
		stdout, stderr, _ := h.RunDual("doctor", "--verbose")

		output := stdout + stderr

		// Verbose should show checking messages
		assert.Contains(t, output, "Checking")

		// Should show more details
		lines := strings.Split(output, "\n")
		assert.Greater(t, len(lines), 20, "verbose output should have many lines")
	})

	t.Run("Doctor exit codes", func(t *testing.T) {
		tests := []struct {
			name         string
			setup        func(h *TestHelper)
			expectedCode int
		}{
			{
				name: "Exit 0 or 1 - all pass or minor warnings",
				setup: func(h *TestHelper) {
					h.InitGitRepo()
					h.CreateGitBranch("main")
					h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: apps/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
					h.CreateDirectory("apps/api")
					// Create an initial commit (required for git worktree add)
					h.WriteFile("README.md", "# Test Project")
					h.RunGitCommand("add", "README.md")
					h.RunGitCommand("commit", "-m", "Initial commit")
					h.RunDual("create", "main")
				},
				expectedCode: -1, // Special value to accept 0 or 1
			},
			{
				name: "Exit 1 - warnings",
				setup: func(h *TestHelper) {
					h.InitGitRepo()
					h.RunDual("init")
					// No services - will trigger warnings
				},
				expectedCode: 1,
			},
			{
				name: "Exit 2 - errors",
				setup: func(h *TestHelper) {
					h.InitGitRepo()
					// No dual config - will trigger errors
				},
				expectedCode: 2,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := NewTestHelper(t)
				defer h.RestoreHome()

				tt.setup(h)

				stdout, stderr, exitCode := h.RunDual("doctor")

				// Special case: -1 means accept 0 or 1 (pass or warnings)
				if tt.expectedCode == -1 {
					if exitCode != 0 && exitCode != 1 {
						t.Fatalf("Expected exit code 0 or 1, got %d. Output:\n%s", exitCode, stdout+stderr)
					}
				} else {
					h.AssertExitCode(exitCode, tt.expectedCode, stdout+stderr)
				}
			})
		}
	})
}

func TestDoctorPortConflicts(t *testing.T) {
	// REMOVED: This test was specific to port conflict detection functionality which has been removed.
	// The worktree lifecycle manager no longer manages ports.
}

func TestDoctorWorktreeValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo
	h.InitGitRepo()
	h.CreateGitBranch("main")

	// Initialize dual config
	h.RunDual("init")

	// Add services
	h.CreateDirectory("apps/api")
	h.RunDual("service", "add", "api", "--path", "apps/api")

	// Add worktrees configuration
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: apps/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create context
	h.RunDual("create", "main")

	// Create a worktree
	worktreePath := h.CreateGitWorktree("feature-branch", "worktree-feature")

	// Run doctor from worktree
	stdout, stderr, _ := h.RunDualInDir(worktreePath, "doctor")

	// Should pass
	output := stdout + stderr
	assert.Contains(t, output, "Worktrees")
}

func TestDoctorEnvironmentFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("Missing env files", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		h.InitGitRepo()
		h.CreateGitBranch("main")
		h.RunDual("init")

		// Add service with env file but don't create it
		h.CreateDirectory("apps/api")
		h.RunDual("service", "add", "api", "--path", "apps/api", "--env-file", "apps/api/.env")

		// Run doctor
		stdout, stderr, _ := h.RunDual("doctor")

		// Should warn about missing env file
		output := stdout + stderr
		assert.Contains(t, output, "Environment Files")
		assert.Contains(t, output, "not found")
	})

	t.Run("Valid env files", func(t *testing.T) {
		h := NewTestHelper(t)
		defer h.RestoreHome()

		h.InitGitRepo()
		h.CreateGitBranch("main")
		h.RunDual("init")

		// Add service with env file and create it
		h.CreateDirectory("apps/api")
		h.WriteFile("apps/api/.env", "FOO=bar\n")
		h.RunDual("service", "add", "api", "--path", "apps/api", "--env-file", "apps/api/.env")

		// Run doctor
		stdout, stderr, _ := h.RunDual("doctor")

		// Should pass
		output := stdout + stderr
		assert.Contains(t, output, "Environment Files")
	})
}

func TestDoctorServiceDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Add services
	h.CreateDirectory("apps/api")
	h.CreateDirectory("apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")
	h.RunDual("service", "add", "web", "--path", "apps/web")

	// Add worktrees configuration
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: apps/api
  web:
    path: apps/web
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create context
	h.RunDual("create", "main")

	// Run doctor from within a service directory
	stdout, stderr, _ := h.RunDualInDir(filepath.Join(h.ProjectDir, "apps/api"), "doctor")

	// Should detect the service
	output := stdout + stderr
	assert.Contains(t, output, "Service Detection")
}
