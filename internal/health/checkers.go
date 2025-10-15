package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/context"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/lightfastai/dual/internal/service"
	"github.com/lightfastai/dual/internal/worktree"
)

// CheckerContext holds the context for running health checks
type CheckerContext struct {
	Config         *config.Config
	ProjectRoot    string
	ProjectID      string
	Registry       *registry.Registry
	CurrentContext string
	AutoFix        bool
	Verbose        bool
}

// CheckGitRepository validates that we're in a git repository
func CheckGitRepository() Check {
	check := NewCheck("Git Repository", StatusPass, "")

	// Try to run git status
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return check.
			WithStatus(StatusError).
			WithMessage("Not in a git repository or git is not installed").
			WithError(err).
			WithFixAction("Run 'git init' or navigate to a git repository")
	}

	gitDir := strings.TrimSpace(string(output))
	return check.WithMessage(fmt.Sprintf("Valid git repository (git-dir: %s)", gitDir))
}

// CheckConfigFile validates the configuration file
func CheckConfigFile(ctx *CheckerContext) Check {
	check := NewCheck("Configuration File", StatusPass, "")

	if ctx.Config == nil {
		return check.
			WithStatus(StatusError).
			WithMessage(fmt.Sprintf("No %s found", config.ConfigFileName)).
			WithFixAction("Run 'dual init' to create a configuration file")
	}

	configPath := filepath.Join(ctx.ProjectRoot, config.ConfigFileName)
	details := []string{
		fmt.Sprintf("Location: %s", configPath),
		fmt.Sprintf("Version: %d", ctx.Config.Version),
		fmt.Sprintf("Services: %d", len(ctx.Config.Services)),
	}

	// Validate config version
	if ctx.Config.Version != config.SupportedVersion {
		return check.
			WithStatus(StatusError).
			WithMessage(fmt.Sprintf("Unsupported config version %d (expected %d)", ctx.Config.Version, config.SupportedVersion)).
			WithDetails(details...)
	}

	// Warn if no services
	if len(ctx.Config.Services) == 0 {
		return check.
			WithStatus(StatusWarn).
			WithMessage("Configuration is valid but no services defined").
			WithDetails(details...).
			WithFixAction("Run 'dual service add <name> --path <path>' to add services")
	}

	return check.
		WithMessage(fmt.Sprintf("Valid configuration with %d service(s)", len(ctx.Config.Services))).
		WithDetails(details...)
}

// CheckRegistry validates the registry file
func CheckRegistry(ctx *CheckerContext) Check {
	check := NewCheck("Registry", StatusPass, "")

	if ctx.Registry == nil {
		return check.
			WithStatus(StatusError).
			WithMessage("Registry could not be loaded").
			WithFixAction("Delete $PROJECT_ROOT/.dual/registry.json and run 'dual create'")
	}

	// Count contexts
	totalContexts := 0
	for _, project := range ctx.Registry.Projects {
		totalContexts += len(project.Contexts)
	}

	registryPath, _ := registry.GetRegistryPath(ctx.ProjectRoot)

	// Check if registry file exists (optional - may not exist in tests)
	details := []string{
		fmt.Sprintf("Projects: %d", len(ctx.Registry.Projects)),
		fmt.Sprintf("Total contexts: %d", totalContexts),
	}

	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		// File doesn't exist yet but Registry object is valid (e.g. in tests or new setup)
		if len(ctx.Registry.Projects) == 0 {
			return check.
				WithStatus(StatusWarn).
				WithMessage("Registry file does not exist yet").
				WithDetails("Location: " + registryPath).
				WithFixAction("Run 'dual create' to initialize registry")
		}
		// Registry has data but no file - likely in a test
		return check.
			WithMessage(fmt.Sprintf("Valid registry with %d project(s) and %d context(s)", len(ctx.Registry.Projects), totalContexts)).
			WithDetails(details...)
	}

	// File exists - validate it
	details = append([]string{fmt.Sprintf("Location: %s", registryPath)}, details...)

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return check.
			WithStatus(StatusError).
			WithMessage("Failed to read registry file").
			WithError(err)
	}

	var testRegistry registry.Registry
	if err := json.Unmarshal(data, &testRegistry); err != nil {
		return check.
			WithStatus(StatusError).
			WithMessage("Registry file is corrupt (invalid JSON)").
			WithError(err).
			WithFixAction("Delete " + registryPath + " and run 'dual create'")
	}

	return check.
		WithMessage(fmt.Sprintf("Valid registry with %d project(s) and %d context(s)", len(ctx.Registry.Projects), totalContexts)).
		WithDetails(details...)
}

