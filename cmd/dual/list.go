package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/worktree"
	"github.com/spf13/cobra"
)

var (
	listOutputJSON bool
	listAll        bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all contexts for the current project",
	Long: `List all development contexts for the current project.

By default, lists contexts with their creation dates.
Use --json for machine-readable output.
Use --all to show contexts from all projects.

Examples:
  dual list              # List contexts for current project
  dual list --json       # Output as JSON
  dual list --all        # Show contexts from all projects`,
	RunE: runList,
}

func init() {
	listCmd.Flags().BoolVar(&listOutputJSON, "json", false, "Output as JSON")
	listCmd.Flags().BoolVar(&listAll, "all", false, "Include contexts from all projects")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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

	// Load registry (using projectIdentifier to ensure worktrees access parent repo's registry)
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	if listAll {
		// List contexts from all projects
		return listAllProjectContexts(reg)
	}

	// List contexts for current project only
	return listCurrentProjectContexts(reg, projectIdentifier)
}

func listAllProjectContexts(reg *registry.Registry) error {
	projects := reg.GetAllProjects()

	if len(projects) == 0 {
		fmt.Println("No projects found in registry")
		return nil
	}

	if listOutputJSON {
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

func listCurrentProjectContexts(reg *registry.Registry, projectIdentifier string) error {
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
			fmt.Println("\nHint: Run 'dual create <branch>' to create a worktree with a context")
			return nil
		}
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	if len(contexts) == 0 {
		fmt.Printf("No contexts found for project: %s\n", projectIdentifier)
		fmt.Println("\nHint: Run 'dual create <branch>' to create a worktree with a context")
		return nil
	}

	if listOutputJSON {
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
