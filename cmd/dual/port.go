package main

import (
	"errors"
	"fmt"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var portVerbose bool

var portCmd = &cobra.Command{
	Use:   "port [service]",
	Short: "Get the port for a service",
	Long: `Get the port number for a service in the current context.

If no service is specified, dual will attempt to auto-detect the service
based on your current working directory.

Examples:
  dual port www              # Get port for www service
  dual port                  # Auto-detect service and get port
  dual port --verbose www    # Show context and service info`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPort,
}

func init() {
	portCmd.Flags().BoolVarP(&portVerbose, "verbose", "v", false, "Show context and service information")
	rootCmd.AddCommand(portCmd)
}

func runPort(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Detect context
	contextName, err := context.DetectContext()
	if err != nil {
		return fmt.Errorf("failed to detect context: %w", err)
	}

	// Determine service name
	var serviceName string
	switch {
	case serviceOverride != "":
		// Use --service flag override (global persistent flag)
		serviceName = serviceOverride
		// Validate service exists in config
		if _, exists := cfg.Services[serviceName]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", serviceName, getServiceNames(cfg))
		}
	case len(args) > 0:
		// Service specified explicitly as argument
		serviceName = args[0]
		// Validate service exists in config
		if _, exists := cfg.Services[serviceName]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", serviceName, getServiceNames(cfg))
		}
	default:
		// Auto-detect service
		serviceName, err = service.DetectService(cfg, projectRoot)
		if err != nil {
			if errors.Is(err, service.ErrServiceNotDetected) {
				return fmt.Errorf("could not auto-detect service from current directory\nAvailable services: %v\nHint: Run this command from within a service directory or use --service flag", getServiceNames(cfg))
			}
			return fmt.Errorf("failed to detect service: %w", err)
		}
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

	// Calculate port
	port, err := service.CalculatePort(cfg, reg, projectIdentifier, contextName, serviceName)
	if err != nil {
		if errors.Is(err, service.ErrContextNotFound) {
			return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
		}
		return fmt.Errorf("failed to calculate port: %w", err)
	}

	// Output
	if portVerbose {
		fmt.Printf("[dual] Context: %s | Service: %s | Port: %d\n", contextName, serviceName, port)
	} else {
		fmt.Println(port)
	}

	return nil
}

// getServiceNames returns a sorted list of service names from the config
func getServiceNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	return names
}
