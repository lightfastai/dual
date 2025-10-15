package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/env"
	"github.com/lightfastai/dual/internal/hooks"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var (
	createFromRef string
)

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

	// Validate that we're running from the project root
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

	// Get the normalized project identifier for registry operations
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry (using projectIdentifier so worktrees share the parent repo's registry)
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Check if context already exists
	if reg.ContextExists(projectIdentifier, branchName) {
		return fmt.Errorf("context %q already exists\nHint: Use a different branch name or delete the existing context first", branchName)
	}

	// Determine worktree path
	worktreesBasePath := cfg.GetWorktreePath(projectRoot)
	worktreeName := cfg.GetWorktreeName(branchName)
	worktreePath := filepath.Join(worktreesBasePath, worktreeName)

	// Check if worktree directory already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree directory already exists: %s\nHint: Remove it manually or use a different branch name", worktreePath)
	}

	// Create worktrees base directory if it doesn't exist
	if err := os.MkdirAll(worktreesBasePath, 0o755); err != nil {
		return fmt.Errorf("failed to create worktrees directory: %w", err)
	}

	// Build git worktree add command
	gitArgs := []string{"worktree", "add"}

	// Add branch flag
	if createFromRef == "" {
		// Create new branch from current HEAD
		gitArgs = append(gitArgs, "-b", branchName)
	} else {
		// Create new branch from specific ref
		gitArgs = append(gitArgs, "-b", branchName, worktreePath, createFromRef)
		// We've already added all args, so we'll handle this case specially
	}

	// Add worktree path
	if createFromRef == "" {
		gitArgs = append(gitArgs, worktreePath)
	}

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
	gitCmd.Stderr = os.Stderr

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create git worktree: %w\nHint: Make sure you're in a git repository and the branch name is valid", err)
	}

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

	fmt.Fprintf(os.Stderr, "[dual] Created context: %s\n", branchName)

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
		// Don't rollback - the worktree is created and might be useful
		envOverrides = hooks.NewEnvOverrides() // Use empty overrides
	}

	// Apply environment overrides to registry
	if !envOverrides.IsEmpty() {
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

	fmt.Fprintf(os.Stderr, "\n[dual] Worktree created successfully!\n")
	fmt.Fprintf(os.Stderr, "  Context: %s\n", branchName)
	fmt.Fprintf(os.Stderr, "  Path: %s\n", worktreePath)

	fmt.Fprintf(os.Stderr, "\nTo switch to this worktree:\n")
	fmt.Fprintf(os.Stderr, "  cd %s\n", worktreePath)

	return nil
}

// removeGitWorktree removes a git worktree
func removeGitWorktree(worktreePath, projectRoot string) error {
	// #nosec G204 - Git command with controlled arguments
	cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = projectRoot
	return cmd.Run()
}
