package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/env"
	"github.com/lightfastai/dual/internal/logger"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/spf13/cobra"
)

// CLI entry point for the dual tool

var (
	// Version information - will be set via ldflags during build
	version = "dev"
	commit  = "none"
	date    = "unknown"
	// Global flag for service override
	serviceOverride string
	// Global flags for logging
	verboseFlag bool
	debugFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "dual",
	Short: "Manage port assignments across development contexts",
	Long: `dual is a CLI tool that manages port assignments across different
development contexts (git branches, worktrees, or clones). It eliminates
port conflicts when working on multiple features simultaneously by
automatically detecting the context and service, then injecting the
appropriate PORT environment variable.`,
	Version: version,
	// Disable default behavior for unknown commands
	SilenceErrors: true,
	// Handle unknown commands as passthrough
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no args, show help
		if len(args) == 0 {
			return cmd.Help()
		}

		// This is a command passthrough - execute it with PORT env var
		return runCommandWrapper(args)
	},
	// Don't show usage on errors from wrapped commands
	SilenceUsage: true,
}

func init() {
	// Custom version template that includes commit and build date
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}
Commit: {{.Annotations.commit}}
Built: {{.Annotations.date}}
`)

	// Set annotations for version info
	if rootCmd.Annotations == nil {
		rootCmd.Annotations = make(map[string]string)
	}
	rootCmd.Annotations["commit"] = commit
	rootCmd.Annotations["date"] = date

	// Add version flag (cobra adds this automatically, but we ensure it's there)
	rootCmd.Flags().BoolP("version", "v", false, "version for dual")

	// Add global --service flag for command wrapper
	rootCmd.PersistentFlags().StringVar(&serviceOverride, "service", "", "override service detection")

	// Add global logging flags
	rootCmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "debug output (includes verbose)")

	// Allow unknown flags to be passed through to wrapped commands
	rootCmd.FParseErrWhitelist.UnknownFlags = true
}

// runCommandWrapper executes an arbitrary command with PORT environment variable injected
func runCommandWrapper(args []string) error {
	// Initialize logger
	logger.Init(verboseFlag, debugFlag)

	// Load config
	logger.Verbose("Loading configuration...")
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}
	logger.Debug("Config: %s", projectRoot)

	// Get service names for debug output
	serviceNames := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		serviceNames = append(serviceNames, name)
	}
	logger.Debug("Services: %d (%v)", len(cfg.Services), serviceNames)

	// Detect context
	logger.Verbose("Detecting context...")
	contextName, err := context.DetectContext()
	if err != nil {
		return fmt.Errorf("failed to detect context: %w", err)
	}

	// Determine service name
	var serviceName string
	if serviceOverride != "" {
		// Use --service flag override
		logger.Verbose("Using service override: %s", serviceOverride)
		serviceName = serviceOverride
		// Validate service exists in config
		if _, exists := cfg.Services[serviceName]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", serviceName, getServiceNames(cfg))
		}
	} else {
		// Auto-detect service
		logger.Verbose("Detecting service...")
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
	logger.Debug("Loading registry...")
	reg, err := registry.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Calculate port
	logger.Verbose("Calculating port...")
	port, err := service.CalculatePort(cfg, reg, projectIdentifier, contextName, serviceName)
	if err != nil {
		_ = reg.Close()
		if errors.Is(err, service.ErrContextNotFound) {
			return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
		}
		return fmt.Errorf("failed to calculate port: %w", err)
	}

	// Get context from registry to load environment overrides
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		// This shouldn't happen since we just calculated the port successfully
		// But handle it gracefully
		ctx = &registry.Context{}
	}

	// Get environment overrides for the detected service (merges global + service-specific)
	overrides := ctx.GetEnvOverrides(serviceName)

	// Load layered environment
	layeredEnv, err := env.LoadLayeredEnv(projectRoot, cfg, contextName, overrides, port)
	if err != nil {
		// Non-fatal: warn but continue with just PORT
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to load environment: %v\n", err)
		layeredEnv = &env.LayeredEnv{
			Base:      make(map[string]string),
			Overrides: make(map[string]string),
			Runtime:   map[string]string{"PORT": fmt.Sprintf("%d", port)},
		}
	}

	// Get environment stats
	stats := layeredEnv.Stats()

	// Print info message with environment stats
	if stats.BaseVars > 0 || stats.OverrideVars > 0 {
		fmt.Fprintf(os.Stderr, "[dual] Context: %s | Service: %s | Port: %d\n", contextName, serviceName, port)
		fmt.Fprintf(os.Stderr, "[dual] Env: base=%d overrides=%d total=%d\n", stats.BaseVars, stats.OverrideVars, stats.TotalVars)
	} else {
		fmt.Fprintf(os.Stderr, "[dual] Context: %s | Service: %s | Port: %d\n", contextName, serviceName, port)
	}

	// Prepare command
	cmdName := args[0]
	cmdArgs := args[1:]

	logger.Verbose("Executing command: %s %v", cmdName, cmdArgs)
	logger.Debug("Environment: %d variables total", stats.TotalVars)

	// #nosec G204 - Command name and args are controlled by dual's logic
	cmd := exec.Command(cmdName, cmdArgs...)

	// Set environment variables - start with existing, then add layered env
	cmd.Env = append(os.Environ(), layeredEnv.ToSlice()...)

	// Stream output in real-time
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Execute command
	err = cmd.Run()

	// Close registry before exiting to avoid exitAfterDefer lint error
	if closeErr := reg.Close(); closeErr != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to close registry: %v\n", closeErr)
	}

	if err != nil {
		// Check if it's an exit error with a specific code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		// Other errors (command not found, etc.)
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}

// nolint:gocyclo // Command parsing logic is inherently complex
func main() {
	// Special handling: check if we should treat this as a command passthrough
	// Look for --service flag first, or check if first non-flag arg is not a known command
	shouldPassthrough := false
	var firstNonFlagArg string

	// Find first non-flag argument and check for dual-specific flags
	hasServiceFlag := false
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Parse dual-specific flags
		switch {
		case arg == "--service" && i+1 < len(os.Args):
			serviceOverride = os.Args[i+1]
			hasServiceFlag = true
			i++ // Skip the next arg
		case strings.HasPrefix(arg, "--service="):
			serviceOverride = strings.TrimPrefix(arg, "--service=")
			hasServiceFlag = true
		case arg == "--verbose":
			verboseFlag = true
		case arg == "--debug":
			debugFlag = true
		case !strings.HasPrefix(arg, "-") && firstNonFlagArg == "":
			firstNonFlagArg = arg
		}
	}

	// Check if first non-flag arg is a known command
	// If it's NOT a known command, then it's passthrough mode
	if firstNonFlagArg != "" {
		isKnownCommand := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == firstNonFlagArg || cmd.HasAlias(firstNonFlagArg) {
				isKnownCommand = true
				// Special case: "env" command has its own --service flag for service-specific overrides
				// Reset serviceOverride so it doesn't interfere with cobra parsing of env subcommands
				if hasServiceFlag && (cmd.Name() == "env") {
					serviceOverride = ""
				}
				break
			}
		}
		if !isKnownCommand {
			shouldPassthrough = true
		}
	}

	if shouldPassthrough && firstNonFlagArg != "" {
		// Build args for wrapped command (filtering out dual-specific flags)
		var wrappedArgs []string
		skipNext := false

		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]

			if skipNext {
				skipNext = false
				continue
			}

			// Filter out dual-specific flags
			switch {
			case arg == "--service":
				skipNext = true
				continue
			case strings.HasPrefix(arg, "--service="):
				continue
			case arg == "--verbose" || arg == "--debug":
				continue
			}

			wrappedArgs = append(wrappedArgs, arg)
		}

		if err := runCommandWrapper(wrappedArgs); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Normal cobra command execution
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
