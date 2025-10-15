package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/env"
	"github.com/lightfastai/dual/internal/logger"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/spf13/cobra"
)

var (
	// Flags for env commands
	envShowValues       bool
	envShowBaseOnly     bool
	envShowOverrideOnly bool
	envShowJSON         bool
	envExportFormat     string
	envServiceFlag      string // --service flag for service-specific overrides
	envVerbose          bool
	envDebug            bool
)

// getServiceNames returns a sorted list of service names from config
func getServiceNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Services))
	for name := range cfg.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage context-specific environment variables",
	Long: `Manage context-specific environment variable overrides.

Environment variables are layered in the following priority (highest to lowest):
  1. Runtime values (PORT)
  2. Context-specific overrides
  3. Base environment file

Use 'dual env set' to override variables for the current context.`,
	RunE: runEnvShow, // Default to show command
}

var envShowCmd = &cobra.Command{
	Use:     "show",
	Aliases: []string{"list"},
	Short:   "Show environment summary and overrides",
	Long: `Display environment variable summary for the current context.

Shows the base environment file path, variable counts, and context-specific overrides.

Examples:
  dual env show              # Show summary
  dual env show --values     # Show all variable values
  dual env show --base-only  # Show only base variables
  dual env show --overrides-only  # Show only overrides
  dual env show --json       # Output as JSON`,
	RunE: runEnvShow,
}

var envSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a context-specific environment override",
	Long: `Set an environment variable override for the current context.

This override will be applied whenever commands are run in this context.
The override takes precedence over the base environment file but not over PORT.

Use --service to set a service-specific override that only applies to that service.

Examples:
  dual env set DATABASE_URL "mysql://localhost/mydb"
  dual env set DEBUG "true"
  dual env set --service api DATABASE_URL "mysql://localhost/api_db"`,
	Args: cobra.ExactArgs(2),
	RunE: runEnvSet,
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a context-specific environment override",
	Long: `Remove an environment variable override for the current context.

If the variable exists in the base environment file, it will show the fallback value.

Use --service to remove a service-specific override.

Examples:
  dual env unset DATABASE_URL
  dual env unset DEBUG
  dual env unset --service api DATABASE_URL`,
	Args: cobra.ExactArgs(1),
	RunE: runEnvUnset,
}

var envExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export merged environment to stdout",
	Long: `Export the complete merged environment to stdout.

The output includes all layers merged together (base, overrides, runtime).

Examples:
  dual env export              # dotenv format
  dual env export --format=json    # JSON format
  dual env export --format=shell   # Shell export format
  dual env export > .env.local     # Save to file`,
	RunE: runEnvExport,
}

var envCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate environment configuration",
	Long: `Validate the environment configuration for the current context.

Checks:
  - Base environment file exists and is readable
  - All required variables are present
  - No conflicts or issues

Exit code:
  0 - Environment is valid
  1 - Issues found`,
	RunE: runEnvCheck,
}

var envDiffCmd = &cobra.Command{
	Use:   "diff <context1> <context2>",
	Short: "Compare environments between contexts",
	Long: `Compare environment variables between two contexts.

Shows variables that are:
  - Changed (different values)
  - Added (only in context2)
  - Removed (only in context1)

Examples:
  dual env diff main feature-auth
  dual env diff feature-a feature-b`,
	Args: cobra.ExactArgs(2),
	RunE: runEnvDiff,
}

var envRemapCmd = &cobra.Command{
	Use:   "remap",
	Short: "Regenerate service-specific .env files from registry",
	Long: `Reads environment overrides from registry and generates .dual/.local/service/<service>/.env files.

This command regenerates all service-specific environment files based on the current
registry state. This is useful if you've manually edited the registry or if the
files are out of sync.

The files are automatically generated when you use 'dual env set' or 'dual env unset',
so you typically don't need to run this command manually.

Examples:
  dual env remap    # Regenerate all service env files`,
	RunE: runEnvRemap,
}

