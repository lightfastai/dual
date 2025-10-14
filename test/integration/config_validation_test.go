package integration

import (
	"path/filepath"
	"testing"
)

// TestConfigValidationInvalidVersion tests config validation with invalid version
func TestConfigValidationInvalidVersion(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// Create config with invalid version
	invalidConfig := `version: 99
services: {}
`
	h.WriteFile("dual.config.yml", invalidConfig)

	// Try to use dual - should fail with version error
	stdout, stderr, exitCode := h.RunDual("service", "add", "test", "--path", "apps/test")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "unsupported config version 99")
}

// TestConfigValidationMissingVersion tests config validation with missing version
func TestConfigValidationMissingVersion(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// Create config without version
	invalidConfig := `services: {}
`
	h.WriteFile("dual.config.yml", invalidConfig)

	// Try to use dual - should fail
	stdout, stderr, exitCode := h.RunDual("service", "add", "test", "--path", "apps/test")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "version field is required")
}

// TestConfigValidationAbsolutePath tests that absolute paths are rejected
func TestConfigValidationAbsolutePath(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	// Try to add service with absolute path
	stdout, stderr, exitCode := h.RunDual("service", "add", "test", "--path", "/absolute/path")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "path must be relative to project root")
	h.AssertOutputContains(stderr, "absolute path")
}

// TestConfigValidationNonExistentPath tests that non-existent paths are rejected
func TestConfigValidationNonExistentPath(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	// Try to add service with non-existent path
	stdout, stderr, exitCode := h.RunDual("service", "add", "test", "--path", "apps/nonexistent")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "path does not exist")
}

// TestConfigValidationFileNotDirectory tests that file paths are rejected
func TestConfigValidationFileNotDirectory(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	// Create a file instead of directory
	h.WriteFile("apps/test.txt", "not a directory")

	// Try to add service with file path
	stdout, stderr, exitCode := h.RunDual("service", "add", "test", "--path", "apps/test.txt")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "path must be a directory")
}

// TestConfigValidationEnvFileAbsolutePath tests that absolute env file paths are rejected
func TestConfigValidationEnvFileAbsolutePath(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	h.CreateDirectory("apps/web")

	// Try to add service with absolute env file path
	stdout, stderr, exitCode := h.RunDual("service", "add", "web", "--path", "apps/web", "--env-file", "/absolute/path/.env")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "env-file must be relative to project root")
	h.AssertOutputContains(stdout+stderr, "absolute path")
}

// TestConfigValidationEnvFileNonExistentDirectory tests env file directory validation
func TestConfigValidationEnvFileNonExistentDirectory(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	h.CreateDirectory("apps/web")

	// Try to add service with env file in non-existent directory
	stdout, stderr, exitCode := h.RunDual("service", "add", "web", "--path", "apps/web", "--env-file", "nonexistent/dir/.env")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "env-file directory does not exist")
}

// TestConfigValidationEmptyServiceName tests that empty service names are rejected
func TestConfigValidationEmptyServiceName(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	h.CreateDirectory("apps/web")

	// Try to add service with empty name
	stdout, stderr, exitCode := h.RunDual("service", "add", "", "--path", "apps/web")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	// Should get an error about service name
	h.AssertOutputContains(stdout+stderr, "service name cannot be empty")
}

// TestConfigValidationDuplicateService tests that duplicate service names are rejected
func TestConfigValidationDuplicateService(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/web2")

	// Add first service
	stdout, stderr, exitCode := h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Try to add service with same name
	stdout, stderr, exitCode = h.RunDual("service", "add", "web", "--path", "apps/web2")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "service \"web\" already exists")
}

