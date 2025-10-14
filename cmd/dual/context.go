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
	"github.com/lightfastai/dual/internal/service"
	"github.com/lightfastai/dual/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	basePort      int
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
	Long: `Create a new development context in the registry with the specified name and base port.

If no name is provided, the context name will be auto-detected from the current git branch,
or fall back to "default" if not in a git repository.

If no base port is provided, an available port will be automatically assigned.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runContextCreate,
}

var contextListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts for the current project",
	Long: `List all development contexts for the current project.

By default, lists contexts with their base ports and creation dates.
Use --json for machine-readable output.
Use --ports to show calculated ports for each service.
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
	contextCreateCmd.Flags().IntVar(&basePort, "base-port", 0, "Base port for the context (auto-assigned if not specified)")

	// List flags
	contextListCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
	contextListCmd.Flags().BoolVar(&listPorts, "ports", false, "Show calculated ports for each service")
	contextListCmd.Flags().BoolVar(&contextAll, "all", false, "Include contexts from all projects")

	// Delete flags
	contextDeleteCmd.Flags().BoolVarP(&contextDelete, "force", "f", false, "Skip confirmation prompt")

	contextCmd.AddCommand(contextCreateCmd)
	contextCmd.AddCommand(contextListCmd)
	contextCmd.AddCommand(contextDeleteCmd)
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
	defer reg.Close()

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

func runContextList(cmd *cobra.Command, args []string) error {
	// Load registry
	reg, err := registry.LoadRegistry()
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

	// Print header based on flags
	if listPorts {
		fmt.Fprintln(w, "NAME\tBASE PORT\tCREATED\tPORTS\tCURRENT")
	} else {
		fmt.Fprintln(w, "NAME\tBASE PORT\tCREATED\tCURRENT")
	}

	// Print each context
	for _, name := range names {
		ctx := contexts[name]
		currentMarker := ""
		if name == currentContext {
			currentMarker = "(current)"
		}

		createdDate := ctx.Created.Format("2006-01-02")

		if listPorts {
			// Load config to calculate ports
			cfg, _, err := config.LoadConfig()
			if err != nil {
				// If config load fails, skip port calculation
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", name, ctx.BasePort, createdDate, "N/A", currentMarker)
				continue
			}

			ports, err := service.CalculateAllPorts(cfg, reg, projectIdentifier, name)
			if err != nil {
				// If port calculation fails, skip
				fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", name, ctx.BasePort, createdDate, "N/A", currentMarker)
				continue
			}

			// Format ports as "service:port,..."
			portStrs := make([]string, 0, len(ports))
			for svc, port := range ports {
				portStrs = append(portStrs, fmt.Sprintf("%s:%d", svc, port))
			}
			sort.Strings(portStrs)
			portsStr := strings.Join(portStrs, ", ")

			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", name, ctx.BasePort, createdDate, portsStr, currentMarker)
		} else {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\n", name, ctx.BasePort, createdDate, currentMarker)
		}
	}

	return w.Flush()
}

func outputContextsJSON(reg *registry.Registry, projectIdentifier, currentContext string, contexts map[string]registry.Context) error {
	type contextJSON struct {
		Name     string         `json:"name"`
		BasePort int            `json:"basePort"`
		Created  string         `json:"created"`
		Ports    map[string]int `json:"ports,omitempty"`
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
			Name:     name,
			BasePort: ctx.BasePort,
			Created:  ctx.Created.Format("2006-01-02T15:04:05Z"),
		}

		// Add ports if requested
		if listPorts {
			cfg, _, err := config.LoadConfig()
			if err == nil {
				ports, err := service.CalculateAllPorts(cfg, reg, projectIdentifier, name)
				if err == nil {
					ctxJSON.Ports = ports
				}
			}
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
		Name     string `json:"name"`
		BasePort int    `json:"basePort"`
		Created  string `json:"created"`
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
			contextList = append(contextList, contextJSON{
				Name:     name,
				BasePort: ctx.BasePort,
				Created:  ctx.Created.Format("2006-01-02T15:04:05Z"),
			})
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

	// Load registry
	reg, err := registry.LoadRegistry()
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
	fmt.Printf("  Base Port: %d\n", ctx.BasePort)
	if len(ctx.EnvOverrides) > 0 {
		fmt.Printf("  Environment Overrides: %d\n", len(ctx.EnvOverrides))
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
