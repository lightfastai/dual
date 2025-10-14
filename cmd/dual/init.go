package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lightfastai/dual/internal/config"
	"github.com/spf13/cobra"
)

var forceInit bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new dual configuration",
	Long: `Creates a new dual.config.yml file in the current directory with an empty services configuration.

If a configuration file already exists, use --force to overwrite it.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&forceInit, "force", false, "Overwrite existing configuration file")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	configPath := filepath.Join(cwd, config.ConfigFileName)

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		if !forceInit {
			return fmt.Errorf("configuration file already exists at %s\nUse --force to overwrite", configPath)
		}
		fmt.Printf("[dual] Overwriting existing configuration at %s\n", configPath)
	}

	// Create template config
	templateConfig := &config.Config{
		Version:  config.SupportedVersion,
		Services: make(map[string]config.Service),
	}

	// Save the config
	if err := config.SaveConfig(templateConfig, configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("[dual] Initialized configuration at %s\n", configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Add services with: dual service add <name> --path <path>")
	fmt.Println("  2. Create a context with: dual context create")
	fmt.Println("  3. Run your commands with: dual <command>")

	return nil
}
