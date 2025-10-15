package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/hooks"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var (
	createBasePort int
	createFromRef  string
)

var createCmd = &cobra.Command{
	Use:   "create <branch-name>",
	Short: "Create a new worktree with dual context",
	Long: `Create a new git worktree with an integrated dual context.

This command:
1. Creates a git worktree at the configured location
2. Registers a new dual context with automatic port assignment
3. Runs lifecycle hooks (postWorktreeCreate, postPortAssign)

Examples:
  dual create feature-auth              # Create worktree for feature-auth branch
  dual create feature-api --base-port 4300  # Create with specific base port
  dual create hotfix-123 --from main    # Create from specific ref`,
	Args: cobra.ExactArgs(1),
	RunE: runCreate,
}

func init() {
	createCmd.Flags().IntVar(&createBasePort, "base-port", 0, "Base port for the context (auto-assigned if not specified)")
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
		existingContext, _ := reg.GetContext(projectIdentifier, branchName)
		return fmt.Errorf("context %q already exists with base port %d\nHint: Use a different branch name or delete the existing context first", branchName, existingContext.BasePort)
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

	// Auto-assign base port if not specified
	if createBasePort == 0 {
		createBasePort = reg.FindNextAvailablePort()
		fmt.Fprintf(os.Stderr, "[dual] Auto-assigned base port: %d\n", createBasePort)
	}

	// Validate base port
	if createBasePort < 1024 || createBasePort > 65535 {
		// Cleanup: remove the worktree we just created
		_ = removeGitWorktree(worktreePath, projectRoot)
		return fmt.Errorf("base port must be between 1024 and 65535, got %d", createBasePort)
	}

	// Create context in registry
	if err := reg.SetContext(projectIdentifier, branchName, createBasePort, worktreePath); err != nil {
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

	fmt.Fprintf(os.Stderr, "[dual] Created context: %s (base port: %d)\n", branchName, createBasePort)

	// Calculate service ports for hook context
	servicePorts, err := service.CalculateAllPorts(cfg, reg, projectIdentifier, branchName)
	if err != nil {
		// Non-fatal: continue without service ports
		fmt.Fprintf(os.Stderr, "[dual] Warning: could not calculate service ports: %v\n", err)
		servicePorts = make(map[string]int)
	}

	// Prepare hook context
	hookCtx := hooks.HookContext{
		ContextName:  branchName,
		ContextPath:  worktreePath,
		ProjectRoot:  projectRoot,
		BasePort:     createBasePort,
		ServicePorts: servicePorts,
	}

	// Create hook manager
	hookMgr := hooks.NewManager(cfg, projectRoot)

	// Run postWorktreeCreate hooks
	hookCtx.Event = hooks.PostWorktreeCreate
	if err := hookMgr.Execute(hooks.PostWorktreeCreate, hookCtx); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: postWorktreeCreate hook failed: %v\n", err)
		fmt.Fprintf(os.Stderr, "[dual] Worktree created but hooks failed. You may need to run setup manually.\n")
		// Don't rollback - the worktree is created and might be useful
	}

	// Run postPortAssign hooks
	hookCtx.Event = hooks.PostPortAssign
	if err := hookMgr.Execute(hooks.PostPortAssign, hookCtx); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: postPortAssign hook failed: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "\n[dual] Worktree created successfully!\n")
	fmt.Fprintf(os.Stderr, "  Context: %s\n", branchName)
	fmt.Fprintf(os.Stderr, "  Path: %s\n", worktreePath)
	fmt.Fprintf(os.Stderr, "  Base Port: %d\n", createBasePort)

	if len(servicePorts) > 0 {
		fmt.Fprintf(os.Stderr, "\nService Ports:\n")
		// Sort service names for consistent output
		serviceNames := make([]string, 0, len(servicePorts))
		for svcName := range servicePorts {
			serviceNames = append(serviceNames, svcName)
		}
		sort.Strings(serviceNames)
		for _, svcName := range serviceNames {
			fmt.Fprintf(os.Stderr, "  %s: %d\n", svcName, servicePorts[svcName])
		}
	}

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
