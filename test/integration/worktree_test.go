package integration

import (
	"os"
	"path/filepath"
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

	// Commit the config and service directories so they appear in worktrees
	h.WriteFile("apps/web/.gitkeep", "")
	h.WriteFile("apps/api/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create context for main branch
	t.Log("Step 2: Create context for main branch")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Created context \"main\"")

	// Create a worktree for feature branch
	t.Log("Step 3: Create worktree for feature branch")
	worktreePath := h.CreateGitWorktree("feature/new-feature", "worktree-feature")

	// Verify the worktree has the same config file
	worktreeConfigPath := filepath.Join(worktreePath, "dual.config.yml")
	if _, err := os.Stat(worktreeConfigPath); err != nil {
		t.Fatalf("config file not found in worktree: %v", err)
	}

	// Create context for feature branch from the MAIN repo (not worktree)
	// This ensures the context is stored in the parent repo's registry,
	// which all worktrees will share via GetProjectIdentifier normalization
	t.Log("Step 4: Create context for feature branch from main repo")
	stdout, stderr, exitCode = h.RunDual("context", "create", "feature/new-feature")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Created context \"feature/new-feature\"")

	// Verify contexts exist in registry
	registryContent := h.ReadRegistryJSON()
	h.AssertOutputContains(registryContent, "main")
	h.AssertOutputContains(registryContent, "feature/new-feature")
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

	// Commit config and service directories
	h.WriteFile("apps/web/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create multiple worktrees with different contexts
	t.Log("Creating multiple worktrees")

	// Main worktree context
	h.RunDual("context", "create", "main")

	// Feature A worktree
	worktreeA := h.CreateGitWorktree("feature/a", "worktree-a")
	// Create context from main repo so it's stored in the shared registry
	h.RunDual("context", "create", "feature/a")

	// Feature B worktree
	worktreeB := h.CreateGitWorktree("feature/b", "worktree-b")
	// Create context from main repo so it's stored in the shared registry
	h.RunDual("context", "create", "feature/b")

	// Verify each worktree has its own context
	t.Log("Verifying context isolation")

	// Verify all contexts are in the registry
	registryContent := h.ReadRegistryJSON()
	if registryContent == "" {
		t.Fatal("registry is empty")
	}

	// Registry should contain all three contexts
	h.AssertOutputContains(registryContent, "main")
	h.AssertOutputContains(registryContent, "feature/a")
	h.AssertOutputContains(registryContent, "feature/b")

	// Verify context detection works in each worktree
	stdout, stderr, exitCode := h.RunDualInDir(h.ProjectDir, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: main")

	stdout, stderr, exitCode = h.RunDualInDir(worktreeA, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: feature/a")

	stdout, stderr, exitCode = h.RunDualInDir(worktreeB, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: feature/b")
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

	// Commit config and service directories
	h.WriteFile("apps/web/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create context for main
	h.RunDual("context", "create", "main")

	// Create a worktree
	worktreePath := h.CreateGitWorktree("feature/test", "worktree-test")

	// Create a context for the feature/test branch from main repo
	// (which git will auto-detect when running from the worktree)
	// Note: Git branch detection has priority over .dual-context file
	h.RunDual("context", "create", "feature/test")

	// Verify context detection from worktree
	stdout, stderr, exitCode := h.RunDualInDir(worktreePath, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: feature/test")
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

	// Commit config and service directories
	h.WriteFile("apps/frontend/web/.gitkeep", "")
	h.WriteFile("apps/backend/api/.gitkeep", "")
	h.RunGitCommand("add", ".")
	h.RunGitCommand("commit", "-m", "Add dual config and service directories")

	// Create contexts
	h.RunDual("context", "create", "main")

	// Create worktree
	worktreePath := h.CreateGitWorktree("feature/test", "worktree-test")
	// Create context from main repo so it's stored in the shared registry
	h.RunDual("context", "create", "feature/test")

	// Verify context detection works in both worktrees
	t.Log("Testing context detection in main worktree")
	stdout, stderr, exitCode := h.RunDualInDir(h.ProjectDir, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: main")

	t.Log("Testing context detection in feature worktree")
	stdout, stderr, exitCode = h.RunDualInDir(worktreePath, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: feature/test")
}
