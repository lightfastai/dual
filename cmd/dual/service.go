package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lightfastai/dual/internal/config"
	"github.com/spf13/cobra"
)

var (
	servicePath    string
	serviceEnvFile string
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

func init() {
	serviceAddCmd.Flags().StringVar(&servicePath, "path", "", "Relative path to the service directory (required)")
	serviceAddCmd.Flags().StringVar(&serviceEnvFile, "env-file", "", "Relative path to the env file for the service (optional)")
	serviceAddCmd.MarkFlagRequired("path")

	serviceCmd.AddCommand(serviceAddCmd)
	rootCmd.AddCommand(serviceCmd)
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
