package errors

import (
	"fmt"
	"strings"
)

// ConfigNotFound returns an error for when the config file is not found
func ConfigNotFound() *Error {
	return New(ErrConfigNotFound, "Configuration file not found").
		WithFixes(
			"Run 'dual init' to create a new configuration",
			"Make sure you're in the correct project directory",
		)
}

// ConfigInvalid returns an error for invalid config file
func ConfigInvalid(reason string, cause error) *Error {
	err := New(ErrConfigInvalid, "Configuration file is invalid").
		WithContext("Reason", reason).
		WithFixes(
			"Check the YAML syntax in dual.config.yml",
			"See documentation for correct schema",
		)
	if cause != nil {
		err = err.WithCause(cause)
	}
	return err
}

// ConfigExists returns an error when trying to init but config already exists
func ConfigExists(path string) *Error {
	return New(ErrConfigExists, "Configuration already exists").
		WithContext("File", path).
		WithFixes(
			"Use --force to overwrite existing config",
			"Or edit dual.config.yml manually",
		)
}

// RegistryCorrupted returns an error for corrupted registry file
func RegistryCorrupted(path string, cause error) *Error {
	err := New(ErrRegistryCorrupted, "Registry file is corrupted").
		WithContext("File", path).
		WithFixes(
			"The registry will be reset automatically",
			"You may need to recreate contexts with 'dual context create'",
		)
	if cause != nil {
		err = err.WithCause(cause)
	}
	return err
}

// ContextNotFound returns an error when a context is not registered
func ContextNotFound(contextName string, projectRoot string) *Error {
	err := New(ErrContextNotFound, fmt.Sprintf("Context '%s' not registered", contextName)).
		WithContext("Context", contextName).
		WithContext("Project", projectRoot)

	// Add specific fixes based on the context
	if contextName == "default" {
		err = err.WithFixes(
			"Run: dual context create default --base-port 4000",
			"Or create a context for your current branch",
		)
	} else {
		err = err.WithFixes(
			fmt.Sprintf("Run: dual context create %s --base-port <port>", contextName),
			"Or: dual context create (auto-assigns port)",
			"View all contexts: dual context list",
		)
	}

	return err
}

// ServiceNotFoundInConfig returns an error when a service is not in config
func ServiceNotFoundInConfig(serviceName string, availableServices []string) *Error {
	err := New(ErrServiceNotFound, fmt.Sprintf("Service '%s' not found in dual.config.yml", serviceName)).
		WithContext("Service", serviceName)

	if len(availableServices) > 0 {
		err = err.WithContext("Available services", strings.Join(availableServices, ", "))

		// Try to suggest similar service names
		for _, svc := range availableServices {
			if strings.Contains(strings.ToLower(svc), strings.ToLower(serviceName)) ||
				strings.Contains(strings.ToLower(serviceName), strings.ToLower(svc)) {
				err = err.WithFix(fmt.Sprintf("Did you mean '%s'?", svc))
			}
		}
	}

	err = err.WithFix(fmt.Sprintf("Run 'dual service add %s --path <path>' to register it", serviceName))

	return err
}

// ServiceNotDetected returns an error when auto-detection fails
func ServiceNotDetected(currentDir string, serviceNames []string, servicePaths []string) *Error {
	err := New(ErrServiceNotDetected, "Could not auto-detect service").
		WithContext("Current directory", currentDir)

	// Show available services
	if len(serviceNames) > 0 && len(servicePaths) > 0 {
		var services []string
		for i, name := range serviceNames {
			if i < len(servicePaths) {
				services = append(services, fmt.Sprintf("%s (%s)", name, servicePaths[i]))
			}
		}
		err = err.WithContext("Registered services", strings.Join(services, ", "))
	}

	err = err.WithFixes(
		"cd into a service directory",
		"Or specify service explicitly: dual --service <name> <command>",
		"Or register this as a service: dual service add <name> --path <path>",
	)

	return err
}

// PortConflict returns an error when a port is already in use
func PortConflict(port int, processInfo string) *Error {
	err := New(ErrPortConflict, fmt.Sprintf("Port %d already in use", port)).
		WithContext("Port", fmt.Sprintf("%d", port))

	if processInfo != "" {
		err = err.WithContext("Process", processInfo)
	}

	err = err.WithFixes(
		"Kill the process using this port",
		"Use different base port: dual context create --base-port <port>",
		"Or wait for the process to finish",
	)

	return err
}

// CommandFailed returns an error when command execution fails
func CommandFailed(cmd string, exitCode int, stderr string) *Error {
	err := New(ErrCommandFailed, fmt.Sprintf("Command '%s' failed", cmd)).
		WithContext("Command", cmd).
		WithContext("Exit code", fmt.Sprintf("%d", exitCode))

	if stderr != "" {
		// Limit stderr to reasonable length
		if len(stderr) > 500 {
			stderr = stderr[:500] + "..."
		}
		err = err.WithContext("Error output", stderr)
	}

	err = err.WithFixes(
		"Check the error output above",
		"Ensure the command is correct and dependencies are installed",
	)

	return err
}

// EnvNotFound returns an error when environment file is not found
func EnvNotFound(path string, isWorktree bool) *Error {
	err := New(ErrEnvNotFound, "Environment file not found").
		WithContext("Expected path", path)

	if isWorktree {
		err = err.WithFixes(
			"Ensure base environment file exists in the main repository",
			"Check that the worktree parent repository is accessible",
			"Or configure env.baseFile in dual.config.yml",
		)
	} else {
		err = err.WithFixes(
			"Create the environment file at the expected path",
			"Or configure env.baseFile in dual.config.yml",
			"Environment file is optional - this is just a warning",
		)
	}

	return err
}

// EnvParseFailed returns an error when env file parsing fails
func EnvParseFailed(path string, line int, cause error) *Error {
	err := New(ErrEnvParseFailed, "Failed to parse environment file").
		WithContext("File", path)

	if line > 0 {
		err = err.WithContext("Line", fmt.Sprintf("%d", line))
	}

	if cause != nil {
		err = err.WithCause(cause)
	}

	err = err.WithFixes(
		"Check the file syntax (should be KEY=value format)",
		"Ensure no invalid characters or malformed lines",
	)

	return err
}

// ProjectRootNotFound returns an error when project root cannot be determined
func ProjectRootNotFound(currentDir string) *Error {
	return New(ErrProjectRootNotFound, "Could not find project root").
		WithContext("Current directory", currentDir).
		WithFixes(
			"Ensure you're inside a git repository",
			"Or ensure dual.config.yml exists in a parent directory",
			"Run 'dual init' in your project root",
		)
}

// WorktreeDetectionFailed returns an error when worktree detection fails
func WorktreeDetectionFailed(cause error) *Error {
	err := New(ErrWorktreeDetectionFailed, "Failed to detect git worktree status")

	if cause != nil {
		err = err.WithCause(cause)
	}

	err = err.WithFixes(
		"Ensure you're in a valid git repository or worktree",
		"Check that the .git file/directory is not corrupted",
	)

	return err
}

// PermissionDenied returns an error for permission issues
func PermissionDenied(path string, operation string, cause error) *Error {
	err := New(ErrPermissionDenied, fmt.Sprintf("Permission denied: %s", operation)).
		WithContext("Path", path)

	if cause != nil {
		err = err.WithCause(cause)
	}

	err = err.WithFixes(
		"Check file/directory permissions",
		"Ensure you have access to the path",
	)

	return err
}
