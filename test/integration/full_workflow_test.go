package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFullWorkflow tests the complete workflow: init → service add → context create → run
func TestFullWorkflow(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repository
	h.InitGitRepo()

	// Create a main branch
	h.CreateGitBranch("main")

	// Step 1: dual init
	t.Log("Step 1: Running dual init")
	stdout, stderr, exitCode := h.RunDual("init")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Initialized configuration")

	// Verify config file was created
	if !h.FileExists("dual.config.yml") {
		t.Fatal("dual.config.yml was not created")
	}

	h.AssertFileContains("dual.config.yml", "version: 1")
	h.AssertFileContains("dual.config.yml", "services: {}")

	// Step 2: Add services
	t.Log("Step 2: Adding services")

	// Create service directories
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.CreateDirectory("apps/worker")

	// Add web service
	stdout, stderr, exitCode = h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"web\"")

	// Add api service
	stdout, stderr, exitCode = h.RunDual("service", "add", "api", "--path", "apps/api")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"api\"")

	// Add worker service
	stdout, stderr, exitCode = h.RunDual("service", "add", "worker", "--path", "apps/worker")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"worker\"")

	// Verify services were added to config
	h.AssertFileContains("dual.config.yml", "web:")
	h.AssertFileContains("dual.config.yml", "apps/web")
	h.AssertFileContains("dual.config.yml", "api:")
	h.AssertFileContains("dual.config.yml", "apps/api")
	h.AssertFileContains("dual.config.yml", "worker:")
	h.AssertFileContains("dual.config.yml", "apps/worker")

	// Step 3: Create context
	t.Log("Step 3: Creating context")
	stdout, stderr, exitCode = h.RunDual("context", "create", "main", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Created context \"main\"")
	h.AssertOutputContains(stdout, "Base Port: 4100")

	// Verify registry was created
	if !h.RegistryExists() {
		t.Fatal("registry.json was not created")
	}

	registryContent := h.ReadRegistryJSON()
	if !strings.Contains(registryContent, "4100") {
		t.Errorf("registry does not contain base port 4100: %s", registryContent)
	}

	// Step 4: Query context
	t.Log("Step 4: Querying context")
	stdout, stderr, exitCode = h.RunDual("context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: main")
	h.AssertOutputContains(stdout, "Base Port: 4100")

	// Step 5: Query individual port
	t.Log("Step 5: Querying individual port")
	stdout, stderr, exitCode = h.RunDualInDir(filepath.Join(h.ProjectDir, "apps/api"), "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// API should be port 4101 (alphabetically first: api=0, web=2, worker=3)
	// port = basePort (4100) + index (0) + 1 = 4101
	h.AssertOutputContains(stdout, "4101")

	// Step 6: Query all ports
	t.Log("Step 6: Querying all ports")
	stdout, stderr, exitCode = h.RunDual("ports")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "api")
	h.AssertOutputContains(stdout, "4101") // api: 4100 + 0 + 1
	h.AssertOutputContains(stdout, "web")
	h.AssertOutputContains(stdout, "4102") // web: 4100 + 1 + 1
	h.AssertOutputContains(stdout, "worker")
	h.AssertOutputContains(stdout, "4103") // worker: 4100 + 2 + 1

	// Step 7: Test command wrapper
	t.Log("Step 7: Testing command wrapper")

	// Create a simple script that prints the PORT environment variable
	scriptContent := `#!/bin/sh
echo "PORT=$PORT"
`
	h.WriteFile("print-port.sh", scriptContent)

	// Make it executable
	scriptPath := filepath.Join(h.ProjectDir, "print-port.sh")
	if err := makeExecutable(scriptPath); err != nil {
		t.Fatalf("failed to make script executable: %v", err)
	}

	// Run the script through dual from the web service directory
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"sh", filepath.Join(h.ProjectDir, "print-port.sh"),
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Context: main")
	h.AssertOutputContains(stderr, "Service: web")
	h.AssertOutputContains(stderr, "Port: 4102")
	h.AssertOutputContains(stdout, "PORT=4102")

	// Test with --service flag override
	t.Log("Step 8: Testing --service flag override")
	stdout, stderr, exitCode = h.RunDual(
		"--service", "worker",
		"sh", scriptPath,
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Service: worker")
	h.AssertOutputContains(stderr, "Port: 4103")
	h.AssertOutputContains(stdout, "PORT=4103")
}

// TestFullWorkflowWithEnvFile tests the workflow with env file configuration
func TestFullWorkflowWithEnvFile(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repository
	h.InitGitRepo()
	h.CreateGitBranch("main")

	// Initialize dual
	h.RunDual("init")

	// Create service directory with env file directory
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/web/.env")

	// Add service with env-file
	stdout, stderr, exitCode := h.RunDual(
		"service", "add", "web",
		"--path", "apps/web",
		"--env-file", "apps/web/.env/development.local",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Added service \"web\"")
	h.AssertOutputContains(stdout, "Env File: apps/web/.env/development.local")

	// Verify env file path in config
	h.AssertFileContains("dual.config.yml", "envFile: apps/web/.env/development.local")
}

// TestInitForceFlag tests the --force flag for dual init
func TestInitForceFlag(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// First init
	stdout, stderr, exitCode := h.RunDual("init")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Modify the config
	h.WriteFile("dual.config.yml", "version: 1\nservices:\n  test:\n    path: apps/test\n")

	// Try to init again without --force (should fail)
	stdout, stderr, exitCode = h.RunDual("init")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "configuration file already exists")

	// Init with --force (should succeed)
	stdout, stderr, exitCode = h.RunDual("init", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Overwriting existing configuration")

	// Verify config was reset
	configContent := h.ReadFile("dual.config.yml")
	if strings.Contains(configContent, "test:") {
		t.Error("config was not reset by --force flag")
	}
}

// TestContextAutoDetection tests automatic context name detection
func TestContextAutoDetection(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()
	h.CreateGitBranch("feature/awesome-feature")

	// Initialize dual
	h.RunDual("init")

	// Create context without specifying name (should auto-detect from git branch)
	stdout, stderr, exitCode := h.RunDual("context", "create", "--base-port", "5000")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Auto-detected context name: feature/awesome-feature")
	h.AssertOutputContains(stdout, "Created context \"feature/awesome-feature\"")

	// Query context should show the auto-detected name
	stdout, stderr, exitCode = h.RunDual("context")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Context: feature/awesome-feature")
}

// TestContextAutoPortAssignment tests automatic port assignment
func TestContextAutoPortAssignment(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	h.InitGitRepo()

	// Initialize dual
	h.RunDual("init")

	// Create context without specifying port
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Auto-assigned base port: 4100")

	// Create another context without specifying port
	h.CreateGitBranch("develop")
	stdout, stderr, exitCode = h.RunDual("context", "create", "develop")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Auto-assigned base port: 4200")

	// Create a third context
	h.CreateGitBranch("feature")
	stdout, stderr, exitCode = h.RunDual("context", "create", "feature")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Auto-assigned base port: 4300")
}

// makeExecutable makes a file executable
func makeExecutable(path string) error {
	return os.Chmod(path, 0755)
}
