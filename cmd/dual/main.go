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

	// Allow unknown flags to be passed through to wrapped commands
	rootCmd.FParseErrWhitelist.UnknownFlags = true
}

// runCommandWrapper executes an arbitrary command with PORT environment variable injected
func runCommandWrapper(args []string) error {
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
	if serviceOverride != "" {
		// Use --service flag override
		serviceName = serviceOverride
		// Validate service exists in config
		if _, exists := cfg.Services[serviceName]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", serviceName, getServiceNames(cfg))
		}
	} else {
		// Auto-detect service
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
	reg, err := registry.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	// Calculate port
	port, err := service.CalculatePort(cfg, reg, projectIdentifier, contextName, serviceName)
	if err != nil {
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
		ctx = &registry.Context{EnvOverrides: make(map[string]string)}
	}

	// Load layered environment
	layeredEnv, err := env.LoadLayeredEnv(projectRoot, cfg, contextName, ctx.EnvOverrides, port)
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

	// Find first non-flag argument and check for --service
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		// Parse --service flag
		switch {
		case arg == "--service" && i+1 < len(os.Args):
			serviceOverride = os.Args[i+1]
			i++ // Skip the next arg
			shouldPassthrough = true
		case strings.HasPrefix(arg, "--service="):
			serviceOverride = strings.TrimPrefix(arg, "--service=")
			shouldPassthrough = true
		case !strings.HasPrefix(arg, "-") && firstNonFlagArg == "":
			firstNonFlagArg = arg
		}
	}

	// If we found --service flag, we're in passthrough mode
	// Or if first non-flag arg is not a known command
	if firstNonFlagArg != "" {
		isKnownCommand := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == firstNonFlagArg || cmd.HasAlias(firstNonFlagArg) {
				isKnownCommand = true
				break
			}
		}
		if !isKnownCommand {
			shouldPassthrough = true
		}
	}

	if shouldPassthrough && firstNonFlagArg != "" {
		// Build args for wrapped command (without --service flag)
		var wrappedArgs []string
		skipNext := false

		for i := 1; i < len(os.Args); i++ {
			arg := os.Args[i]

			if skipNext {
				skipNext = false
				continue
			}

			if arg == "--service" {
				skipNext = true
				continue
			} else if strings.HasPrefix(arg, "--service=") {
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