func init() {
	rootCmd.AddCommand(envCmd)

	// Add subcommands
	envCmd.AddCommand(envShowCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envUnsetCmd)
	envCmd.AddCommand(envExportCmd)
	envCmd.AddCommand(envCheckCmd)
	envCmd.AddCommand(envDiffCmd)
	envCmd.AddCommand(envRemapCmd)

	// Flags for show command
	envShowCmd.Flags().BoolVar(&envShowValues, "values", false, "show all variable values")
	envShowCmd.Flags().BoolVar(&envShowBaseOnly, "base-only", false, "show only base variables")
	envShowCmd.Flags().BoolVar(&envShowOverrideOnly, "overrides-only", false, "show only overrides")
	envShowCmd.Flags().BoolVar(&envShowJSON, "json", false, "output as JSON")
	envShowCmd.Flags().StringVar(&envServiceFlag, "service", "", "show overrides for specific service")

	// Flags for set command
	envSetCmd.Flags().StringVar(&envServiceFlag, "service", "", "set service-specific override")

	// Flags for unset command
	envUnsetCmd.Flags().StringVar(&envServiceFlag, "service", "", "unset service-specific override")

	// Flags for export command
	envExportCmd.Flags().StringVar(&envExportFormat, "format", "dotenv", "output format (dotenv, json, shell)")
	envExportCmd.Flags().StringVar(&envServiceFlag, "service", "", "export for specific service")
}

func runEnvShow(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger.Init(envVerbose, envDebug)

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

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Get context from registry
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
	}

	// Get environment overrides for the specified service (or global if no service specified)
	overrides := ctx.GetEnvOverrides(envServiceFlag)

	// Load layered environment (without PORT since we're just showing info)
	layeredEnv, err := env.LoadLayeredEnv(projectRoot, cfg, contextName, overrides, 0)
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Remove PORT from runtime since we added it with 0
	delete(layeredEnv.Runtime, "PORT")

	// Get stats
	stats := layeredEnv.Stats()

	// Handle JSON output
	if envShowJSON {
		return outputEnvJSON(layeredEnv, cfg, contextName, stats)
	}

	// Handle different display modes
	if envShowBaseOnly {
		return showBaseOnly(layeredEnv, cfg)
	}

	if envShowOverrideOnly {
		return showOverridesOnly(layeredEnv, contextName)
	}

	// Default: show summary
	return showEnvSummary(layeredEnv, cfg, contextName, stats)
}

func showEnvSummary(layeredEnv *env.LayeredEnv, cfg *config.Config, contextName string, stats env.EnvStats) error {
	// Show base file info
	if cfg.Env.BaseFile != "" {
		fmt.Printf("Base:      %s (%d vars)\n", cfg.Env.BaseFile, stats.BaseVars)
	} else {
		fmt.Println("Base:      (none configured)")
	}

	// Show overrides count
	fmt.Printf("Overrides: %d vars\n", stats.OverrideVars)

	// Show total
	totalVars := stats.BaseVars + stats.OverrideVars
	fmt.Printf("Effective: %d vars total\n", totalVars)

	// Show overrides if any
	if stats.OverrideVars > 0 {
		fmt.Printf("\nOverrides for context '%s':\n", contextName)

		// Sort keys for consistent output
		keys := make([]string, 0, len(layeredEnv.Overrides))
		for k := range layeredEnv.Overrides {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := layeredEnv.Overrides[k]
			if envShowValues {
				fmt.Printf("  %s=%s\n", k, v)
			} else {
				// Show truncated value for security
				displayValue := v
				if len(v) > 40 {
					displayValue = v[:37] + "..."
				}
				fmt.Printf("  %s=%s\n", k, displayValue)
			}
		}
	}

	return nil
}

func showBaseOnly(layeredEnv *env.LayeredEnv, cfg *config.Config) error {
	if cfg.Env.BaseFile == "" {
		fmt.Println("No base environment file configured")
		return nil
	}

	if len(layeredEnv.Base) == 0 {
		fmt.Printf("Base file %s has no variables\n", cfg.Env.BaseFile)
		return nil
	}

	fmt.Printf("Base environment (%s):\n", cfg.Env.BaseFile)

	// Sort keys
	keys := make([]string, 0, len(layeredEnv.Base))
	for k := range layeredEnv.Base {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if envShowValues {
			fmt.Printf("%s=%s\n", k, layeredEnv.Base[k])
		} else {
			// Show key only for security
			fmt.Printf("%s\n", k)
		}
	}

	return nil
}

func showOverridesOnly(layeredEnv *env.LayeredEnv, contextName string) error {
	if len(layeredEnv.Overrides) == 0 {
		fmt.Printf("No overrides for context '%s'\n", contextName)
		return nil
	}

	fmt.Printf("Overrides for context '%s':\n", contextName)

	// Sort keys
	keys := make([]string, 0, len(layeredEnv.Overrides))
	for k := range layeredEnv.Overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if envShowValues {
			fmt.Printf("%s=%s\n", k, layeredEnv.Overrides[k])
		} else {
			fmt.Printf("%s\n", k)
		}
	}

	return nil
}

