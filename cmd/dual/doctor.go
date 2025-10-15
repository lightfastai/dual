package main

import (
	"fmt"
	"os"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/health"
	"github.com/lightfastai/dual/internal/logger"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var (
	doctorAutoFix bool
	doctorJSON    bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run health checks and validate dual configuration",
	Long: `Run comprehensive health checks to validate dual configuration and detect issues.

The doctor command performs the following checks:
  - Git repository validation
  - Configuration file validation
  - Registry validation
  - Current context verification
  - Service paths validation
  - Environment files validation
  - Port conflict detection
  - Worktree validation
  - Orphaned context cleanup
  - File permissions check

Exit codes:
  0 - All checks passed
  1 - Some checks passed with warnings
  2 - Some checks failed with errors

Examples:
  # Run all health checks
  dual doctor

  # Run with automatic fixes
  dual doctor --fix

  # Output results as JSON for CI/automation
  dual doctor --json

  # Verbose output with detailed information
  dual doctor --verbose`,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorAutoFix, "fix", false, "Automatically fix issues where possible")
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON")
	rootCmd.AddCommand(doctorCmd)
}

//nolint:gocyclo // Health check function naturally has high complexity due to 11 sequential checks
func runDoctor(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger.Init(verboseFlag, debugFlag)

	// Create result container
	result := health.NewResult()

	// Build checker context
	ctx := &health.CheckerContext{
		AutoFix: doctorAutoFix,
		Verbose: verboseFlag || debugFlag,
	}

	// === Check 1: Git Repository ===
	if verboseFlag {
		logger.Verbose("Checking git repository...")
	}
	result.AddCheck(health.CheckGitRepository())

	// === Check 2: Configuration File ===
	if verboseFlag {
		logger.Verbose("Checking configuration file...")
	}

	cfg, projectRoot, err := config.LoadConfig()
	var projectID string
	if err != nil {
		// Config not found or invalid - still record the check
		ctx.Config = nil
		ctx.ProjectRoot = ""
		result.AddCheck(health.CheckConfigFile(ctx))
	} else {
		ctx.Config = cfg
		ctx.ProjectRoot = projectRoot

		// Get project identifier for registry operations
		projectID, err = config.GetProjectIdentifier(projectRoot)
		if err != nil {
			logger.Verbose("Warning: failed to get project identifier: %v", err)
			projectID = projectRoot // Fallback
		}
		ctx.ProjectID = projectID

		result.AddCheck(health.CheckConfigFile(ctx))
	}

	// === Check 3: Registry ===
	if verboseFlag {
		logger.Verbose("Checking registry...")
	}

	// Load registry (using projectID so worktrees share the parent repo's registry)
	// Only load if config was successfully loaded (projectID will be non-empty)
	if projectID == "" {
		// Skip registry check if config failed to load
		ctx.Registry = nil
		result.AddCheck(health.CheckRegistry(ctx))
	} else {
		reg, err := registry.LoadRegistry(projectID)
		if err != nil {
			logger.Verbose("Warning: failed to load registry: %v", err)
			ctx.Registry = nil
			result.AddCheck(health.CheckRegistry(ctx))
		} else {
			ctx.Registry = reg
			result.AddCheck(health.CheckRegistry(ctx))
		}
	}

	// === Check 4: Current Context ===
	if verboseFlag {
		logger.Verbose("Checking current context...")
	}

	currentContext, err := context.DetectContext()
	if err != nil {
		logger.Verbose("Warning: failed to detect context: %v", err)
		ctx.CurrentContext = ""
	} else {
		ctx.CurrentContext = currentContext
	}
	result.AddCheck(health.CheckCurrentContext(ctx))

	// === Check 5: Service Paths ===
	if verboseFlag {
		logger.Verbose("Checking service paths...")
	}
	result.AddCheck(health.CheckServicePaths(ctx))

	// === Check 6: Environment Files ===
	if verboseFlag {
		logger.Verbose("Checking environment files...")
	}
	result.AddCheck(health.CheckEnvironmentFiles(ctx))

	// === Check 7: Port Conflicts ===
	if verboseFlag {
		logger.Verbose("Checking for port conflicts...")
	}
	result.AddCheck(health.CheckPortConflicts(ctx))

	// === Check 8: Worktrees ===
	if verboseFlag {
		logger.Verbose("Checking worktree configuration...")
	}
	result.AddCheck(health.CheckWorktrees(ctx))

	// === Check 9: Orphaned Contexts ===
	if verboseFlag {
		logger.Verbose("Checking for orphaned contexts...")
	}
	result.AddCheck(health.CheckOrphanedContexts(ctx))

	// === Check 10: Permissions ===
	if verboseFlag {
		logger.Verbose("Checking file permissions...")
	}
	result.AddCheck(health.CheckPermissions(ctx))

	// === Check 11: Service Detection ===
	if verboseFlag {
		logger.Verbose("Checking service detection...")
	}
	result.AddCheck(health.CheckServiceDetection(ctx))

	// Close registry before exiting
	if ctx.Registry != nil {
		if err := ctx.Registry.Close(); err != nil {
			logger.Verbose("Warning: failed to close registry: %v", err)
		}
	}

	// Determine exit code
	result.ExitCode = result.DetermineExitCode()

	// Output results
	if doctorJSON {
		jsonOutput, err := result.FormatJSON()
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		fmt.Println(jsonOutput)
	} else {
		fmt.Print(result.Format(verboseFlag || debugFlag))
	}

	// Exit with appropriate code
	os.Exit(result.ExitCode)
	return nil
}