// CheckCurrentContext validates the current context
func CheckCurrentContext(ctx *CheckerContext) Check {
	check := NewCheck("Current Context", StatusPass, "")

	if ctx.CurrentContext == "" {
		// Try to detect
		detectedContext, err := context.DetectContext()
		if err != nil {
			return check.
				WithStatus(StatusError).
				WithMessage("Failed to detect current context").
				WithError(err)
		}
		ctx.CurrentContext = detectedContext
	}

	// Check if context exists in registry
	if ctx.Registry != nil && ctx.ProjectID != "" {
		exists := ctx.Registry.ContextExists(ctx.ProjectID, ctx.CurrentContext)
		if !exists {
			return check.
				WithStatus(StatusWarn).
				WithMessage(fmt.Sprintf("Context '%s' exists locally but not in registry", ctx.CurrentContext)).
				WithFixAction(fmt.Sprintf("Run 'dual create' to register context '%s'", ctx.CurrentContext))
		}

		// Get context details
		regCtx, err := ctx.Registry.GetContext(ctx.ProjectID, ctx.CurrentContext)
		if err == nil {
			details := []string{
				fmt.Sprintf("Name: %s", ctx.CurrentContext),
				fmt.Sprintf("Created: %s", regCtx.Created.Format("2006-01-02 15:04:05")),
			}
			if regCtx.Path != "" {
				details = append(details, fmt.Sprintf("Path: %s", regCtx.Path))
			}
			return check.
				WithMessage(fmt.Sprintf("Context '%s' is valid and registered", ctx.CurrentContext)).
				WithDetails(details...)
		}
	}

	return check.WithMessage(fmt.Sprintf("Context: %s", ctx.CurrentContext))
}

// CheckServicePaths validates that all service paths exist
func CheckServicePaths(ctx *CheckerContext) Check {
	check := NewCheck("Service Paths", StatusPass, "")

	if ctx.Config == nil || len(ctx.Config.Services) == 0 {
		return check.
			WithStatus(StatusWarn).
			WithMessage("No services configured")
	}

	invalidPaths := []string{}
	validPaths := []string{}

	for name, svc := range ctx.Config.Services {
		fullPath := filepath.Join(ctx.ProjectRoot, svc.Path)

		// Resolve symlinks
		realPath, err := filepath.EvalSymlinks(fullPath)
		if err != nil {
			invalidPaths = append(invalidPaths, fmt.Sprintf("%s: %s (error: %v)", name, svc.Path, err))
			continue
		}

		// Check if path exists
		info, err := os.Stat(realPath)
		if err != nil {
			if os.IsNotExist(err) {
				invalidPaths = append(invalidPaths, fmt.Sprintf("%s: %s (does not exist)", name, svc.Path))
			} else {
				invalidPaths = append(invalidPaths, fmt.Sprintf("%s: %s (error: %v)", name, svc.Path, err))
			}
			continue
		}

		// Check if it's a directory
		if !info.IsDir() {
			invalidPaths = append(invalidPaths, fmt.Sprintf("%s: %s (not a directory)", name, svc.Path))
			continue
		}

		validPaths = append(validPaths, fmt.Sprintf("%s: %s", name, svc.Path))
	}

	if len(invalidPaths) > 0 {
		return check.
			WithStatus(StatusError).
			WithMessage(fmt.Sprintf("%d service path(s) are invalid", len(invalidPaths))).
			WithDetails(invalidPaths...).
			WithFixAction("Update service paths in dual.config.yml or remove invalid services")
	}

	return check.
		WithMessage(fmt.Sprintf("All %d service path(s) are valid", len(validPaths))).
		WithDetails(validPaths...)
}