func outputEnvJSON(layeredEnv *env.LayeredEnv, cfg *config.Config, contextName string, stats env.EnvStats) error {
	output := map[string]interface{}{
		"context":  contextName,
		"baseFile": cfg.Env.BaseFile,
		"stats": map[string]int{
			"baseVars":     stats.BaseVars,
			"overrideVars": stats.OverrideVars,
			"totalVars":    stats.BaseVars + stats.OverrideVars,
		},
		"base":      layeredEnv.Base,
		"overrides": layeredEnv.Overrides,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Initialize logger
	logger.Init(envVerbose, envDebug)

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

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Check if context exists
	_, err = reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
	}

	// If service is specified, validate it exists in config
	if envServiceFlag != "" {
		if _, exists := cfg.Services[envServiceFlag]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", envServiceFlag, getServiceNames(cfg))
		}
	}

	// Check if we're overriding a base variable
	if cfg.Env.BaseFile != "" {
		loader := env.NewLoader()
		baseEnv, err := loader.LoadEnvFile(projectRoot + "/" + cfg.Env.BaseFile)
		if err == nil {
			if _, exists := baseEnv[key]; exists {
				fmt.Fprintf(os.Stderr, "[dual] Warning: Overriding variable %q from base environment\n", key)
			}
		}
	}

	// Set the override (with service if specified)
	if err := reg.SetEnvOverrideForService(projectIdentifier, contextName, key, value, envServiceFlag); err != nil {
		return fmt.Errorf("failed to set environment override: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Generate service env files
	if err := env.GenerateServiceEnvFiles(cfg, reg, projectRoot, projectIdentifier, contextName); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to regenerate service env files: %v\n", err)
		// Don't fail the command - the override is saved, env files are optional
	}

	// Show success message
	if envServiceFlag != "" {
		fmt.Printf("Set %s=%s for service '%s' in context '%s'\n", key, value, envServiceFlag, contextName)
	} else {
		fmt.Printf("Set %s=%s for context '%s' (global)\n", key, value, contextName)
	}

	// Show current override count
	ctx, _ := reg.GetContext(projectIdentifier, contextName)
	if ctx != nil {
		globalCount := 0
		serviceCount := 0
		if ctx.EnvOverridesV2 != nil {
			globalCount = len(ctx.EnvOverridesV2.Global)
			for _, serviceOverrides := range ctx.EnvOverridesV2.Services {
				serviceCount += len(serviceOverrides)
			}
		}
		totalCount := globalCount + serviceCount
		if totalCount > 0 {
			fmt.Printf("Context '%s' now has %d override(s) (%d global, %d service-specific)\n",
				contextName, totalCount, globalCount, serviceCount)
		}
	}

	return nil
}

func runEnvUnset(cmd *cobra.Command, args []string) error {
	key := args[0]

	// Initialize logger
	logger.Init(envVerbose, envDebug)

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

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Check if context exists
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
	}

	// If service is specified, validate it exists in config
	if envServiceFlag != "" {
		if _, exists := cfg.Services[envServiceFlag]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", envServiceFlag, getServiceNames(cfg))
		}
	}

	// Check if override exists
	if !ctx.HasEnvOverride(key, envServiceFlag) {
		if envServiceFlag != "" {
			return fmt.Errorf("no override found for %q in service '%s' for context '%s'", key, envServiceFlag, contextName)
		}
		return fmt.Errorf("no override found for %q in context '%s'", key, contextName)
	}

	// Unset the override
	if err := reg.UnsetEnvOverrideForService(projectIdentifier, contextName, key, envServiceFlag); err != nil {
		return fmt.Errorf("failed to unset environment override: %w", err)
	}

	// Save registry
	if err := reg.SaveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	// Generate service env files
	if err := env.GenerateServiceEnvFiles(cfg, reg, projectRoot, projectIdentifier, contextName); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: failed to regenerate service env files: %v\n", err)
		// Don't fail the command - the override is removed, env files are optional
	}

	// Show success message
	if envServiceFlag != "" {
		fmt.Printf("Removed override for %s in service '%s' for context '%s'\n", key, envServiceFlag, contextName)
	} else {
		fmt.Printf("Removed override for %s in context '%s'\n", key, contextName)
	}

	// Check if there's a fallback value in base
	if cfg.Env.BaseFile != "" {
		loader := env.NewLoader()
		baseEnv, err := loader.LoadEnvFile(projectRoot + "/" + cfg.Env.BaseFile)
		if err == nil {
			if baseValue, exists := baseEnv[key]; exists {
				fmt.Printf("Fallback to base value: %s=%s\n", key, baseValue)
			}
		}
	}

	return nil
}

