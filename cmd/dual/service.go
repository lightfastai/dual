package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/spf13/cobra"
)

var (
	servicePath    string
	serviceEnvFile string
	// list command flags
	listJSON     bool
	listAbsPaths bool
	// remove command flags
	forceRemove bool
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services in the dual configuration",
	Long:  `Add, list, or remove services from the dual configuration.`,
}

var serviceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new service to the configuration",
	Long: `Add a new service to the dual configuration with the specified name and path.

The path should be relative to the project root (where dual.config.yml is located).
Optionally, you can specify an env file for the service using --env-file.`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceAdd,
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services in the configuration",
	Long: `List all services defined in the dual configuration.

By default, shows service name, path, and env file in a human-readable format.
Use --json for machine-readable output.
Use --paths to show absolute paths instead of relative paths.`,
	Args: cobra.NoArgs,
	RunE: runServiceList,
}

var serviceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a service from the configuration",
	Long: `Remove a service from the dual configuration.

This command does NOT delete any files or directories.`,
	Args: cobra.ExactArgs(1),
	RunE: runServiceRemove,
}

func init() {
	serviceAddCmd.Flags().StringVar(&servicePath, "path", "", "Relative path to the service directory (required)")
	serviceAddCmd.Flags().StringVar(&serviceEnvFile, "env-file", "", "Relative path to the env file for the service (optional)")
	_ = serviceAddCmd.MarkFlagRequired("path")

	serviceListCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	serviceListCmd.Flags().BoolVar(&listAbsPaths, "paths", false, "Show absolute paths instead of relative paths")

	serviceRemoveCmd.Flags().BoolVarP(&forceRemove, "force", "f", false, "Skip confirmation prompt")

	serviceCmd.AddCommand(serviceAddCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceRemoveCmd)
	rootCmd.AddCommand(serviceCmd)

	// Register completion function for service remove command
	serviceRemoveCmd.ValidArgsFunction = serviceNameCompletion
}

func runServiceAdd(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	// Validate service name
	if serviceName == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Load existing config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Check if service already exists
	if _, exists := cfg.Services[serviceName]; exists {
		return fmt.Errorf("service %q already exists in the configuration", serviceName)
	}

	// Validate that the path is not absolute
	if filepath.IsAbs(servicePath) {
		return fmt.Errorf("path must be relative to project root, got absolute path: %s", servicePath)
	}

	// Validate that the path exists
	fullPath := filepath.Join(projectRoot, servicePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s (resolved to %s)", servicePath, fullPath)
		}
		return fmt.Errorf("failed to check path: %w", err)
	}

	// Validate that the path is a directory
	if !info.IsDir() {
		return fmt.Errorf("path must be a directory, got file: %s", servicePath)
	}

	// Validate env file if provided
	if serviceEnvFile != "" {
		if filepath.IsAbs(serviceEnvFile) {
			return fmt.Errorf("env-file must be relative to project root, got absolute path: %s", serviceEnvFile)
		}

		// Check if the directory containing the env file exists
		envFileFullPath := filepath.Join(projectRoot, serviceEnvFile)
		envFileDir := filepath.Dir(envFileFullPath)
		if _, err := os.Stat(envFileDir); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("env-file directory does not exist: %s (resolved to %s)", filepath.Dir(serviceEnvFile), envFileDir)
			}
			return fmt.Errorf("failed to check env-file directory: %w", err)
		}
	}

	// Add service to config
	cfg.Services[serviceName] = config.Service{
		Path:    servicePath,
		EnvFile: serviceEnvFile,
	}

	// Save the config
	configPath := filepath.Join(projectRoot, config.ConfigFileName)
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[dual] Added service %q\n", serviceName)
	fmt.Printf("  Path: %s\n", servicePath)
	if serviceEnvFile != "" {
		fmt.Printf("  Env File: %s\n", serviceEnvFile)
	}

	return nil
}

func runServiceList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// If no services, print message and exit
	if len(cfg.Services) == 0 {
		if listJSON {
			fmt.Println(`{"services":[]}`)
		} else {
			fmt.Println("No services configured")
			fmt.Println("Run 'dual service add <name> --path <path>' to add a service")
		}
		return nil
	}

	// Get sorted service names for consistent output
	serviceNames := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	// Output in requested format
	if listJSON {
		return outputListJSON(cfg, projectRoot, serviceNames)
	}

	return outputListHuman(cfg, projectRoot, serviceNames)
}

func outputListJSON(cfg *config.Config, projectRoot string, serviceNames []string) error {
	type serviceOutput struct {
		Name         string `json:"name"`
		Path         string `json:"path"`
		EnvFile      string `json:"envFile,omitempty"`
		AbsolutePath string `json:"absolutePath,omitempty"`
	}

	output := struct {
		Services []serviceOutput `json:"services"`
	}{
		Services: make([]serviceOutput, 0, len(serviceNames)),
	}

	for _, name := range serviceNames {
		svc := cfg.Services[name]
		svcOut := serviceOutput{
			Name:    name,
			Path:    svc.Path,
			EnvFile: svc.EnvFile,
		}

		if listAbsPaths {
			svcOut.AbsolutePath = filepath.Join(projectRoot, svc.Path)
		}

		output.Services = append(output.Services, svcOut)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func outputListHuman(cfg *config.Config, projectRoot string, serviceNames []string) error {
	fmt.Println("Services in dual.config.yml:")

	// Calculate column widths for alignment
	maxNameLen := 0
	maxPathLen := 0
	for _, name := range serviceNames {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		svc := cfg.Services[name]
		pathStr := svc.Path
		if listAbsPaths {
			pathStr = filepath.Join(projectRoot, svc.Path)
		}
		if len(pathStr) > maxPathLen {
			maxPathLen = len(pathStr)
		}
	}

	// Print services
	for _, name := range serviceNames {
		svc := cfg.Services[name]
		pathStr := svc.Path
		if listAbsPaths {
			pathStr = filepath.Join(projectRoot, svc.Path)
		}

		// Format: name (padded) path (padded) [envfile]
		fmt.Printf("  %-*s  %-*s", maxNameLen, name, maxPathLen, pathStr)

		if svc.EnvFile != "" {
			fmt.Printf("  %s", svc.EnvFile)
		}
		fmt.Println()
	}

	fmt.Printf("\nTotal: %d service", len(serviceNames))
	if len(serviceNames) != 1 {
		fmt.Print("s")
	}
	fmt.Println()

	return nil
}

func runServiceRemove(cmd *cobra.Command, args []string) error {
	serviceName := args[0]

	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Check if service exists
	if _, exists := cfg.Services[serviceName]; !exists {
		return fmt.Errorf("service %q not found in configuration", serviceName)
	}

	// Remove service from config
	delete(cfg.Services, serviceName)

	// Save the config
	configPath := filepath.Join(projectRoot, config.ConfigFileName)
	if err := config.SaveConfig(cfg, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[dual] Service %q removed from config\n", serviceName)

	return nil
}

func promptConfirm(message string) bool {
	fmt.Printf("%s (y/N): ", message)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
