package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/env"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [command] [args...]",
	Short: "Run a command with full environment injection (base + service + overrides)",
	Long: `Run a command with complete environment variable injection.

The run command loads environment variables from multiple sources and injects them
into the command execution in priority order (lowest to highest):

  1. Base environment (.env.base if configured)
  2. Service-specific environment (<service-path>/.env)
  3. Context-specific overrides (.dual/.local/service/<service>/.env)

This enables running services with isolated environments per worktree without
requiring applications to load dotenv files manually.

Examples:
  # Run Node.js server with environment
  dual run node server.js

  # Run npm start with full environment
  dual run npm start

  # Run Python server
  dual run python app.py

  # Explicitly specify service
  dual run --service api node server.js`,
	RunE:               runCommand,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
}

var runServiceName string

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVar(&runServiceName, "service", "", "Explicitly specify service name (auto-detected if not provided)")
}

func runCommand(cmd *cobra.Command, args []string) error {
	// Load config (finds project root automatically)
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Detect current service if not explicitly specified
	serviceName := runServiceName
	if serviceName == "" {
		detector := service.NewDetector()
		detectedService, err := detector.DetectService(cfg, projectRoot)
		if err != nil {
			return fmt.Errorf("failed to detect service (use --service flag to specify): %w", err)
		}
		serviceName = detectedService
	}

	// Validate service exists in config
	if _, exists := cfg.Services[serviceName]; !exists {
		return fmt.Errorf("service %q not found in config", serviceName)
	}

	// Detect current context
	ctxDetector := context.NewDetector()
	ctxName, err := ctxDetector.DetectContext()
	if err != nil {
		return fmt.Errorf("failed to detect context: %w", err)
	}

	// Use the unified LoadLayeredEnv function to load all three layers
	// Note: We don't pass overrides from registry here, letting LoadLayeredEnv
	// load them from the filesystem if they exist
	layeredEnv, err := env.LoadLayeredEnv(projectRoot, cfg, serviceName, ctxName, nil)
	if err != nil {
		return fmt.Errorf("failed to load layered environment: %w", err)
	}

	// Merge all layers
	mergedEnv := layeredEnv.Merge()

	// Build environment for exec
	execEnv := buildExecEnv(mergedEnv)

	// Prepare command
	command := args[0]
	commandArgs := args[1:]

	// Execute command with injected environment
	execCmd := exec.Command(command, commandArgs...)
	execCmd.Env = execEnv
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	fmt.Fprintf(os.Stderr, "[dual] Running: %s %v\n", command, commandArgs)
	fmt.Fprintf(os.Stderr, "[dual] Service: %s\n", serviceName)
	fmt.Fprintf(os.Stderr, "[dual] Context: %s\n", ctxName)
	fmt.Fprintf(os.Stderr, "[dual] Environment variables loaded: %d\n\n", len(mergedEnv))

	// Run command and return exit code
	if err := execCmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("command execution failed: %w", err)
	}

	return nil
}

// buildExecEnv creates the environment slice for exec.Command
func buildExecEnv(mergedEnv map[string]string) []string {
	// Start with current process environment
	execEnv := os.Environ()

	// Create a map of current env for override tracking
	currentEnv := make(map[string]bool)
	for _, envVar := range execEnv {
		// Extract key from KEY=value format
		for i := 0; i < len(envVar); i++ {
			if envVar[i] == '=' {
				key := envVar[:i]
				currentEnv[key] = true
				break
			}
		}
	}

	// Add/override with merged environment
	for key, value := range mergedEnv {
		if currentEnv[key] {
			// Override existing environment variable
			for i, envVar := range execEnv {
				for j := 0; j < len(envVar); j++ {
					if envVar[j] == '=' {
						if envVar[:j] == key {
							execEnv[i] = fmt.Sprintf("%s=%s", key, value)
							break
						}
					}
				}
			}
		} else {
			// Add new environment variable
			execEnv = append(execEnv, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return execEnv
}