func runEnvExport(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger.Init(envVerbose, envDebug)

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

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Get context from registry
	ctx, err := reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
	}

	// If service is specified, validate it exists in config
	if envServiceFlag != "" {
		if _, exists := cfg.Services[envServiceFlag]; !exists {
			return fmt.Errorf("service %q not found in config\nAvailable services: %v", envServiceFlag, getServiceNames(cfg))
		}
	}

	// Get environment overrides for the specified service (or global if no service specified)
	overrides := ctx.GetEnvOverrides(envServiceFlag)

	// Load layered environment (with PORT as placeholder - will be replaced by actual port at runtime)
	layeredEnv, err := env.LoadLayeredEnv(projectRoot, cfg, contextName, overrides, 0)
	if err != nil {
		return fmt.Errorf("failed to load environment: %w", err)
	}

	// Merge all layers
	merged := layeredEnv.Merge()

	// Sort keys for consistent output
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Output in requested format
	switch envExportFormat {
	case "dotenv":
		for _, k := range keys {
			v := merged[k]
			// Quote values that contain spaces or special characters
			if strings.ContainsAny(v, " \t\n\"'") {
				v = fmt.Sprintf(`"%s"`, strings.ReplaceAll(v, `"`, `\"`))
			}
			fmt.Printf("%s=%s\n", k, v)
		}
	case "json":
		data, err := json.MarshalIndent(merged, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	case "shell":
		for _, k := range keys {
			v := merged[k]
			// Escape single quotes for shell
			v = strings.ReplaceAll(v, `'`, `'\''`)
			fmt.Printf("export %s='%s'\n", k, v)
		}
	default:
		return fmt.Errorf("unsupported format: %s (supported: dotenv, json, shell)", envExportFormat)
	}

	return nil
}

func runEnvCheck(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger.Init(envVerbose, envDebug)

	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to load config: %v\n", err)
		return fmt.Errorf("configuration check failed")
	}

	hasIssues := false

	// Check base environment file
	if cfg.Env.BaseFile != "" {
		baseFilePath := projectRoot + "/" + cfg.Env.BaseFile
		loader := env.NewLoader()
		baseEnv, err := loader.LoadEnvFile(baseFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Base environment file (%s) is not readable: %v\n", cfg.Env.BaseFile, err)
			hasIssues = true
		} else {
			fmt.Printf("✓ Base environment file exists: %s (%d vars)\n", cfg.Env.BaseFile, len(baseEnv))
		}
	} else {
		fmt.Println("ℹ No base environment file configured")
	}

	// Check context
	contextName, err := context.DetectContext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to detect context: %v\n", err)
		hasIssues = true
	} else {
		fmt.Printf("✓ Context detected: %s\n", contextName)
	}

	// Check registry
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get project identifier: %v\n", err)
		hasIssues = true
	} else {
		reg, err := registry.LoadRegistry(projectIdentifier)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to load registry: %v\n", err)
			hasIssues = true
		} else {
			defer reg.Close()
			ctx, err := reg.GetContext(projectIdentifier, contextName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Context '%s' not found in registry\n", contextName)
				hasIssues = true
			} else {
				// Count all overrides (global + service-specific)
				globalCount := 0
				serviceCount := 0
				if ctx.EnvOverridesV2 != nil {
					globalCount = len(ctx.EnvOverridesV2.Global)
					for _, serviceOverrides := range ctx.EnvOverridesV2.Services {
						serviceCount += len(serviceOverrides)
					}
				}
				totalCount := globalCount + serviceCount
				if totalCount > 0 {
					fmt.Printf("✓ Context has %d environment override(s) (%d global, %d service-specific)\n",
						totalCount, globalCount, serviceCount)
				} else {
					fmt.Println("ℹ Context has no environment overrides")
				}
			}
		}
	}

	if hasIssues {
		fmt.Println("\n❌ Environment configuration has issues")
		return fmt.Errorf("environment configuration has issues")
	}

	fmt.Println("\n✓ Environment configuration is valid")
	return nil
}

type envDiff struct {
	changed map[string][2]string
	added   map[string]string
	removed map[string]string
}