// CheckEnvironmentFiles validates environment files
func CheckEnvironmentFiles(ctx *CheckerContext) Check {
	check := NewCheck("Environment Files", StatusPass, "")

	if ctx.Config == nil {
		return check.WithStatus(StatusWarn).WithMessage("No configuration loaded")
	}

	var issues []string
	var validFiles []string
	hasEnvFiles := false

	// Check base env file
	if ctx.Config.Env.BaseFile != "" {
		hasEnvFiles = true
		baseFilePath := filepath.Join(ctx.ProjectRoot, ctx.Config.Env.BaseFile)
		if _, err := os.Stat(baseFilePath); os.IsNotExist(err) {
			issues = append(issues, fmt.Sprintf("Base env file not found: %s", ctx.Config.Env.BaseFile))
		} else {
			validFiles = append(validFiles, fmt.Sprintf("Base: %s", ctx.Config.Env.BaseFile))
		}
	}

	// Check service env files
	for name, svc := range ctx.Config.Services {
		if svc.EnvFile != "" {
			hasEnvFiles = true
			envFilePath := filepath.Join(ctx.ProjectRoot, svc.EnvFile)
			if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("Service '%s' env file not found: %s", name, svc.EnvFile))
			} else {
				validFiles = append(validFiles, fmt.Sprintf("%s: %s", name, svc.EnvFile))
			}
		}
	}

	if !hasEnvFiles {
		return check.
			WithStatus(StatusWarn).
			WithMessage("No environment files configured").
			WithFixAction("Add env.baseFile or service envFile in dual.config.yml if needed")
	}

	if len(issues) > 0 {
		return check.
			WithStatus(StatusWarn).
			WithMessage(fmt.Sprintf("%d environment file(s) not found", len(issues))).
			WithDetails(issues...).
			WithFixAction("Create missing .env files or update paths in dual.config.yml")
	}

	return check.
		WithMessage(fmt.Sprintf("%d environment file(s) configured and exist", len(validFiles))).
		WithDetails(validFiles...)
}


// CheckWorktrees validates worktree configuration
func CheckWorktrees(ctx *CheckerContext) Check {
	check := NewCheck("Worktrees", StatusPass, "")

	detector := worktree.NewDetector()

	// Find git root
	gitRoot, err := detector.FindGitRoot(ctx.ProjectRoot)
	if err != nil {
		// Not a git repo or can't find root - not necessarily an error for this check
		return check.
			WithStatus(StatusWarn).
			WithMessage("Not in a git repository or cannot determine git root").
			WithError(err)
	}

	// Check if this is a worktree
	isWorktree, err := detector.IsWorktree(gitRoot)
	if err != nil {
		return check.
			WithStatus(StatusWarn).
			WithMessage("Failed to determine if this is a worktree").
			WithError(err)
	}

	if !isWorktree {
		return check.WithMessage("Not a worktree (main repository)")
	}

	// Get parent repo
	parentRepo, err := detector.GetParentRepo(gitRoot)
	if err != nil {
		return check.
			WithStatus(StatusWarn).
			WithMessage("This appears to be a worktree but cannot find parent repository").
			WithError(err)
	}

	details := []string{
		fmt.Sprintf("Worktree git root: %s", gitRoot),
		fmt.Sprintf("Parent repository: %s", parentRepo),
	}

	// Check if parent repo has the config
	parentConfigPath := filepath.Join(parentRepo, config.ConfigFileName)
	if _, err := os.Stat(parentConfigPath); err == nil {
		details = append(details, fmt.Sprintf("Parent config: %s", parentConfigPath))
	}

	return check.
		WithMessage("Valid worktree configuration").
		WithDetails(details...)
}

// CheckOrphanedContexts finds contexts that no longer have valid paths
func CheckOrphanedContexts(ctx *CheckerContext) Check {
	check := NewCheck("Orphaned Contexts", StatusPass, "")

	if ctx.Registry == nil {
		return check.WithStatus(StatusWarn).WithMessage("Cannot check without registry")
	}

	var orphaned []string
	var cleaned []string

	// Check all contexts in all projects
	for projectPath, project := range ctx.Registry.Projects {
		for contextName, regCtx := range project.Contexts {
			// If context has a path set, check if it exists
			if regCtx.Path != "" {
				if _, err := os.Stat(regCtx.Path); os.IsNotExist(err) {
					orphanedEntry := fmt.Sprintf("%s:%s (path: %s)", projectPath, contextName, regCtx.Path)
					orphaned = append(orphaned, orphanedEntry)

					// Auto-fix: delete orphaned context
					if ctx.AutoFix {
						if err := ctx.Registry.DeleteContext(projectPath, contextName); err == nil {
							cleaned = append(cleaned, orphanedEntry)
						}
					}
				}
			}
		}
	}

	if ctx.AutoFix && len(cleaned) > 0 {
		// Save registry after cleanup
		if err := ctx.Registry.SaveRegistry(); err == nil {
			return check.
				WithMessage(fmt.Sprintf("Cleaned up %d orphaned context(s)", len(cleaned))).
				WithDetails(cleaned...).
				WithFixApplied()
		}
	}

	if len(orphaned) > 0 {
		return check.
			WithStatus(StatusWarn).
			WithMessage(fmt.Sprintf("Found %d orphaned context(s)", len(orphaned))).
			WithDetails(orphaned...).
			WithFixAction("Run 'dual doctor --fix' to remove orphaned contexts")
	}

	return check.WithMessage("No orphaned contexts found")
}

