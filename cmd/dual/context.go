package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var (
	basePort int
)

var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage development contexts",
	Long:  `Create, list, or remove development contexts from the registry.`,
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
	contextCreateCmd.Flags().IntVar(&basePort, "base-port", 0, "Base port for the context (auto-assigned if not specified)")

	contextCmd.AddCommand(contextCreateCmd)
	rootCmd.AddCommand(contextCmd)
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

	// Load registry
	reg, err := registry.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Check if context already exists
	if reg.ContextExists(projectRoot, contextName) {
		existingContext, _ := reg.GetContext(projectRoot, contextName)
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
	if err := reg.SetContext(projectRoot, contextName, basePort, currentPath); err != nil {
		return fmt.Errorf("failed to set context: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("[dual] Created context %q\n", contextName)
	fmt.Printf("  Project: %s\n", projectRoot)
	fmt.Printf("  Base Port: %d\n", basePort)
	fmt.Println("\nServices will be assigned ports starting from:", basePort+1)

	return nil
}

// getProjectRoot attempts to find the project root using git or config file
func getProjectRoot() (string, error) {
	// Try git first
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err == nil {
		gitRoot := strings.TrimSpace(string(output))
		if gitRoot != "" {
			// Convert to absolute path
			absPath, err := filepath.Abs(gitRoot)
			if err == nil {
				return absPath, nil
			}
		}
	}

	// Fall back to config file search
	_, projectRoot, err := config.LoadConfig()
	if err == nil {
		return projectRoot, nil
	}

	return "", fmt.Errorf("could not determine project root via git or config file")
}
