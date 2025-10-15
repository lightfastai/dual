package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync PORT values to service env files",
	Long: `Writes the PORT environment variable to each service's env file.

This is a fallback mechanism for environments where the command wrapper
cannot be used. The sync command reads the current context, calculates
the port for each service, and writes it to the service's envFile.

Example:
  dual sync    # Write PORT to all service env files`,
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

	// Detect context
	contextName, err := context.DetectContext()
	if err != nil {
		return fmt.Errorf("failed to detect context: %w", err)
	}

	// Get the normalized project identifier for registry operations
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry (using projectIdentifier so worktrees share the parent repo's registry)
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Calculate ports for all services
	ports, err := service.CalculateAllPorts(cfg, reg, projectIdentifier, contextName)
	if err != nil {
		if errors.Is(err, service.ErrContextNotFound) {
			return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
		}
		return fmt.Errorf("failed to calculate ports: %w", err)
	}

	// Update each service's env file
	updatedCount := 0
	skippedCount := 0

	for serviceName, port := range ports {
		svc := cfg.Services[serviceName]

		// Skip if no envFile configured
		if svc.EnvFile == "" {
			fmt.Printf("[dual] Skipped %s (no envFile configured)\n", serviceName)
			skippedCount++
			continue
		}

		// Get absolute path to env file
		envFilePath := filepath.Join(projectRoot, svc.EnvFile)

		// Update env file
		if err := updateEnvFile(envFilePath, port); err != nil {
			return fmt.Errorf("failed to update %s: %w", serviceName, err)
		}

		fmt.Printf("[dual] Updated %s → PORT=%d in %s\n", serviceName, port, svc.EnvFile)
		updatedCount++
	}

	// Summary
	fmt.Printf("\n[dual] Sync complete: %d updated, %d skipped\n", updatedCount, skippedCount)

	return nil
}

// updateEnvFile reads an env file, updates or adds the PORT variable, and writes it back atomically
func updateEnvFile(filePath string, port int) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Read existing file if it exists
	var lines []string
	portUpdated := false

	if _, err := os.Stat(filePath); err == nil {
		// File exists, read it
		// #nosec G304 - File path is controlled by config
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer func() { _ = file.Close() }()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			// Check if this line sets PORT
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "PORT=") {
				// Update the PORT value
				lines = append(lines, fmt.Sprintf("PORT=%d", port))
				portUpdated = true
			} else {
				// Keep the line as-is
				lines = append(lines, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	// If PORT wasn't found in the file, add it
	if !portUpdated {
		lines = append(lines, fmt.Sprintf("PORT=%d", port))
	}

	// Write to temporary file
	tempFile := filePath + ".tmp"
	// #nosec G304 - File path is controlled by config
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			_ = file.Close()
			_ = os.Remove(tempFile)
			return fmt.Errorf("failed to write to temporary file: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		_ = file.Close()
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to flush temporary file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, filePath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}
