package integration

import (
	"os"
	"strings"
	"testing"
)

// TestFullWorkflow tests the complete workflow: init → service add → context create → run
func TestFullWorkflow(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repository
	h.InitGitRepo()

	// Step 1: dual init
	t.Log("Step 1: Running dual init")
	stdout, stderr, exitCode := h.RunDual("init")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Initialized configuration")

	// Verify config file was created
	if !h.FileExists("dual.config.yml") {
		t.Fatal("dual.config.yml was not created")
	}

	h.AssertFileContains("dual.config.yml", "version: 1")
	h.AssertFileContains("dual.config.yml", "services: {}")

	// Step 2: Add services
	t.Log("Step 2: Adding services")

	// Create service directories
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.CreateDirectory("apps/worker")

	// Add web service
	stdout, stderr, exitCode = h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"web\"")

	// Add api service
	stdout, stderr, exitCode = h.RunDual("service", "add", "api", "--path", "apps/api")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"api\"")

	// Add worker service
	stdout, stderr, exitCode = h.RunDual("service", "add", "worker", "--path", "apps/worker")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"worker\"")

	// Verify services were added to config
	h.AssertFileContains("dual.config.yml", "web:")
	h.AssertFileContains("dual.config.yml", "apps/web")
	h.AssertFileContains("dual.config.yml", "api:")
	h.AssertFileContains("dual.config.yml", "apps/api")
	h.AssertFileContains("dual.config.yml", "worker:")
	h.AssertFileContains("dual.config.yml", "apps/worker")

	// Add worktrees configuration to enable dual create
	h.WriteFile("dual.config.yml", `version: 1
services:
  web:
    path: apps/web
  api:
    path: apps/api
  worker:
    path: apps/worker
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Step 3: Create context (using a feature branch name to avoid conflicts)
	t.Log("Step 3: Creating context")
	stdout, stderr, exitCode = h.RunDual("create", "feature-x")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Worktree created successfully")

	// Verify registry was created
	if !h.RegistryExists() {
		t.Fatal("registry.json was not created")
	}

	// Step 4: Verify context exists in list
	t.Log("Step 4: Verifying context in list")
	stdout, stderr, exitCode = h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "feature-x")
}

// TestFullWorkflowWithEnvFile tests the workflow with env file configuration
func TestFullWorkflowWithEnvFile(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repository
	h.InitGitRepo()
	h.CreateGitBranch("main")

	// Initialize dual
	h.RunDual("init")

	// Create service directory with env file directory
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/web/.env")

	// Add service with env-file
	stdout, stderr, exitCode := h.RunDual(
		"service", "add", "web",
		"--path", "apps/web",
		"--env-file", "apps/web/.env/development.local",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"web\"")
	h.AssertOutputContains(stdout, "Env File: apps/web/.env/development.local")

	// Verify env file path in config
	h.AssertFileContains("dual.config.yml", "envFile: apps/web/.env/development.local")
}

// TestInitForceFlag tests the --force flag for dual init
func TestInitForceFlag(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// First init
	stdout, stderr, exitCode := h.RunDual("init")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Modify the config
	h.WriteFile("dual.config.yml", "version: 1\nservices:\n  test:\n    path: apps/test\n")

	// Try to init again without --force (should fail)
	stdout, stderr, exitCode = h.RunDual("init")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "configuration file already exists")

	// Init with --force (should succeed)
	stdout, stderr, exitCode = h.RunDual("init", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Overwriting existing configuration")

	// Verify config was reset
	configContent := h.ReadFile("dual.config.yml")
	if strings.Contains(configContent, "test:") {
		t.Error("config was not reset by --force flag")
	}
}

// TestContextAutoDetection tests automatic context name detection
func TestContextAutoDetection(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// Initialize dual with worktrees config
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create a feature branch worktree - dual create will create both the branch and worktree
	stdout, stderr, exitCode := h.RunDual("create", "feature/awesome-feature")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Worktree created successfully")

	// Verify the worktree was created and appears in the list
	stdout, stderr, exitCode = h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "feature/awesome-feature")
	// The test validates that worktrees can be created with branch names containing slashes
}

// TestContextAutoPortAssignment tests automatic port assignment
// REMOVED: This test was specific to port assignment functionality which has been removed.
// The worktree lifecycle manager no longer manages ports.

// makeExecutable makes a file executable
func makeExecutable(path string) error {
	return os.Chmod(path, 0o755)
}
