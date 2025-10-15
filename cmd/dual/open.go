package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [service]",
	Short: "Open a service directory in VS Code",
	Long: `Opens the service directory in VS Code.

If no service is specified, dual will attempt to auto-detect the service
based on your current working directory.

Examples:
  dual open www    # Open www service directory in VS Code
  dual open        # Auto-detect service and open`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Determine service name
	var serviceName string
	if len(args) > 0 {
		// Service specified explicitly
		serviceName = args[0]
		// Validate service exists in config
		if _, exists := cfg.Services[serviceName]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", serviceName, getServiceNames(cfg))
		}
	} else {
		// Auto-detect service
		serviceName, err = service.DetectService(cfg, projectRoot)
		if err != nil {
			if errors.Is(err, service.ErrServiceNotDetected) {
				return fmt.Errorf("could not auto-detect service from current directory\nAvailable services: %v\nHint: Run this command from within a service directory or specify the service name", getServiceNames(cfg))
			}
			return fmt.Errorf("failed to detect service: %w", err)
		}
	}

	// Get service path
	svc := cfg.Services[serviceName]
	servicePath := filepath.Join(projectRoot, svc.Path)

	// Print message
	fmt.Printf("[dual] Opening %s in VS Code\n", servicePath)

	// Open in VS Code
	if err := openVSCode(servicePath); err != nil {
		return fmt.Errorf("failed to open VS Code: %w", err)
	}

	return nil
}

// openVSCode opens a directory in VS Code
func openVSCode(path string) error {
	// #nosec G204 - Command execution with user-provided path is intentional
	cmd := exec.Command("code", path)
	return cmd.Run()
}
