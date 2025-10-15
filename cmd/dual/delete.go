package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/env"
	"github.com/lightfastai/dual/internal/hooks"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var deleteForce bool

var deleteCmd = &cobra.Command{
	Use:   "delete <context-name>",
	Short: "Delete a worktree and its dual context",
	Long: `Delete a git worktree and its associated dual context.

This command:
1. Runs pre-delete hooks (preWorktreeDelete)
2. Removes the dual context from the registry
3. Removes the git worktree
4. Runs post-delete hooks (postWorktreeDelete)

By default, prompts for confirmation before deleting.
Cannot delete the currently active context.

Examples:
  dual delete feature-auth         # Delete worktree with confirmation
  dual delete feature-api --force  # Delete without confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteForce, "force", "f", false, "Skip confirmation prompt")
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	contextName := args[0]

	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Get the normalized project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Detect current context
	currentContext, err := context.DetectContext()
	if err != nil {
		// Non-fatal: just can't check if deleting current context
		currentContext = ""
	}

	// Prevent deleting current context
	if contextName == currentContext {
		return fmt.Errorf("cannot delete current context %q\nHint: Switch to a different branch or worktree first", contextName)
	}

	// Load registry (using projectRoot to construct the correct registry file path)
	reg, err := registry.LoadRegistry(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Get context info
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		if errors.Is(err, registry.ErrContextNotFound) || errors.Is(err, registry.ErrProjectNotFound) {
			return fmt.Errorf("context %q not found\nHint: Run 'dual list' to see available contexts", contextName)
		}
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Show what will be deleted
	fmt.Fprintf(os.Stderr, "About to delete worktree:\n")
	fmt.Fprintf(os.Stderr, "  Context: %s\n", contextName)
	fmt.Fprintf(os.Stderr, "  Path: %s\n", ctx.Path)

	// Confirm deletion unless --force
	if !deleteForce {
		fmt.Fprintf(os.Stderr, "\nAre you sure you want to delete this worktree? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Fprintf(os.Stderr, "[dual] Deletion cancelled\n")
			return nil
		}
	}

	// Prepare hook context
	hookCtx := hooks.HookContext{
		Event:       hooks.PreWorktreeDelete,
		ContextName: contextName,
		ContextPath: ctx.Path,
		ProjectRoot: projectRoot,
	}

	// Create hook manager
	hookMgr := hooks.NewManager(cfg, projectRoot)

	// Run preWorktreeDelete hooks
	// Note: We ignore env overrides for preWorktreeDelete since the worktree is being deleted
	_, err = hookMgr.Execute(hooks.PreWorktreeDelete, hookCtx)
	if err != nil {
		return fmt.Errorf("preWorktreeDelete hook failed: %w\nHint: Fix the hook error or use --force to skip", err)
	}

	// Cleanup service env files before deleting context
	if err := env.CleanupServiceEnvFiles(projectRoot); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to cleanup service env files: %v\n", err)
		// Don't fail the command - continue with deletion
	}

	// Delete context from registry
	if err := reg.DeleteContext(projectIdentifier, contextName); err != nil {
		return fmt.Errorf("failed to delete context from registry: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[dual] Deleted context from registry\n")

	// Remove git worktree
	if ctx.Path != "" {
		fmt.Fprintf(os.Stderr, "[dual] Removing git worktree...\n")

		// #nosec G204 - Git command with controlled arguments
		gitCmd := exec.Command("git", "worktree", "remove", ctx.Path, "--force")
		gitCmd.Dir = projectRoot
		gitCmd.Stdout = os.Stdout
		gitCmd.Stderr = os.Stderr

		if err := gitCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "[dual] Warning: failed to remove git worktree: %v\n", err)
			fmt.Fprintf(os.Stderr, "[dual] You may need to remove it manually: %s\n", ctx.Path)
			// Continue anyway - context is already deleted from registry
		} else {
			fmt.Fprintf(os.Stderr, "[dual] Removed git worktree\n")
		}
	}

	// Run postWorktreeDelete hooks (non-fatal - worktree already deleted)
	hookCtx.Event = hooks.PostWorktreeDelete
	hookMgr.ExecuteWithFallback(hooks.PostWorktreeDelete, hookCtx)

	fmt.Fprintf(os.Stderr, "\n[dual] Worktree deleted successfully!\n")
	fmt.Fprintf(os.Stderr, "  Context: %s\n", contextName)

	return nil
}
