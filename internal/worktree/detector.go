package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Detector handles git worktree detection logic
type Detector struct {
	// gitCommand allows for dependency injection in tests
	gitCommand func(args ...string) (string, error)
	// stat allows for dependency injection in tests
	stat func(path string) (os.FileInfo, error)
	// readFile allows for dependency injection in tests
	readFile func(path string) ([]byte, error)
	// evalSymlinks allows for dependency injection in tests
	evalSymlinks func(path string) (string, error)
}

// NewDetector creates a new Detector with default implementations
func NewDetector() *Detector {
	return &Detector{
		gitCommand:   execGitCommand,
		stat:         os.Stat,
		readFile:     os.ReadFile,
		evalSymlinks: filepath.EvalSymlinks,
	}
}

// IsWorktree checks if the given directory is a git worktree
// A worktree has a .git file (not directory) pointing to the parent repository
// This distinguishes from git submodules, which also have a .git file but with different content
func (d *Detector) IsWorktree(dir string) (bool, error) {
	gitPath := filepath.Join(dir, ".git")

	// Check if .git exists
	info, err := d.stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No .git at all, not a git repo or worktree
			return false, nil
		}
		return false, fmt.Errorf("failed to stat .git: %w", err)
	}

	// If .git is a directory, it's a normal repo, not a worktree
	if info.IsDir() {
		return false, nil
	}

	// .git is a file - could be a worktree or submodule
	// Read the file to distinguish between them
	content, err := d.readFile(gitPath)
	if err != nil {
		return false, fmt.Errorf("failed to read .git file: %w", err)
	}

	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir: ") {
		return false, nil
	}

	// Extract the gitdir path
	gitdir := strings.TrimPrefix(line, "gitdir: ")

	// Worktrees point to .git/worktrees/<name>
	// Submodules point to ../.git/modules/<name> or similar
	// Check if the gitdir contains "/worktrees/"
	return strings.Contains(gitdir, "/worktrees/") || strings.Contains(gitdir, "\\worktrees\\"), nil
}

// GetParentRepo returns the path to the parent repository for a worktree
// Returns an error if the directory is not a worktree or if the parent cannot be found
func (d *Detector) GetParentRepo(worktreeDir string) (string, error) {
	// First verify it's actually a worktree
	isWT, err := d.IsWorktree(worktreeDir)
	if err != nil {
		return "", err
	}
	if !isWT {
		return "", fmt.Errorf("directory is not a worktree")
	}

	// Read the .git file to get the gitdir path
	// Format: gitdir: /path/to/main/.git/worktrees/name
	gitFilePath := filepath.Join(worktreeDir, ".git")
	content, err := d.readFile(gitFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read .git file: %w", err)
	}

	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", fmt.Errorf("invalid .git file format: expected 'gitdir: <path>'")
	}

	// Extract the gitdir path
	gitdir := strings.TrimPrefix(line, "gitdir: ")

	// The parent repo is three directories up from the gitdir
	// e.g., /home/user/project/.git/worktrees/name → /home/user/project
	// gitdir: /home/user/project/.git/worktrees/name
	// parent of gitdir: /home/user/project/.git/worktrees
	// parent of parent: /home/user/project/.git
	// parent of that: /home/user/project
	parentRepo := filepath.Dir(filepath.Dir(filepath.Dir(gitdir)))

	// Validate the parent repo exists
	if _, err := d.stat(parentRepo); err != nil {
		return "", fmt.Errorf("parent repo not found at %s: %w", parentRepo, err)
	}

	// Resolve symlinks for consistency
	resolved, err := d.evalSymlinks(parentRepo)
	if err != nil {
		// If symlink resolution fails, return the unresolved path
		return parentRepo, nil
	}

	return resolved, nil
}

// GetProjectRoot returns the project root, accounting for worktrees
// If in a worktree, returns the parent repository path
// If in a normal repo, returns the repository path
// If not in a git repo, returns an error
func (d *Detector) GetProjectRoot(dir string) (string, error) {
	// Check if this is a worktree
	isWT, err := d.IsWorktree(dir)
	if err != nil {
		return "", err
	}

	if isWT {
		// Return parent repository path
		return d.GetParentRepo(dir)
	}

	// Not a worktree, check if it's a normal git repo
	gitPath := filepath.Join(dir, ".git")
	if info, err := d.stat(gitPath); err == nil && info.IsDir() {
		// It's a normal git repo, resolve symlinks and return
		resolved, err := d.evalSymlinks(dir)
		if err != nil {
			return dir, nil
		}
		return resolved, nil
	}

	// Not a git repo at all
	return "", fmt.Errorf("not a git repository or worktree")
}

// FindGitRoot walks up from the given directory to find the git root
// This can be either a worktree or a normal repository
func (d *Detector) FindGitRoot(startDir string) (string, error) {
	currentDir := startDir

	for {
		// Check if .git exists (file or directory)
		gitPath := filepath.Join(currentDir, ".git")
		if _, err := d.stat(gitPath); err == nil {
			// Found .git, return this directory
			return currentDir, nil
		}

		// Move up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached the root without finding .git
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("no .git found in directory tree")
}

// GetProjectRootFromCwd finds the project root starting from the current working directory
// This is a convenience function that combines FindGitRoot and GetProjectRoot
func (d *Detector) GetProjectRootFromCwd() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Find the git root (worktree or normal repo)
	gitRoot, err := d.FindGitRoot(cwd)
	if err != nil {
		return "", err
	}

	// Get the project root (accounting for worktrees)
	return d.GetProjectRoot(gitRoot)
}

// execGitCommand executes a git command and returns the output
func execGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// IsWorktree is a convenience function that checks if the current directory is a worktree
func IsWorktree() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}

	detector := NewDetector()
	gitRoot, err := detector.FindGitRoot(cwd)
	if err != nil {
		return false, nil // Not in a git repo, so not a worktree
	}

	return detector.IsWorktree(gitRoot)
}

// GetProjectRoot is a convenience function that gets the project root from the current directory
func GetProjectRoot() (string, error) {
	detector := NewDetector()
	return detector.GetProjectRootFromCwd()
}
