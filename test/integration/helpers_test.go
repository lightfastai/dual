package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelper provides utility functions for integration tests
type TestHelper struct {
	t            *testing.T
	TempDir      string
	ProjectDir   string
	DualBin      string
	OriginalHome string
	TestHome     string
}

// NewTestHelper creates a new test helper with a temp directory and builds the dual binary
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	// Create temp directory for the test
	tempDir := t.TempDir()

	// Create a project directory within the temp directory
	projectDir := filepath.Join(tempDir, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("failed to create project directory: %v", err)
	}

	// Build the dual binary
	dualBin := filepath.Join(tempDir, "dual")
	if err := buildDualBinary(t, dualBin); err != nil {
		t.Fatalf("failed to build dual binary: %v", err)
	}

	// Create a test home directory for registry isolation
	testHome := filepath.Join(tempDir, "home")
	if err := os.MkdirAll(testHome, 0o755); err != nil {
		t.Fatalf("failed to create test home directory: %v", err)
	}

	// Save original HOME
	originalHome := os.Getenv("HOME")

	helper := &TestHelper{
		t:            t,
		TempDir:      tempDir,
		ProjectDir:   projectDir,
		DualBin:      dualBin,
		OriginalHome: originalHome,
		TestHome:     testHome,
	}

	// Set HOME to test home for registry isolation
	helper.SetTestHome()

	return helper
}

// SetTestHome sets HOME to the test home directory
func (h *TestHelper) SetTestHome() {
	os.Setenv("HOME", h.TestHome)
}

// RestoreHome restores the original HOME environment variable
func (h *TestHelper) RestoreHome() {
	if h.OriginalHome != "" {
		os.Setenv("HOME", h.OriginalHome)
	}
}

// RunDual executes the dual binary with the given arguments
func (h *TestHelper) RunDual(args ...string) (string, string, int) {
	h.t.Helper()
	return h.RunDualInDir(h.ProjectDir, args...)
}