// CheckPermissions validates file permissions
func CheckPermissions(ctx *CheckerContext) Check {
	check := NewCheck("Permissions", StatusPass, "")

	var issues []string

	// Check registry directory permissions
	registryPath, _ := registry.GetRegistryPath(ctx.ProjectRoot)
	registryDir := filepath.Dir(registryPath)

	if info, err := os.Stat(registryDir); err == nil {
		mode := info.Mode().Perm()
		// Check if directory is readable and writable by owner
		if mode&0o600 != 0o600 {
			issues = append(issues, fmt.Sprintf("Registry directory has unexpected permissions: %o (expected at least 0600)", mode))
		}
	} else if !os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("Cannot check registry directory permissions: %v", err))
	}

	// Check registry file permissions if it exists
	if info, err := os.Stat(registryPath); err == nil {
		mode := info.Mode().Perm()
		if mode&0o600 != 0o600 {
			issues = append(issues, fmt.Sprintf("Registry file has unexpected permissions: %o (expected at least 0600)", mode))
		}
	} else if !os.IsNotExist(err) {
		issues = append(issues, fmt.Sprintf("Cannot check registry file permissions: %v", err))
	}

	// Check config file permissions
	if ctx.ProjectRoot != "" {
		configPath := filepath.Join(ctx.ProjectRoot, config.ConfigFileName)
		if info, err := os.Stat(configPath); err == nil {
			mode := info.Mode().Perm()
			if mode&0o400 != 0o400 {
				issues = append(issues, fmt.Sprintf("Config file is not readable: %o", mode))
			}
		}
	}

	if len(issues) > 0 {
		return check.
			WithStatus(StatusWarn).
			WithMessage(fmt.Sprintf("Found %d permission issue(s)", len(issues))).
			WithDetails(issues...).
			WithFixAction("Fix file permissions manually with chmod")
	}

	return check.WithMessage("All file permissions are correct")
}

// CheckServiceDetection validates service detection for current directory
func CheckServiceDetection(ctx *CheckerContext) Check {
	check := NewCheck("Service Detection", StatusPass, "")

	if ctx.Config == nil || len(ctx.Config.Services) == 0 {
		return check.
			WithStatus(StatusWarn).
			WithMessage("No services configured, cannot test detection")
	}

	// Try to detect service from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return check.
			WithStatus(StatusWarn).
			WithMessage("Cannot determine current directory").
			WithError(err)
	}

	serviceName, err := service.DetectService(ctx.Config, ctx.ProjectRoot)
	if err != nil {
		if errors.Is(err, service.ErrServiceNotDetected) {
			// This is OK - we might not be in a service directory
			return check.
				WithStatus(StatusWarn).
				WithMessage("No service detected for current directory").
				WithDetails(fmt.Sprintf("Current directory: %s", cwd)).
				WithFixAction("Navigate to a service directory or use --service flag")
		}
		return check.
			WithStatus(StatusError).
			WithMessage("Failed to detect service").
			WithError(err)
	}

	details := []string{
		fmt.Sprintf("Current directory: %s", cwd),
		fmt.Sprintf("Detected service: %s", serviceName),
	}

	return check.
		WithMessage(fmt.Sprintf("Service '%s' detected successfully", serviceName)).
		WithDetails(details...)
}

// Helper to update status
func (c Check) WithStatus(status Status) Check {
	c.Status = status
	return c
}

// WithMessage updates the message
func (c Check) WithMessage(message string) Check {
	c.Message = message
	return c
}