// TestConfigValidationEmptyServices tests that empty services config is valid
func TestConfigValidationEmptyServices(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")

	// Init creates empty services
	stdout, stderr, exitCode := h.RunDual("init")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Verify config is valid with empty services
	h.AssertFileContains("dual.config.yml", "version: 1")
	h.AssertFileContains("dual.config.yml", "services: {}")

	// Context creation should work with empty services
	stdout, stderr, exitCode = h.RunDual("context", "create", "main", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
}

// TestConfigValidationMalformedYAML tests handling of malformed YAML
func TestConfigValidationMalformedYAML(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// Create malformed YAML config
	malformedConfig := `version: 1
services:
  web:
    path: apps/web
    invalid indentation
`
	h.WriteFile("dual.config.yml", malformedConfig)

	// Try to use dual - should fail with parse error
	stdout, stderr, exitCode := h.RunDual("service", "add", "api", "--path", "apps/api")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "failed to parse")
}

// TestConfigNotFound tests behavior when config is not found
func TestConfigNotFound(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Don't create config, try to use dual
	stdout, stderr, exitCode := h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "failed to load config")
	h.AssertOutputContains(stderr, "dual init")
}

// TestConfigSearchUpDirectory tests that config is found in parent directories
func TestConfigSearchUpDirectory(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create nested directory structure
	h.CreateDirectory("apps/web/src/components")
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Query port from deeply nested directory - should find config in parent
	stdout, stderr, exitCode := h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web/src/components"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101")
}

// TestConfigValidationRelativePathNormalization tests path normalization
func TestConfigValidationRelativePathNormalization(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.RunDual("init")

	h.CreateDirectory("apps/web")

	// Add service with path containing ".."
	stdout, stderr, exitCode := h.RunDual("service", "add", "web", "--path", "apps/../apps/web")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Verify service was added
	h.AssertFileContains("dual.config.yml", "web:")
}

// TestConfigValidationServicePathOverlap tests overlapping service paths
func TestConfigValidationServicePathOverlap(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create overlapping directory structure
	h.CreateDirectory("apps")
	h.CreateDirectory("apps/web")

	// Add both as services (should succeed - overlapping is allowed)
	stdout, stderr, exitCode := h.RunDual("service", "add", "apps", "--path", "apps")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	stdout, stderr, exitCode = h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Test longest match - from apps/web should match "web" service
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4102") // web (alphabetically second)

	// From apps should match "apps" service
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101") // apps (alphabetically first)
}

// TestContextValidationInvalidNames tests context name validation
func TestContextValidationInvalidNames(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Test context with slashes (should work - git branches can have slashes)
	stdout, stderr, exitCode := h.RunDual("context", "create", "feature/test", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test context with spaces (should work - treated as separate args)
	// Note: This will only use the first word before the space
	stdout, stderr, exitCode = h.RunDual("context", "create", "test-context", "--base-port", "4200")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
}

// TestContextNotRegistered tests error messages when context is not registered
func TestContextNotRegistered(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("unregistered-branch")
	h.RunDual("init")

	h.CreateDirectory("apps/web")
	h.RunDual("service", "add", "web", "--path", "apps/web")

	// Try to query port without creating context
	stdout, stderr, exitCode := h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "context \"unregistered-branch\" not found")
	h.AssertOutputContains(stderr, "dual context create")
}

// TestServiceNotInConfig tests error when service is not in config
func TestServiceNotInConfig(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	h.CreateDirectory("apps/web")
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Try to use --service with non-existent service
	stdout, stderr, exitCode := h.RunDual("--service", "api", "port")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "service \"api\" not found")
	h.AssertOutputContains(stderr, "Available services:")
	h.AssertOutputContains(stderr, "web")
}

// TestConfigWithSpecialCharacters tests service names with special characters
func TestConfigWithSpecialCharacters(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Test service names with hyphens, underscores
	testServices := []string{
		"web-app",
		"api_server",
		"worker-service",
	}

	for _, serviceName := range testServices {
		h.CreateDirectory("apps/" + serviceName)
		stdout, stderr, exitCode := h.RunDual("service", "add", serviceName, "--path", "apps/"+serviceName)
		h.AssertExitCode(exitCode, 0, stdout+stderr)
	}

	// Create context and verify all services work
	h.RunDual("context", "create", "main", "--base-port", "4100")

	stdout, stderr, exitCode := h.RunDual("ports")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	for _, serviceName := range testServices {
		h.AssertOutputContains(stdout, serviceName)
	}
}
