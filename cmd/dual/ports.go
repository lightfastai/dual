package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var portsJSON bool

var portsCmd = &cobra.Command{
	Use:   "ports",
	Short: "List all service ports for the current context",
	Long: `List all configured services and their assigned ports for the current context.

By default, output is formatted as a human-readable table.
Use --json for machine-readable output.

Examples:
  dual ports           # Show all ports in table format
  dual ports --json    # Output as JSON`,
	Args: cobra.NoArgs,
	RunE: runPorts,
}

func init() {
	portsCmd.Flags().BoolVar(&portsJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(portsCmd)
}

func runPorts(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Check if there are any services configured
	if len(cfg.Services) == 0 {
		return fmt.Errorf("no services configured in dual.config.yml\nHint: Run 'dual service add <name> --path <path>' to add a service")
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
	ctx, err := reg.GetContext(projectRoot, contextName)
	if err != nil {
		if errors.Is(err, registry.ErrContextNotFound) || errors.Is(err, registry.ErrProjectNotFound) {
			return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
		}
		return fmt.Errorf("failed to get context: %w", err)
	}

	// Calculate all ports
	ports, err := service.CalculateAllPorts(cfg, reg, projectRoot, contextName)
	if err != nil {
		return fmt.Errorf("failed to calculate ports: %w", err)
	}

	// Output
	if portsJSON {
		return outputPortsJSON(contextName, ctx.BasePort, ports)
	}
	return outputPortsTable(contextName, ctx.BasePort, ports)
}

// outputPortsTable prints ports in a human-readable table format
func outputPortsTable(contextName string, basePort int, ports map[string]int) error {
	fmt.Printf("Context: %s (base: %d)\n", contextName, basePort)

	// Get sorted service names for consistent output
	serviceNames := make([]string, 0, len(ports))
	for name := range ports {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	// Find the longest service name for alignment
	maxNameLen := 0
	for _, name := range serviceNames {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	// Print each service with its port
	for _, name := range serviceNames {
		fmt.Printf("%-*s  %d\n", maxNameLen, name+":", ports[name])
	}

	return nil
}

// outputPortsJSON prints ports in JSON format
func outputPortsJSON(contextName string, basePort int, ports map[string]int) error {
	output := map[string]interface{}{
		"context":  contextName,
		"basePort": basePort,
		"ports":    ports,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
