package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/logger"
	"github.com/lightfastai/dual/internal/worktree"
)

// ErrServiceNotDetected is returned when no service matches the current working directory
var ErrServiceNotDetected = fmt.Errorf("no service detected for current directory")

// ErrProjectRootNotFound is returned when the project root cannot be determined
var ErrProjectRootNotFound = fmt.Errorf("project root not found")

// Detector handles service detection logic
type Detector struct {
	// gitCommand allows for dependency injection in tests
	gitCommand func(args ...string) (string, error)
	// getwd allows for dependency injection in tests
	getwd func() (string, error)
	// evalSymlinks allows for dependency injection in tests
	evalSymlinks func(path string) (string, error)
}

// NewDetector creates a new Detector with default implementations
func NewDetector() *Detector {
	return &Detector{
		gitCommand:   execGitCommand,
		getwd:        os.Getwd,
		evalSymlinks: filepath.EvalSymlinks,
	}
}

// DetectService detects which service the current working directory belongs to
// It returns the service name and the project root path, or an error if no service matches
func (d *Detector) DetectService(cfg *config.Config, projectRoot string) (string, error) {
	// Get current working directory
	cwd, err := d.getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	logger.Debug("Current path: %s", cwd)

	// Resolve symlinks in both cwd and project root
	resolvedCwd, err := d.evalSymlinks(cwd)
	if err != nil {
		// If symlink resolution fails, use the original path
		resolvedCwd = cwd
	}

	resolvedProjectRoot, err := d.evalSymlinks(projectRoot)
	if err != nil {
		// If symlink resolution fails, use the original path
		resolvedProjectRoot = projectRoot
	}

	// Convert all service paths to absolute and resolve symlinks
	logger.Debug("Checking service paths...")
	servicePaths := make(map[string]string)
	for name, service := range cfg.Services {
		// Join with project root to make absolute
		absPath := filepath.Join(resolvedProjectRoot, service.Path)

		// Resolve symlinks
		resolvedPath, err := d.evalSymlinks(absPath)
		if err != nil {
			// If resolution fails, use the absolute path
			resolvedPath = absPath
		}

		// Clean the path to normalize it
		servicePaths[name] = filepath.Clean(resolvedPath)
		logger.Debug("  %s: %s", name, servicePaths[name])
	}

	// Check if CWD is within any service path
	// We need to find the longest matching path for nested structures
	var longestMatch string
	var longestMatchLen int

	for name, servicePath := range servicePaths {
		// Check if CWD is within this service path
		if isWithinPath(resolvedCwd, servicePath) {
			matchLen := len(servicePath)
			if matchLen > longestMatchLen {
				longestMatch = name
				longestMatchLen = matchLen
				logger.Debug("  %s: Match!", name)
			}
		}
	}

	if longestMatch == "" {
		return "", ErrServiceNotDetected
	}

	logger.Success("Service: %s", longestMatch)
	return longestMatch, nil
}

// FindProjectRoot attempts to find the project root using git or by walking up the directory tree
// If in a git worktree, returns the parent repository path to ensure all worktrees share the same project root
func (d *Detector) FindProjectRoot() (string, error) {
	// Try using worktree-aware git detection first
	wtDetector := worktree.NewDetector()

	// Get current working directory
	cwd, err := d.getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Try to find git root (could be worktree or normal repo)
	gitRoot, err := wtDetector.FindGitRoot(cwd)
	if err == nil {
		// Found a git root, get the project root (accounting for worktrees)
		projectRoot, err := wtDetector.GetProjectRoot(gitRoot)
		if err == nil {
			return projectRoot, nil
		}
		// If GetProjectRoot fails, fall through to config-based detection
	}

	// Fallback: walk up directory tree looking for dual.config.yml
	currentDir := cwd
	for {
		configPath := filepath.Join(currentDir, config.ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			// Found the config file
			resolved, err := d.evalSymlinks(currentDir)
			if err != nil {
				return currentDir, nil
			}
			return resolved, nil
		}

		// Move up one directory
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Reached the root without finding config
			break
		}
		currentDir = parent
	}

	return "", ErrProjectRootNotFound
}

// isWithinPath checks if targetPath is within or equal to basePath
func isWithinPath(targetPath, basePath string) bool {
	// Clean both paths
	target := filepath.Clean(targetPath)
	base := filepath.Clean(basePath)

	// Exact match
	if target == base {
		return true
	}

	// Check if target starts with base + separator
	// This ensures we don't match "/app" with "/application"
	baseWithSep := base + string(filepath.Separator)
	return strings.HasPrefix(target+string(filepath.Separator), baseWithSep)
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

// DetectService is a convenience function that creates a detector and detects the service
// It also attempts to find the project root automatically
func DetectService(cfg *config.Config, projectRoot string) (string, error) {
	detector := NewDetector()
	return detector.DetectService(cfg, projectRoot)
}

// DetectServiceWithRoot is a convenience function that finds the project root and detects the service
func DetectServiceWithRoot(cfg *config.Config) (string, string, error) {
	detector := NewDetector()
	projectRoot, err := detector.FindProjectRoot()
	if err != nil {
		return "", "", err
	}

	serviceName, err := detector.DetectService(cfg, projectRoot)
	if err != nil {
		return "", projectRoot, err
	}

	return serviceName, projectRoot, nil
}