func runEnvDiff(cmd *cobra.Command, args []string) error {
	context1 := args[0]
	context2 := args[1]

	// Initialize logger
	logger.Init(envVerbose, envDebug)

	// Load environments for both contexts
	merged1, merged2, err := loadAndMergeContextEnvs(context1, context2)
	if err != nil {
		return err
	}

	// Calculate differences
	diff := calculateEnvDiff(merged1, merged2)

	// Display results
	displayEnvDiff(context1, context2, diff)

	return nil
}

func loadAndMergeContextEnvs(context1, context2 string) (map[string]string, map[string]string, error) {
	// Load config
	cfg, projectRoot, err := config.LoadConfig()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w\nHint: Run 'dual init' to create a configuration file", err)
	}

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Get both contexts
	ctx1, err := reg.GetContext(projectIdentifier, context1)
	if err != nil {
		return nil, nil, fmt.Errorf("context %q not found in registry", context1)
	}

	ctx2, err := reg.GetContext(projectIdentifier, context2)
	if err != nil {
		return nil, nil, fmt.Errorf("context %q not found in registry", context2)
	}

	// Load environments for both contexts
	env1, err := env.LoadLayeredEnv(projectRoot, cfg, context1, ctx1.EnvOverrides, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load environment for %q: %w", context1, err)
	}

	env2, err := env.LoadLayeredEnv(projectRoot, cfg, context2, ctx2.EnvOverrides, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load environment for %q: %w", context2, err)
	}

	// Merge environments
	return env1.Merge(), env2.Merge(), nil
}

func calculateEnvDiff(merged1, merged2 map[string]string) envDiff {
	diff := envDiff{
		changed: make(map[string][2]string),
		added:   make(map[string]string),
		removed: make(map[string]string),
	}

	// Find changed and removed
	for k, v1 := range merged1 {
		if k == "PORT" {
			continue // Skip PORT comparison
		}
		if v2, exists := merged2[k]; exists {
			if v1 != v2 {
				diff.changed[k] = [2]string{v1, v2}
			}
		} else {
			diff.removed[k] = v1
		}
	}

	// Find added
	for k, v2 := range merged2 {
		if k == "PORT" {
			continue // Skip PORT comparison
		}
		if _, exists := merged1[k]; !exists {
			diff.added[k] = v2
		}
	}

	return diff
}

func displayEnvDiff(context1, context2 string, diff envDiff) {
	fmt.Printf("Comparing environments: %s → %s\n\n", context1, context2)

	if len(diff.changed) > 0 {
		displayChangedVars(diff.changed)
	}

	if len(diff.added) > 0 {
		displayAddedVars(diff.added)
	}

	if len(diff.removed) > 0 {
		displayRemovedVars(diff.removed)
	}

	if len(diff.changed) == 0 && len(diff.added) == 0 && len(diff.removed) == 0 {
		fmt.Println("No differences found")
	}
}

func displayChangedVars(changed map[string][2]string) {
	fmt.Println("Changed:")
	keys := make([]string, 0, len(changed))
	for k := range changed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vals := changed[k]
		fmt.Printf("  %s: %s → %s\n", k, vals[0], vals[1])
	}
	fmt.Println()
}

func displayAddedVars(added map[string]string) {
	fmt.Println("Added:")
	keys := make([]string, 0, len(added))
	for k := range added {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s=%s\n", k, added[k])
	}
	fmt.Println()
}

func displayRemovedVars(removed map[string]string) {
	fmt.Println("Removed:")
	keys := make([]string, 0, len(removed))
	for k := range removed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %s=%s\n", k, removed[k])
	}
	fmt.Println()
}

func runEnvRemap(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger.Init(envVerbose, envDebug)

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

	// Get project identifier
	projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
	if err != nil {
		return fmt.Errorf("failed to get project identifier: %w", err)
	}

	// Load registry
	reg, err := registry.LoadRegistry(projectIdentifier)
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}
	defer reg.Close()

	// Check if context exists
	_, err = reg.GetContext(projectIdentifier, contextName)
	if err != nil {
		return fmt.Errorf("context %q not found in registry\nHint: Run 'dual context create' to create this context", contextName)
	}

	fmt.Fprintf(os.Stderr, "[dual] Regenerating service env files for context '%s'...\n", contextName)

	// Generate service env files
	if err := env.GenerateServiceEnvFiles(cfg, reg, projectRoot, projectIdentifier, contextName); err != nil {
		return fmt.Errorf("failed to generate service env files: %w", err)
	}

	fmt.Fprintf(os.Stderr, "[dual] Service env files regenerated successfully\n")
	fmt.Fprintf(os.Stderr, "  Files written to: %s/.dual/.local/service/<service>/.env\n", projectRoot)

	return nil
}
