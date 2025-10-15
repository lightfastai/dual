package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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
	RunE:              runCommand,
	Args:              cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
}

var (
	runServiceName string
)

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

	// Build layered environment manually to include service-specific .env
	layeredEnv := &env.LayeredEnv{
		Base:      make(map[string]string),
		Service:   make(map[string]string),
		Overrides: make(map[string]string),
	}

	// Load base environment from .env.base if configured
	if cfg.Env.BaseFile != "" {
		baseFilePath := filepath.Join(projectRoot, cfg.Env.BaseFile)
		baseEnv, err := env.LoadEnvFile(baseFilePath)
		if err == nil {
			layeredEnv.Base = baseEnv
		}
		// Non-fatal: if base file doesn't exist or can't be read, continue with empty base
	}

	// Load service-specific .env file
	servicePath := cfg.Services[serviceName].Path
	serviceEnvPath := filepath.Join(projectRoot, servicePath, ".env")
	serviceEnv, err := env.LoadEnvFile(serviceEnvPath)
	if err == nil {
		layeredEnv.Service = serviceEnv
	}
	// Non-fatal: if service .env doesn't exist, continue without it

	// Load context-specific overrides from .dual/.local/service/<service>/.env
	overridesPath := filepath.Join(projectRoot, ".dual", ".local", "service", serviceName, ".env")
	overridesEnv, err := env.LoadEnvFile(overridesPath)
	if err == nil {
		layeredEnv.Overrides = overridesEnv
	}
	// Non-fatal: if overrides file doesn't exist, continue without it

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
		if exitErr, ok := err.(*exec.ExitError); ok {
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
