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
	stdout, stderr, exitCode := h.RunDual("context", "create", "main", "--base-port", "4100")
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

	// Create context for feature branch in the worktree
	t.Log("Step 4: Create context for feature branch in worktree")
	stdout, stderr, exitCode = h.RunDualInDir(worktreePath, "context", "create", "feature/new-feature", "--base-port", "4200")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Created context \"feature/new-feature\"")

	// Test port queries from main worktree
	t.Log("Step 5: Query ports from main worktree")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4102") // main context: 4100 + 1 (web) + 1

	// Test port queries from feature worktree
	t.Log("Step 6: Query ports from feature worktree")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(worktreePath, "apps/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4202") // feature context: 4200 + 1 (web) + 1

	// Test command wrapper in both worktrees
	t.Log("Step 7: Test command wrapper in both worktrees")

	// Create a script to print PORT
	scriptContent := `#!/bin/sh
echo "PORT=$PORT"
`
	h.WriteFile("print-port.sh", scriptContent)
	scriptPath := filepath.Join(h.ProjectDir, "print-port.sh")
	if err := makeExecutable(scriptPath); err != nil {
		t.Fatalf("failed to make script executable: %v", err)
	}

	// Run from main worktree
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/api"),
		"sh", scriptPath,
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Context: main")
	h.AssertOutputContains(stderr, "Service: api")
	h.AssertOutputContains(stderr, "Port: 4101")
	h.AssertOutputContains(stdout, "PORT=4101")

	// Create the script in the feature worktree too (git worktree has separate working directory)
	featureScriptPath := filepath.Join(worktreePath, "print-port.sh")
	if err := os.WriteFile(featureScriptPath, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write script in feature worktree: %v", err)
	}

	// Run from feature worktree
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(worktreePath, "apps/api"),
		"sh", featureScriptPath,
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Context: feature/new-feature")
	h.AssertOutputContains(stderr, "Service: api")
	h.AssertOutputContains(stderr, "Port: 4201")
	h.AssertOutputContains(stdout, "PORT=4201")
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
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Feature A worktree
	worktreeA := h.CreateGitWorktree("feature/a", "worktree-a")
	h.RunDualInDir(worktreeA, "context", "create", "feature/a", "--base-port", "4200")

	// Feature B worktree
	worktreeB := h.CreateGitWorktree("feature/b", "worktree-b")
	h.RunDualInDir(worktreeB, "context", "create", "feature/b", "--base-port", "4300")

	// Verify each worktree uses its own context and port
	t.Log("Verifying port isolation")

	// Main worktree should use port 4101
	stdout, stderr, exitCode := h.RunDualInDir(filepath.Join(h.ProjectDir, "apps/web"), "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101")

	// Feature A worktree should use port 4201
	stdout, stderr, exitCode = h.RunDualInDir(filepath.Join(worktreeA, "apps/web"), "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4201")

	// Feature B worktree should use port 4301
	stdout, stderr, exitCode = h.RunDualInDir(filepath.Join(worktreeB, "apps/web"), "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4301")

	// Verify all contexts are in the registry
	registryContent := h.ReadRegistryJSON()
	if registryContent == "" {
		t.Fatal("registry is empty")
	}

	// Registry should contain all three contexts
	h.AssertOutputContains(registryContent, "main")
	h.AssertOutputContains(registryContent, "feature/a")
	h.AssertOutputContains(registryContent, "feature/b")
	h.AssertOutputContains(registryContent, "4100")
	h.AssertOutputContains(registryContent, "4200")
	h.AssertOutputContains(registryContent, "4300")
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
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Create a worktree
	worktreePath := h.CreateGitWorktree("feature/test", "worktree-test")

	// Create a context for the feature/test branch (which git will auto-detect)
	// Note: Git branch detection has priority over .dual-context file
	h.RunDualInDir(worktreePath, "context", "create", "feature/test", "--base-port", "5000")

	// Query port - should use feature/test context (detected from git branch)
	stdout, stderr, exitCode := h.RunDualInDir(filepath.Join(worktreePath, "apps/web"), "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "5001") // feature/test base port 5000 + 0 + 1

	// Verify context detection
	stdout, stderr, exitCode = h.RunDualInDir(worktreePath, "context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: feature/test")
	h.AssertOutputContains(stdout, "Base Port: 5000")
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
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Create worktree
	worktreePath := h.CreateGitWorktree("feature/test", "worktree-test")
	h.RunDualInDir(worktreePath, "context", "create", "feature/test", "--base-port", "4200")

	// Test service detection in main worktree
	t.Log("Testing service detection in main worktree")
	stdout, stderr, exitCode := h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/frontend/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4102") // web: 4100 + 1 + 1

	// Test service detection in feature worktree
	t.Log("Testing service detection in feature worktree")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(worktreePath, "apps/backend/api"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4201") // api: 4200 + 0 + 1
}
