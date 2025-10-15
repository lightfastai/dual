package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/env"
	dualerrors "github.com/lightfastai/dual/internal/errors"
	"github.com/lightfastai/dual/internal/hooks"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var createFromRef string

var createCmd = &cobra.Command{
	Use:   "create <branch-name>",
	Short: "Create a new worktree with dual context",
	Long: `Create a new git worktree with an integrated dual context.

This command:
1. Creates a git worktree at the configured location
2. Registers a new dual context
3. Runs lifecycle hooks (postWorktreeCreate)

Examples:
  dual create feature-auth              # Create worktree for feature-auth branch
  dual create hotfix-123 --from main    # Create from specific ref`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().StringVar(&createFromRef, "from", "", "Create worktree from this ref (branch/commit)")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	branchName := args[0]

	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Validate we're in project root
	if err := validateProjectRoot(projectRoot); err != nil {
		return err
	}

	// Get the normalized project identifier for registry operations
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry (using projectRoot to construct the correct registry file path)
	reg, err := registry.LoadRegistry(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Validate context doesn't exist
	if reg.ContextExists(projectIdentifier, branchName) {
		return fmt.Errorf("context %q already exists\nHint: Use a different branch name or delete the existing context first", branchName)
	}

	// Determine worktree path
	worktreePath, err := prepareWorktreePath(cfg, projectRoot, branchName)
	if err != nil {
		return err
	}

	// Create git worktree
	if err := createGitWorktree(projectRoot, branchName, worktreePath); err != nil {
		return err
	}

	// Register context
	if err := registerContext(reg, projectIdentifier, branchName, worktreePath, projectRoot); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "[dual] Created context: %s\n", branchName)

	// Execute hooks and apply env overrides
	executeHooksAndApplyEnv(cfg, reg, projectRoot, projectIdentifier, branchName, worktreePath)

	printSuccess(branchName, worktreePath)
	return nil
}

// validateProjectRoot checks we're running from the project root
func validateProjectRoot(projectRoot string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Resolve both paths to handle symlinks
	currentDirAbs, err := filepath.EvalSymlinks(currentDir)
	if err != nil {
		currentDirAbs = currentDir
	}
	projectRootAbs, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		projectRootAbs = projectRoot
	}

	if currentDirAbs != projectRootAbs {
		return fmt.Errorf("dual create must be run from the project root directory\nCurrent directory: %s\nProject root: %s\nHint: cd %s", currentDir, projectRoot, projectRoot)
	}

	return nil
}

// prepareWorktreePath determines and validates the worktree path
func prepareWorktreePath(cfg *config.Config, projectRoot, branchName string) (string, error) {
	worktreesBasePath := cfg.GetWorktreePath(projectRoot)
	worktreeName := cfg.GetWorktreeName(branchName)
	worktreePath := filepath.Join(worktreesBasePath, worktreeName)

	// Check if worktree directory already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree directory already exists: %s\nHint: Remove it manually or use a different branch name", worktreePath)
	}

	// Create worktrees base directory if it doesn't exist
	if err := os.MkdirAll(worktreesBasePath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	return worktreePath, nil
}

