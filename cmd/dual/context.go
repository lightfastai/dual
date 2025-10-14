package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	basePort    int
	contextJSON bool
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage development contexts",
	Long: `Show or manage development contexts.

When called without a subcommand, displays information about the current context.

Examples:
  dual context              # Show current context info
  dual context --json       # Show current context as JSON
  dual context create       # Create a new context`,
	RunE: runContextInfo,
}

var contextCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new development context",
	Long: `Create a new development context in the registry with the specified name and base port.

If no name is provided, the context name will be auto-detected from the current git branch,
or fall back to "default" if not in a git repository.

If no base port is provided, an available port will be automatically assigned.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runContextCreate,
}

func init() {
	contextCmd.Flags().BoolVar(&contextJSON, "json", false, "Output as JSON")
	contextCreateCmd.Flags().IntVar(&basePort, "base-port", 0, "Base port for the context (auto-assigned if not specified)")

	contextCmd.AddCommand(contextCreateCmd)
	rootCmd.AddCommand(contextCmd)
}

func runContextInfo(cmd *cobra.Command, args []string) error {
	// Load config to get project root
	_, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Get the normalized project identifier for registry lookups
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Detect context
	contextName, err := context.DetectContext()
	if err != nil {
		return fmt.Errorf("failed to detect context: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Get context info
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		if errors.Is(err, registry.ErrContextNotFound) || errors.Is(err, registry.ErrProjectNotFound) {
			return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
		}
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Output
	if contextJSON {
		return outputContextJSON(contextName, ctx)
	}
	return outputContextTable(contextName, ctx)
}

// outputContextTable prints context info in human-readable format
func outputContextTable(contextName string, ctx *registry.Context) error {
	fmt.Printf("Context: %s\n", contextName)
	fmt.Printf("Base Port: %d\n", ctx.BasePort)
	if ctx.Path != "" {
		fmt.Printf("Path: %s\n", ctx.Path)
	}
	return nil
}

// outputContextJSON prints context info in JSON format
func outputContextJSON(contextName string, ctx *registry.Context) error {
	output := map[string]interface{}{
		"name":     contextName,
		"basePort": ctx.BasePort,
	}
	if ctx.Path != "" {
		output["path"] = ctx.Path
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runContextCreate(cmd *cobra.Command, args []string) error {
	// Get or detect context name
	var contextName string
	if len(args) > 0 {
		contextName = args[0]
	} else {
		// Auto-detect from git branch or use "default"
		detectedContext, err := context.DetectContext()
		if err != nil {
			// If detection fails, use "default"
			contextName = context.DefaultContext
		} else {
			contextName = detectedContext
		}
		fmt.Printf("[dual] Auto-detected context name: %s\n", contextName)
	}

	// Get project root - try git first, then config file
	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w\nHint: Make sure you're in a git repository or have a dual.config.yml file", err)
	}

	// Get the normalized project identifier for registry operations
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if context already exists
	if reg.ContextExists(projectIdentifier, contextName) {
		existingContext, _ := reg.GetContext(projectIdentifier, contextName)
		return fmt.Errorf("context %q already exists for this project with base port %d\nUse a different name or delete the existing context first", contextName, existingContext.BasePort)
	}

	// Auto-assign port if not specified
	if basePort == 0 {
		basePort = reg.FindNextAvailablePort()
		fmt.Printf("[dual] Auto-assigned base port: %d\n", basePort)
	}

	// Validate base port
	if basePort < 1024 || basePort > 65535 {
		return fmt.Errorf("base port must be between 1024 and 65535, got %d", basePort)
	}

	// Get current path (for worktree support)
	currentPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Set context in registry
	if err := reg.SetContext(projectIdentifier, contextName, basePort, currentPath); err != nil {
		return fmt.Errorf("failed to set context: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("[dual] Created context %q\n", contextName)
	fmt.Printf("  Project: %s\n", projectIdentifier)
	fmt.Printf("  Base Port: %d\n", basePort)
	fmt.Println("\nServices will be assigned ports starting from:", basePort+1)

	return nil
}

// getProjectRoot attempts to find the project root using git worktree-aware detection or config file
func getProjectRoot() (string, error) {
	// Try worktree-aware git detection first
	wtDetector := worktree.NewDetector()
	projectRoot, err := wtDetector.GetProjectRootFromCwd()
	if err == nil {
		return projectRoot, nil
	}

	// Fall back to config file search
	_, projectRoot, err = config.LoadConfig()
	if err == nil {
		return projectRoot, nil
	}

	return "", fmt.Errorf("could not determine project root via git or config file")
}
