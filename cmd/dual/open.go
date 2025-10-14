package main

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [service]",
	Short: "Open a service in the browser",
	Long: `Opens the URL for a service in the default browser.

If no service is specified, dual will attempt to auto-detect the service
based on your current working directory.

Examples:
  dual open www    # Open www service in browser
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

	// Detect context
	contextName, err := context.DetectContext()
	if err != nil {
		return fmt.Errorf("failed to detect context: %w", err)
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

	// Load registry
	reg, err := registry.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Calculate port
	port, err := service.CalculatePort(cfg, reg, projectRoot, contextName, serviceName)
	if err != nil {
		if errors.Is(err, service.ErrContextNotFound) {
			return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
		}
		return fmt.Errorf("failed to calculate port: %w", err)
	}

	// Construct URL
	url := fmt.Sprintf("http://localhost:%d", port)

	// Print message
	fmt.Printf("[dual] Opening %s\n", url)

	// Open browser
	if err := openBrowser(url); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// openBrowser opens a URL in the default browser (cross-platform)
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try xdg-open first, fall back to sensible-browser
		cmd = exec.Command("xdg-open", url)
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("sensible-browser", url)
		} else {
			return nil
		}
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Run()
}