// createGitWorktree creates the git worktree
func createGitWorktree(projectRoot, branchName, worktreePath string) error {
	// Build git worktree add command
	gitArgs := buildGitWorktreeArgs(branchName, worktreePath)

	fmt.Fprintf(os.Stderr, "[dual] Creating git worktree...\n")
	fmt.Fprintf(os.Stderr, "  Branch: %s\n", branchName)
	if createFromRef != "" {
		fmt.Fprintf(os.Stderr, "  From: %s\n", createFromRef)
	}
	fmt.Fprintf(os.Stderr, "  Path: %s\n", worktreePath)

	// Execute git worktree add
	// #nosec G204 - Git command with controlled arguments
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Dir = projectRoot
	gitCmd.Stdout = os.Stdout

	// Capture stderr to parse errors
	var stderr bytes.Buffer
	gitCmd.Stderr = &stderr

	if err := gitCmd.Run(); err != nil {
		stderrStr := stderr.String()

		// Parse common git errors and provide helpful messages
		dualErr := dualerrors.New(dualerrors.ErrCommandFailed, "Failed to create git worktree")
		dualErr.WithContext("Branch", branchName)
		dualErr.WithContext("Path", worktreePath)
		if createFromRef != "" {
			dualErr.WithContext("From ref", createFromRef)
		}
		dualErr.WithCause(err)

		// Parse specific git errors
		if strings.Contains(stderrStr, "already exists") {
			if strings.Contains(stderrStr, "branch") {
				dualErr.WithContext("Issue", "Branch already exists")
				dualErr.WithFixes(
					fmt.Sprintf("The branch '%s' already exists", branchName),
					"",
					"Solutions:",
					fmt.Sprintf("  1. Use the existing branch: git checkout %s", branchName),
					fmt.Sprintf("  2. Use a different name: dual create %s-2", branchName),
					fmt.Sprintf("  3. Delete the existing branch first:"),
					fmt.Sprintf("     git branch -D %s", branchName),
				)
			} else {
				dualErr.WithContext("Issue", "Path already exists")
				dualErr.WithFixes(
					"The worktree path already exists",
					fmt.Sprintf("Remove it first: rm -rf %s", worktreePath),
				)
			}
		} else if strings.Contains(stderrStr, "not a valid") || strings.Contains(stderrStr, "invalid") {
			dualErr.WithContext("Issue", "Invalid branch name")
			dualErr.WithFixes(
				fmt.Sprintf("'%s' is not a valid branch name", branchName),
				"",
				"Branch names cannot contain:",
				"  • Spaces (use hyphens instead)",
				"  • Special characters: ~, ^, :, ?, *, [, \\",
				"  • Double dots ..",
				"  • Names ending with .lock",
				"",
				"Example valid names:",
				"  • feature-auth",
				"  • bugfix/issue-123",
				"  • release-v1.0.0",
			)
		} else if strings.Contains(stderrStr, "unknown revision") || strings.Contains(stderrStr, "bad revision") {
			dualErr.WithContext("Issue", "Reference not found")
			dualErr.WithFixes(
				fmt.Sprintf("The reference '%s' does not exist", createFromRef),
				"",
				"Check available branches: git branch -a",
				"Check available tags: git tag -l",
				"Check if you need to fetch: git fetch origin",
			)
		} else if strings.Contains(stderrStr, "not a git repository") || strings.Contains(stderrStr, "not in a git") {
			dualErr.WithContext("Issue", "Not a git repository")
			dualErr.WithFixes(
				"This directory is not a git repository",
				"",
				"Initialize a git repository first:",
				"  git init",
				"",
				"Or clone an existing repository:",
				"  git clone <repository-url>",
			)
		} else if strings.Contains(stderrStr, "could not create directory") {
			dualErr.WithContext("Issue", "Permission denied")
			dualErr.WithFixes(
				"Cannot create the worktree directory",
				"Check permissions for the parent directory",
				fmt.Sprintf("Create parent directory: mkdir -p %s", filepath.Dir(worktreePath)),
			)
		} else {
			// Generic error with git output
			dualErr.WithContext("Git error", strings.TrimSpace(stderrStr))
			dualErr.WithFixes(
				"Check the git error message above",
				"Ensure you have the latest git version: git --version",
				"Try running the command manually:",
				fmt.Sprintf("  cd %s", projectRoot),
				fmt.Sprintf("  git %s", strings.Join(gitArgs, " ")),
			)
		}

		return dualErr
	}

	return nil
}

// buildGitWorktreeArgs constructs git worktree add arguments
func buildGitWorktreeArgs(branchName, worktreePath string) []string {
	gitArgs := []string{"worktree", "add"}

	if createFromRef == "" {
		// Create new branch from current HEAD
		gitArgs = append(gitArgs, "-b", branchName, worktreePath)
	} else {
		// Create new branch from specific ref
		gitArgs = append(gitArgs, "-b", branchName, worktreePath, createFromRef)
	}

	return gitArgs
}

