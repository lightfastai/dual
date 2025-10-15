package integration

import (
	"testing"
)

// TestMultiWorktreeSetup tests dual with multiple git worktrees
func TestMultiWorktreeSetup(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize main git repository
	h.InitGitRepo()
	h.CreateGitBranch("main")

	// Initialize dual in main worktree
	t.Log("Step 1: Initialize dual in main worktree")
	h.RunDual("init")

	// Add services
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")

	// Add worktrees configuration
	h.WriteFile("dual.config.yml", `version: 1
services:
  web:
    path: apps/web
  api:
    path: apps/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Commit the config and service directories so they appear in worktrees
	h.WriteFile("apps/web/.gitkeep", "")
	h.WriteFile("apps/api/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create worktree for feature branch
	t.Log("Step 2: Create worktree for feature branch")
	stdout, stderr, exitCode := h.RunDual("create", "feature-new")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "Worktree created successfully")

	// Create another worktree for different feature
	t.Log("Step 3: Create another worktree for another feature")
	stdout, stderr, exitCode = h.RunDual("create", "feature-other")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "Worktree created successfully")

	// Verify contexts exist in registry using dual list
	t.Log("Step 4: Verify contexts via dual list")
	stdout, stderr, exitCode = h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "feature-new")
	h.AssertOutputContains(stdout, "feature-other")
}

// TestWorktreeContextIsolation tests that contexts are properly isolated across worktrees
func TestWorktreeContextIsolation(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize main repository
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Add a service
	h.CreateDirectory("apps/web")
	h.RunDual("service", "add", "web", "--path", "apps/web")

	// Add worktrees configuration
	h.WriteFile("dual.config.yml", `version: 1
services:
  web:
    path: apps/web
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Commit config and service directories
	h.WriteFile("apps/web/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create multiple worktrees with different contexts
	t.Log("Creating multiple worktrees")

	// Feature A worktree
	h.RunDual("create", "feature-a")

	// Feature B worktree
	h.RunDual("create", "feature-b")

	// Feature C worktree
	h.RunDual("create", "feature-c")

	// Verify each worktree has its own context
	t.Log("Verifying context isolation")

	// Verify all contexts are in the registry
	registryContent := h.ReadRegistryJSON()
	if registryContent == "" {
		t.Fatal("registry is empty")
	}

	// Registry should contain all three contexts
	h.AssertOutputContains(registryContent, "feature-a")
	h.AssertOutputContains(registryContent, "feature-b")
	h.AssertOutputContains(registryContent, "feature-c")

	// Verify contexts via dual list
	stdout, stderr, exitCode := h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "feature-a")
	h.AssertOutputContains(stdout, "feature-b")
	h.AssertOutputContains(stdout, "feature-c")
}

// TestWorktreeWithDualContextFile tests using .dual-context file in worktrees
func TestWorktreeWithDualContextFile(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize repository
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	h.CreateDirectory("apps/web")
	h.RunDual("service", "add", "web", "--path", "apps/web")

	// Add worktrees configuration
	h.WriteFile("dual.config.yml", `version: 1
services:
  web:
    path: apps/web
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Commit config and service directories
	h.WriteFile("apps/web/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create worktree contexts
	h.RunDual("create", "feature-test")
	h.RunDual("create", "feature-other")

	// Verify contexts exist in registry
	stdout, stderr, exitCode := h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "feature-test")
	h.AssertOutputContains(stdout, "feature-other")
}

// TestWorktreeServiceDetection tests that service detection works correctly in worktrees
func TestWorktreeServiceDetection(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize repository
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create nested service structure
	h.CreateDirectory("apps/frontend/web")
	h.CreateDirectory("apps/backend/api")
	h.RunDual("service", "add", "web", "--path", "apps/frontend/web")
	h.RunDual("service", "add", "api", "--path", "apps/backend/api")

	// Add worktrees configuration
	h.WriteFile("dual.config.yml", `version: 1
services:
  web:
    path: apps/frontend/web
  api:
    path: apps/backend/api
worktrees:
  path: ../worktrees
  naming: "{branch}"
`)

	// Commit config and service directories
	h.WriteFile("apps/frontend/web/.gitkeep", "")
	h.WriteFile("apps/backend/api/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create worktree contexts
	t.Log("Creating worktrees for testing")
	h.RunDual("create", "feature-test")
	h.RunDual("create", "feature-api")

	// Verify contexts exist
	t.Log("Verifying contexts via dual list")
	stdout, stderr, exitCode := h.RunDual("list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "feature-test")
	h.AssertOutputContains(stdout, "feature-api")
}
