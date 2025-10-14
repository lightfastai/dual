package context

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// DualContextFile is the name of the file that can override context detection
	DualContextFile = ".dual-context"
	// DefaultContext is the fallback context name
	DefaultContext = "default"
)

// Detector is responsible for detecting the current development context
type Detector struct {
	// gitCommand allows for dependency injection in tests
	gitCommand func(args ...string) (string, error)
	// readFile allows for dependency injection in tests
	readFile func(path string) ([]byte, error)
	// getwd allows for dependency injection in tests
	getwd func() (string, error)
}

// NewDetector creates a new Detector with default implementations
func NewDetector() *Detector {
	return &Detector{
		gitCommand: execGitCommand,
		readFile:   os.ReadFile,
		getwd:      os.Getwd,
	}
}

// DetectContext detects the current development context with priority:
// 1. Git branch name (if in a git repository)
// 2. .dual-context file (walks up directory tree)
// 3. "default" (fallback)
func (d *Detector) DetectContext() (string, error) {
	// Priority 1: Try git branch
	if branch, err := d.detectGitBranch(); err == nil && branch != "" {
		return branch, nil
	}

	// Priority 2: Look for .dual-context file
	if context, err := d.findDualContextFile(); err == nil && context != "" {
		return context, nil
	}

	// Priority 3: Return default
	return DefaultContext, nil
}

// detectGitBranch attempts to detect the current git branch
func (d *Detector) detectGitBranch() (string, error) {
	output, err := d.gitCommand("branch", "--show-current")
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		// Could be in detached HEAD state or not a git repo
		return "", fmt.Errorf("no current branch (detached HEAD or not a git repo)")
	}

	return branch, nil
}

// findDualContextFile walks up the directory tree looking for .dual-context file
func (d *Detector) findDualContextFile() (string, error) {
	cwd, err := d.getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Walk up the directory tree
	currentDir := cwd
	for {
		contextPath := filepath.Join(currentDir, DualContextFile)

		// Try to read the file
		data, err := d.readFile(contextPath)
		if err == nil {
			// File exists, read the context name
			context := strings.TrimSpace(string(data))
			if context == "" {
				return "", fmt.Errorf("empty .dual-context file at %s", contextPath)
			}
			return context, nil
		}

		// Check if we've reached the root
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// We've reached the root directory
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("no .dual-context file found")
}

// DetectContext is a convenience function that creates a new detector and detects the context
func DetectContext() (string, error) {
	detector := NewDetector()
	return detector.DetectContext()
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
