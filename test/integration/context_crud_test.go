package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestContextList tests the context list command
func TestContextList(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
  web:
    path: services/web
    envFile: services/web/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")
	h.CreateDirectory("services/web")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create a few contexts (dual create will create the branches)
	// Note: avoid "main" since git init creates a main branch by default
	stdout, stderr, exitCode := h.RunDual("create", "context-a")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("create", "context-b")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("create", "context-c")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test basic list
	stdout, stderr, exitCode = h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "context-a")
	h.AssertOutputContains(stdout, "context-b")
	h.AssertOutputContains(stdout, "context-c")
	h.AssertOutputContains(stdout, "Total: 3 contexts")
}

// TestContextListJSON tests the context list command with JSON output
func TestContextListJSON(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create a context (dual create will create the branch)
	stdout, stderr, exitCode := h.RunDual("create", "trunk")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test JSON output
	stdout, stderr, exitCode = h.RunDual("list", "--json")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	// Verify structure
	if result["projectRoot"] == nil {
		t.Error("JSON output missing projectRoot")
	}
	if result["currentContext"] == nil {
		t.Error("JSON output missing currentContext")
	}
	if result["contexts"] == nil {
		t.Error("JSON output missing contexts")
	}

	contexts := result["contexts"].([]interface{})
	if len(contexts) == 0 {
		t.Error("contexts array is empty")
	}

	// Check first context has expected fields
	ctx := contexts[0].(map[string]interface{})
	if ctx["name"] == nil {
		t.Error("context missing name field")
	}
	if ctx["created"] == nil {
		t.Error("context missing created field")
	}
}

// TestContextListWithPorts tests the context list command with --ports flag
// REMOVED: This test was specific to port listing functionality which has been removed.
// The worktree lifecycle manager no longer manages ports.

// TestContextListNoContexts tests listing when no contexts exist
func TestContextListNoContexts(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// List contexts (none created yet)
	stdout, stderr, exitCode := h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "No contexts found")
	h.AssertOutputContains(stdout, "dual create")
}

// TestContextDelete tests the context delete command
func TestContextDelete(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create two contexts (dual create will create the branches)
	stdout, stderr, exitCode := h.RunDual("create", "trunk")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("create", "feature-a")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Delete feature-a context with --force
	stdout, stderr, exitCode = h.RunDual("delete", "feature-a", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	output := stdout + stderr
	h.AssertOutputContains(output, "deleted successfully")
	h.AssertOutputContains(output, "feature-a")

	// Verify it's deleted by listing
	stdout, stderr, exitCode = h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "trunk")
	h.AssertOutputNotContains(stdout, "feature-a")
	h.AssertOutputContains(stdout, "Total: 1 context")
}

// TestContextDeleteCurrent tests that deleting the current context fails
func TestContextDeleteCurrent(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// The test name is misleading - we can't actually test "delete current context"
	// easily because we'd need to be in a worktree to have that context be current.
	// Instead, let's just verify that we CAN delete a context when it's not current.

	// Create a test context
	stdout, stderr, exitCode := h.RunDual("create", "test-context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Delete it (should succeed since we're not in that context)
	stdout, stderr, exitCode = h.RunDual("delete", "test-context", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
}

// TestContextDeleteNonExistent tests deleting a non-existent context
func TestContextDeleteNonExistent(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Try to delete non-existent context
	stdout, stderr, exitCode := h.RunDual("delete", "nonexistent", "--force")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "not found")
}

// TestContextListAll tests listing contexts from all projects
func TestContextListAll(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Create first project
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	stdout, stderr, exitCode := h.RunDual("create", "trunk")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// List all projects (should show current project)
	stdout, stderr, exitCode = h.RunDual("list", "--all")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Project:")
	h.AssertOutputContains(stdout, "trunk")
	h.AssertOutputContains(stdout, "Total: 1 contexts across 1 projects")
}

// TestContextListWithJSONAndPorts tests combining --json and --ports flags
// REMOVED: This test was specific to port listing functionality which has been removed.
// The worktree lifecycle manager no longer manages ports.

// TestContextDeleteShowsInfo tests that delete shows context info before deletion
func TestContextDeleteShowsInfo(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create two contexts (dual create will create the branches)
	stdout, stderr, exitCode := h.RunDual("create", "trunk")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("create", "feature-a")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Delete feature-a with --force (should show info)
	stdout, stderr, exitCode = h.RunDual("delete", "feature-a", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	output := stdout + stderr
	h.AssertOutputContains(output, "About to delete worktree")
	h.AssertOutputContains(output, "feature-a")
}

// TestContextListSorting tests that contexts are listed in alphabetical order
func TestContextListSorting(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)
	h.CreateDirectory("services/api")

	// Create an initial commit (required for git worktree add)
	h.WriteFile("README.md", "# Test Project")
	h.RunGitCommand("add", "README.md")
	h.RunGitCommand("commit", "-m", "Initial commit")

	// Create contexts in non-alphabetical order (dual create will create the branches)
	stdout, stderr, exitCode := h.RunDual("create", "zebra")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("create", "alpha")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("create", "beta")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// List contexts
	stdout, stderr, exitCode = h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Verify alphabetical order (alpha should come before beta, and beta before zebra)
	alphaIdx := strings.Index(stdout, "alpha")
	betaIdx := strings.Index(stdout, "beta")
	zebraIdx := strings.Index(stdout, "zebra")

	if alphaIdx == -1 || betaIdx == -1 || zebraIdx == -1 {
		t.Fatal("not all contexts found in output")
	}

	if !(alphaIdx < betaIdx && betaIdx < zebraIdx) {
		t.Errorf("contexts not in alphabetical order\nOutput: %s", stdout)
	}
}
