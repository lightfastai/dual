package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	contextJSON   bool
	contextAll    bool
	contextDelete bool
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
	Long: `Create a new development context in the registry with the specified name.

If no name is provided, the context name will be auto-detected from the current git branch,
or fall back to "default" if not in a git repository.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runContextCreate,
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts for the current project",
	Long: `List all development contexts for the current project.

By default, lists contexts with their creation dates.
Use --json for machine-readable output.
Use --all to show contexts from all projects.`,
	RunE: runContextList,
}

var contextDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a context from the registry",
	Long: `Delete a development context from the registry.

By default, prompts for confirmation before deleting.
Use --force to skip the confirmation prompt.
Cannot delete the current context.`,
	Args: cobra.ExactArgs(1),
	RunE: runContextDelete,
}

func init() {
	contextCmd.Flags().BoolVar(&contextJSON, "json", false, "Output as JSON")

	// List flags
	contextListCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	contextListCmd.Flags().BoolVar(&contextAll, "all", false, "Include contexts from all projects")

	// Delete flags
	contextDeleteCmd.Flags().BoolVarP(&contextDelete, "force", "f", false, "Skip confirmation prompt")

	contextCmd.AddCommand(contextCreateCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextDeleteCmd)
	rootCmd.AddCommand(contextCmd)

	// Register completion function for context delete command
	contextDeleteCmd.ValidArgsFunction = contextCompletion
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
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

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
	if ctx.Path != "" {
		fmt.Printf("Path: %s\n", ctx.Path)
	}
	fmt.Printf("Created: %s\n", ctx.Created.Format("2006-01-02 15:04:05"))
	return nil
}

// outputContextJSON prints context info in JSON format
func outputContextJSON(contextName string, ctx *registry.Context) error {
	output := map[string]interface{}{
		"name":    contextName,
		"created": ctx.Created.Format("2006-01-02T15:04:05Z"),
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

	// Load registry (using projectIdentifier so worktrees share the parent repo's registry)
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Check if context already exists
	if reg.ContextExists(projectIdentifier, contextName) {
		return fmt.Errorf("context %q already exists for this project\nUse a different name or delete the existing context first", contextName)
	}

	// Get current path (for worktree support)
	currentPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Set context in registry
	if err := reg.SetContext(projectIdentifier, contextName, currentPath); err != nil {
		return fmt.Errorf("failed to set context: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("[dual] Created context %q\n", contextName)
	fmt.Printf("  Project: %s\n", projectIdentifier)
	fmt.Printf("  Path: %s\n", currentPath)

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

func runContextList(cmd *cobra.Command, args []string) error {
	// Get project root first
	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w\nHint: Make sure you're in a git repository or have a dual.config.yml file", err)
	}

	// Get the normalized project identifier for loading the registry
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

	if contextAll {
		// List contexts from all projects
		return listAllProjectContexts(reg)
	}

	// List contexts for current project only
	return listCurrentProjectContexts(reg)
}

func listAllProjectContexts(reg *registry.Registry) error {
	projects := reg.GetAllProjects()

	if len(projects) == 0 {
		fmt.Println("No projects found in registry")
		return nil
	}

	if listJSON {
		return outputAllProjectsJSON(reg, projects)
	}

	// Human-readable output for all projects
	totalContexts := 0
	for _, projectPath := range projects {
		contexts, err := reg.ListContexts(projectPath)
		if err != nil {
			continue
		}

		fmt.Printf("\nProject: %s\n", projectPath)
		if err := outputContextsTable(reg, projectPath, contexts, ""); err != nil {
			return err
		}
		totalContexts += len(contexts)
	}

	fmt.Printf("\nTotal: %d contexts across %d projects\n", totalContexts, len(projects))
	return nil
}

func listCurrentProjectContexts(reg *registry.Registry) error {
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

	// Detect current context
	currentContext, err := context.DetectContext()
	if err != nil {
		currentContext = "" // Ignore error, just won't mark as current
	}

	// Get contexts for this project
	contexts, err := reg.ListContexts(projectIdentifier)
	if err != nil {
		if errors.Is(err, registry.ErrProjectNotFound) {
			fmt.Printf("No contexts found for project: %s\n", projectIdentifier)
			fmt.Println("\nHint: Run 'dual context create' to create a context")
			return nil
		}
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	if len(contexts) == 0 {
		fmt.Printf("No contexts found for project: %s\n", projectIdentifier)
		fmt.Println("\nHint: Run 'dual context create' to create a context")
		return nil
	}

	if listJSON {
		return outputContextsJSON(reg, projectIdentifier, currentContext, contexts)
	}

	// Human-readable output
	fmt.Printf("Contexts for %s:\n", projectIdentifier)
	if err := outputContextsTable(reg, projectIdentifier, contexts, currentContext); err != nil {
		return err
	}

	fmt.Printf("\nTotal: %d contexts\n", len(contexts))
	return nil
}

func outputContextsTable(reg *registry.Registry, projectIdentifier string, contexts map[string]registry.Context, currentContext string) error {
	// Sort context names
	names := make([]string, 0, len(contexts))
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintln(w, "NAME\tCREATED\tCURRENT")

	// Print each context
	for _, name := range names {
		ctx := contexts[name]
		currentMarker := ""
		if name == currentContext {
			currentMarker = "(current)"
		}

		createdDate := ctx.Created.Format("2006-01-02")
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, createdDate, currentMarker)
	}

	return w.Flush()
}

func outputContextsJSON(reg *registry.Registry, projectIdentifier, currentContext string, contexts map[string]registry.Context) error {
	type contextJSON struct {
		Name    string `json:"name"`
		Created string `json:"created"`
		Path    string `json:"path,omitempty"`
	}

	output := map[string]interface{}{
		"projectRoot":    projectIdentifier,
		"currentContext": currentContext,
		"contexts":       []contextJSON{},
	}

	// Sort context names for consistent output
	names := make([]string, 0, len(contexts))
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	// Build context list
	contextList := make([]contextJSON, 0, len(contexts))
	for _, name := range names {
		ctx := contexts[name]
		ctxJSON := contextJSON{
			Name:    name,
			Created: ctx.Created.Format("2006-01-02T15:04:05Z"),
		}
		if ctx.Path != "" {
			ctxJSON.Path = ctx.Path
		}

		contextList = append(contextList, ctxJSON)
	}

	output["contexts"] = contextList

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func outputAllProjectsJSON(reg *registry.Registry, projects []string) error {
	type contextJSON struct {
		Name    string `json:"name"`
		Created string `json:"created"`
		Path    string `json:"path,omitempty"`
	}

	type projectJSON struct {
		Path     string        `json:"path"`
		Contexts []contextJSON `json:"contexts"`
	}

	output := map[string]interface{}{
		"projects": []projectJSON{},
	}

	projectList := make([]projectJSON, 0, len(projects))
	for _, projectPath := range projects {
		contexts, err := reg.ListContexts(projectPath)
		if err != nil {
			continue
		}

		// Sort context names
		names := make([]string, 0, len(contexts))
		for name := range contexts {
			names = append(names, name)
		}
		sort.Strings(names)

		// Build context list
		contextList := make([]contextJSON, 0, len(contexts))
		for _, name := range names {
			ctx := contexts[name]
			ctxJSON := contextJSON{
				Name:    name,
				Created: ctx.Created.Format("2006-01-02T15:04:05Z"),
			}
			if ctx.Path != "" {
				ctxJSON.Path = ctx.Path
			}
			contextList = append(contextList, ctxJSON)
		}

		projectList = append(projectList, projectJSON{
			Path:     projectPath,
			Contexts: contextList,
		})
	}

	output["projects"] = projectList

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runContextDelete(cmd *cobra.Command, args []string) error {
	contextName := args[0]

	// Get project root
	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w\nHint: Make sure you're in a git repository or have a dual.config.yml file", err)
	}

	// Get the normalized project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Detect current context
	currentContext, err := context.DetectContext()
	if err != nil {
		currentContext = ""
	}

	// Prevent deleting current context
	if contextName == currentContext {
		return fmt.Errorf("cannot delete current context %q\nHint: Switch to a different branch or context first", contextName)
	}

	// Load registry (using projectIdentifier so worktrees share the parent repo's registry)
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Check if context exists
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		if errors.Is(err, registry.ErrContextNotFound) || errors.Is(err, registry.ErrProjectNotFound) {
			return fmt.Errorf("context %q not found for this project", contextName)
		}
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Show what will be deleted
	fmt.Printf("About to delete context: %s\n", contextName)
	fmt.Printf("  Project: %s\n", projectIdentifier)
	if ctx.Path != "" {
		fmt.Printf("  Path: %s\n", ctx.Path)
	}
	// Count all env overrides (deprecated + v2)
	overrideCount := len(ctx.EnvOverrides)
	if ctx.EnvOverridesV2 != nil {
		overrideCount += len(ctx.EnvOverridesV2.Global)
		for _, svcOverrides := range ctx.EnvOverridesV2.Services {
			overrideCount += len(svcOverrides)
		}
	}
	if overrideCount > 0 {
		fmt.Printf("  Environment Overrides: %d\n", overrideCount)
	}

	// Confirm deletion unless --force
	if !contextDelete {
		fmt.Print("\nAre you sure you want to delete this context? (y/N): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete the context
	if err := reg.DeleteContext(projectIdentifier, contextName); err != nil {
		return fmt.Errorf("failed to delete context: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	fmt.Printf("[dual] Deleted context %q\n", contextName)
	return nil
}