// registerContext creates and saves context in registry
func registerContext(reg *registry.Registry, projectIdentifier, branchName, worktreePath, projectRoot string) error {
	// Create context in registry
	if err := reg.SetContext(projectIdentifier, branchName, worktreePath); err != nil {
		// Cleanup: remove the worktree we just created
		_ = removeGitWorktree(worktreePath, projectRoot)
		return fmt.Errorf("failed to create context: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		// Cleanup: remove the worktree and context
		_ = reg.DeleteContext(projectIdentifier, branchName)
		_ = removeGitWorktree(worktreePath, projectRoot)
		return fmt.Errorf("failed to save registry: %w", err)
	}

	return nil
}

// executeHooksAndApplyEnv runs hooks and applies environment overrides
func executeHooksAndApplyEnv(cfg *config.Config, reg *registry.Registry, projectRoot, projectIdentifier, branchName, worktreePath string) {
	// Prepare hook context
	hookCtx := hooks.HookContext{
		Event:       hooks.PostWorktreeCreate,
		ContextName: branchName,
		ContextPath: worktreePath,
		ProjectRoot: projectRoot,
	}

	// Create hook manager
	hookMgr := hooks.NewManager(cfg, projectRoot)

	// Run postWorktreeCreate hooks and capture env overrides
	envOverrides, err := hookMgr.Execute(hooks.PostWorktreeCreate, hookCtx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: postWorktreeCreate hook failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "[dual] Worktree created but hooks failed. You may need to run setup manually.\n")
		return
	}

	// Apply environment overrides
	applyEnvOverrides(cfg, reg, projectIdentifier, branchName, worktreePath, envOverrides)
}

// applyEnvOverrides applies environment overrides to registry and generates env files
func applyEnvOverrides(cfg *config.Config, reg *registry.Registry, projectIdentifier, branchName, worktreePath string, envOverrides *hooks.EnvOverrides) {
	if envOverrides.IsEmpty() {
		return
	}

	// Apply global overrides (serviceName = "")
	for key, value := range envOverrides.Global {
		if err := reg.SetEnvOverrideForService(projectIdentifier, branchName, key, value, ""); err != nil {
			fmt.Fprintf(os.Stderr, "[dual] Warning: failed to set global env override %s: %v\n", key, err)
		}
	}

	// Apply service-specific overrides
	for serviceName, serviceVars := range envOverrides.Services {
		for key, value := range serviceVars {
			if err := reg.SetEnvOverrideForService(projectIdentifier, branchName, key, value, serviceName); err != nil {
				fmt.Fprintf(os.Stderr, "[dual] Warning: failed to set service env override %s.%s: %v\n", serviceName, key, err)
			}
		}
	}

	// Save registry with new overrides
	if err := reg.SaveRegistry(); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to save registry with env overrides: %v\n", err)
	}

	// Generate service env files in the worktree (not parent repo)
	if err := env.GenerateServiceEnvFiles(cfg, reg, worktreePath, projectIdentifier, branchName); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to generate service env files: %v\n", err)
	}
}

// printSuccess prints success message
func printSuccess(branchName, worktreePath string) {
	fmt.Fprintf(os.Stderr, "\n[dual] Worktree created successfully!\n")
	fmt.Fprintf(os.Stderr, "  Context: %s\n", branchName)
	fmt.Fprintf(os.Stderr, "  Path: %s\n", worktreePath)
	fmt.Fprintf(os.Stderr, "\nTo switch to this worktree:\n")
	fmt.Fprintf(os.Stderr, "  cd %s\n", worktreePath)
}

// removeGitWorktree removes a git worktree
func removeGitWorktree(worktreePath, projectRoot string) error {
	// #nosec G204 - Git command with controlled arguments
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = projectRoot
	return cmd.Run()
}
