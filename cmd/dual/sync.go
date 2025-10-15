package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lightfastai/dual/internal/config"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment files for services",
	Long: `Syncs environment files for all services in the configuration.

This command ensures that all service environment files exist and are properly
configured based on the service's envFile setting in dual.config.yml.

Example:
  dual sync    # Sync env files for all services`,
	Args: cobra.NoArgs,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Check if there are any services
	if len(cfg.Services) == 0 {
		return fmt.Errorf("no services configured\nHint: Run 'dual service add' to add services")
	}

	// Sync each service's env file
	syncedCount := 0
	skippedCount := 0

	for serviceName, svc := range cfg.Services {
		// Skip if no envFile configured
		if svc.EnvFile == "" {
			fmt.Printf("[dual] Skipped %s (no envFile configured)\n", serviceName)
			skippedCount++
			continue
		}

		// Get absolute path to env file
		envFilePath := filepath.Join(projectRoot, svc.EnvFile)

		// Ensure env file exists
		if err := ensureEnvFile(envFilePath); err != nil {
			return fmt.Errorf("failed to sync %s: %w", serviceName, err)
		}

		fmt.Printf("[dual] Synced %s → %s\n", serviceName, svc.EnvFile)
		syncedCount++
	}

	// Summary
	fmt.Printf("\n[dual] Sync complete: %d synced, %d skipped\n", syncedCount, skippedCount)

	return nil
}

// ensureEnvFile ensures that an env file exists, creating it if necessary
func ensureEnvFile(filePath string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		// File exists, nothing to do
		return nil
	}

	// Create empty env file
	// #nosec G304 - File path is controlled by config
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create env file: %w", err)
	}
	defer func() { _ = file.Close() }()

	return nil
}