// RunDualInDir executes the dual binary in a specific directory
func (h *TestHelper) RunDualInDir(dir string, args ...string) (string, string, int) {
	h.t.Helper()

	cmd := exec.Command(h.DualBin, args...)
	cmd.Dir = dir

	// Set environment
	cmd.Env = append(os.Environ(), fmt.Sprintf("HOME=%s", h.TestHome))

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			h.t.Fatalf("failed to run command: %v", err)
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

// InitGitRepo initializes a git repository in the project directory
func (h *TestHelper) InitGitRepo() {
	h.t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = h.ProjectDir
	if output, err := cmd.CombinedOutput(); err != nil {
		h.t.Fatalf("failed to init git repo: %v\n%s", err, output)
	}

	// Configure git user for commits
	h.RunGitCommand("config", "user.email", "test@example.com")
	h.RunGitCommand("config", "user.name", "Test User")
}

// RunGitCommand runs a git command in the project directory
func (h *TestHelper) RunGitCommand(args ...string) (string, error) {
	h.t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = h.ProjectDir
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// CreateGitBranch creates and checks out a new git branch
func (h *TestHelper) CreateGitBranch(branchName string) {
	h.t.Helper()

	// Create an initial commit if necessary
	if _, err := h.RunGitCommand("rev-parse", "HEAD"); err != nil {
		// No commits yet, create an initial commit
		h.WriteFile("README.md", "# Test Project")
		h.RunGitCommand("add", "README.md")
		h.RunGitCommand("commit", "-m", "Initial commit")
	}

	// Check if branch already exists
	output, err := h.RunGitCommand("rev-parse", "--verify", branchName)
	if err == nil && strings.TrimSpace(output) != "" {
		// Branch exists, just check it out
		if _, err := h.RunGitCommand("checkout", branchName); err != nil {
			h.t.Fatalf("failed to checkout existing branch %s: %v", branchName, err)
		}
		return
	}

	// Branch doesn't exist, create it
	if _, err := h.RunGitCommand("checkout", "-b", branchName); err != nil {
		h.t.Fatalf("failed to create branch %s: %v", branchName, err)
	}
}

// CreateGitWorktree creates a git worktree
func (h *TestHelper) CreateGitWorktree(branchName, path string) string {
	h.t.Helper()

	worktreePath := filepath.Join(h.TempDir, path)
	output, err := h.RunGitCommand("worktree", "add", worktreePath, "-b", branchName)
	if err != nil {
		h.t.Fatalf("failed to create worktree: %v\n%s", err, output)
	}

	return worktreePath
}

// WriteFile writes content to a file relative to the project directory
func (h *TestHelper) WriteFile(relativePath, content string) {
	h.t.Helper()

	fullPath := filepath.Join(h.ProjectDir, relativePath)
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		h.t.Fatalf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		h.t.Fatalf("failed to write file %s: %v", fullPath, err)
	}
}

// ReadFile reads content from a file relative to the project directory
func (h *TestHelper) ReadFile(relativePath string) string {
	h.t.Helper()

	fullPath := filepath.Join(h.ProjectDir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		h.t.Fatalf("failed to read file %s: %v", fullPath, err)
	}

	return string(content)
}

// FileExists checks if a file exists relative to the project directory
func (h *TestHelper) FileExists(relativePath string) bool {
	h.t.Helper()

	fullPath := filepath.Join(h.ProjectDir, relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// CreateDirectory creates a directory relative to the project directory
func (h *TestHelper) CreateDirectory(relativePath string) {
	h.t.Helper()

	fullPath := filepath.Join(h.ProjectDir, relativePath)
	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		h.t.Fatalf("failed to create directory %s: %v", fullPath, err)
	}
}

// AssertFileContains checks if a file contains a specific string
func (h *TestHelper) AssertFileContains(relativePath, expectedContent string) {
	h.t.Helper()

	content := h.ReadFile(relativePath)
	if !strings.Contains(content, expectedContent) {
		h.t.Errorf("file %s does not contain expected content\nExpected substring: %s\nActual content:\n%s",
			relativePath, expectedContent, content)
	}
}

// AssertExitCode checks if the exit code matches the expected value
func (h *TestHelper) AssertExitCode(exitCode, expected int, output string) {
	h.t.Helper()

	if exitCode != expected {
		h.t.Errorf("unexpected exit code: got %d, want %d\nOutput: %s", exitCode, expected, output)
	}
}

// AssertOutputContains checks if output contains a specific string
func (h *TestHelper) AssertOutputContains(output, expected string) {
	h.t.Helper()

	if !strings.Contains(output, expected) {
		h.t.Errorf("output does not contain expected string\nExpected: %s\nActual: %s", expected, output)
	}
}

// AssertOutputNotContains checks if output does not contain a specific string
func (h *TestHelper) AssertOutputNotContains(output, unexpected string) {
	h.t.Helper()

	if strings.Contains(output, unexpected) {
		h.t.Errorf("output contains unexpected string\nUnexpected: %s\nActual: %s", unexpected, output)
	}
}

// ReadRegistryJSON reads the registry.json file from the project directory
func (h *TestHelper) ReadRegistryJSON() string {
	h.t.Helper()

	// Registry is now project-local, not in home directory
	registryPath := filepath.Join(h.ProjectDir, ".dual", ".local", "registry.json")
	content, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		h.t.Fatalf("failed to read registry.json: %v", err)
	}

	return string(content)
}

// RegistryExists checks if the registry file exists
func (h *TestHelper) RegistryExists() bool {
	h.t.Helper()

	// Registry is now project-local, not in home directory
	registryPath := filepath.Join(h.ProjectDir, ".dual", ".local", "registry.json")
	_, err := os.Stat(registryPath)
	return err == nil
}

// AssertFileExists checks that a file exists
func (h *TestHelper) AssertFileExists(relativePath string) {
	h.t.Helper()

	fullPath := filepath.Join(h.ProjectDir, relativePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		h.t.Fatalf("expected file to exist: %s", relativePath)
	}
}

// AssertFileNotExists checks that a file does not exist
func (h *TestHelper) AssertFileNotExists(relativePath string) {
	h.t.Helper()

	fullPath := filepath.Join(h.ProjectDir, relativePath)
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		if err != nil {
			h.t.Fatalf("error checking file %s: %v", relativePath, err)
		} else {
			h.t.Fatalf("expected file to not exist: %s", relativePath)
		}
	}
}

// ReadFileInDir reads a file from a specific directory (not relative to ProjectDir)
func (h *TestHelper) ReadFileInDir(dir, relativePath string) string {
	h.t.Helper()

	fullPath := filepath.Join(dir, relativePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		h.t.Fatalf("failed to read file %s: %v", fullPath, err)
	}

	return string(content)
}

// FileExistsInDir checks if a file exists in a specific directory
func (h *TestHelper) FileExistsInDir(dir, relativePath string) bool {
	h.t.Helper()

	fullPath := filepath.Join(dir, relativePath)
	_, err := os.Stat(fullPath)
	return err == nil
}

// buildDualBinary builds the dual binary to the specified path
func buildDualBinary(t *testing.T, outputPath string) error {
	t.Helper()

	// Get the project root (3 levels up from test/integration)
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return fmt.Errorf("failed to get project root: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/dual")
	cmd.Dir = projectRoot
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build dual: %w\nOutput: %s", err, output)
	}

	return nil
}
