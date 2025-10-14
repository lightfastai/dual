package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPortConflictHandling tests that ports are assigned deterministically without conflicts
func TestPortConflictHandling(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.RunDual("init")

	// Add services
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.CreateDirectory("apps/worker")
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")
	h.RunDual("service", "add", "worker", "--path", "apps/worker")

	// Create multiple contexts with different base ports
	t.Log("Creating multiple contexts")
	h.CreateGitBranch("main")
	h.RunDual("context", "create", "main", "--base-port", "4100")

	h.CreateGitBranch("develop")
	h.RunDual("context", "create", "develop", "--base-port", "4200")

	h.CreateGitBranch("feature/a")
	h.RunDual("context", "create", "feature/a", "--base-port", "4300")

	// Verify each context has unique port ranges
	contexts := []struct {
		name     string
		basePort int
	}{
		{"main", 4100},
		{"develop", 4200},
		{"feature/a", 4300},
	}

	allPorts := make(map[int]string) // port -> context

	for _, ctx := range contexts {
		t.Logf("Checking context: %s", ctx.name)

		// Switch to the branch
		h.RunGitCommand("checkout", ctx.name)

		// Query all ports for this context
		stdout, stderr, exitCode := h.RunDual("ports")
		h.AssertExitCode(exitCode, 0, stdout+stderr)

		// Extract and verify ports
		expectedPorts := []int{
			ctx.basePort + 1, // api (alphabetically first)
			ctx.basePort + 2, // web (alphabetically second)
			ctx.basePort + 3, // worker (alphabetically third)
		}

		for _, port := range expectedPorts {
			portStr := fmt.Sprintf("%d", port)
			h.AssertOutputContains(stdout, portStr)

			// Check for conflicts
			if existingCtx, exists := allPorts[port]; exists {
				t.Errorf("Port conflict: port %d is used by both %s and %s", port, existingCtx, ctx.name)
			}
			allPorts[port] = ctx.name
		}
	}

	t.Logf("Total unique ports assigned: %d", len(allPorts))
	if len(allPorts) != 9 { // 3 contexts × 3 services
		t.Errorf("Expected 9 unique ports, got %d", len(allPorts))
	}
}

// TestPortCalculationDeterminism tests that port calculation is deterministic
func TestPortCalculationDeterminism(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Add services in a specific order
	h.CreateDirectory("services/zebra")
	h.CreateDirectory("services/alpha")
	h.CreateDirectory("services/beta")

	// Add services in non-alphabetical order
	h.RunDual("service", "add", "zebra", "--path", "services/zebra")
	h.RunDual("service", "add", "alpha", "--path", "services/alpha")
	h.RunDual("service", "add", "beta", "--path", "services/beta")

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Query ports multiple times - should be consistent
	for i := 0; i < 3; i++ {
		t.Logf("Iteration %d", i+1)

		stdout, stderr, exitCode := h.RunDual("ports")
		h.AssertExitCode(exitCode, 0, stdout+stderr)

		// Ports should be assigned alphabetically regardless of add order
		// alpha (0) -> 4101
		// beta (1) -> 4102
		// zebra (2) -> 4103
		h.AssertOutputContains(stdout, "alpha")
		h.AssertOutputContains(stdout, "4101")
		h.AssertOutputContains(stdout, "beta")
		h.AssertOutputContains(stdout, "4102")
		h.AssertOutputContains(stdout, "zebra")
		h.AssertOutputContains(stdout, "4103")
	}
}

// TestAutoPortAssignment tests the FindNextAvailablePort functionality
func TestAutoPortAssignment(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.RunDual("init")

	// Create contexts without specifying ports
	contexts := []struct {
		branch       string
		expectedPort int
	}{
		{"main", 4100},
		{"develop", 4200},
		{"feature/a", 4300},
		{"feature/b", 4400},
		{"feature/c", 4500},
	}

	for _, ctx := range contexts {
		h.CreateGitBranch(ctx.branch)

		stdout, stderr, exitCode := h.RunDual("context", "create", ctx.branch)
		h.AssertExitCode(exitCode, 0, stdout+stderr)

		// Should auto-assign port (output goes to stdout)
		output := stdout + stderr
		h.AssertOutputContains(output, "Auto-assigned base port:")
		h.AssertOutputContains(output, fmt.Sprintf("%d", ctx.expectedPort))
	}

	// Verify all contexts have unique ports
	registry := h.ReadRegistryJSON()
	for _, ctx := range contexts {
		portStr := fmt.Sprintf("%d", ctx.expectedPort)
		if !strings.Contains(registry, portStr) {
			t.Errorf("Registry does not contain expected port %s for context %s", portStr, ctx.branch)
		}
	}
}

// TestPortAssignmentWithGaps tests that auto-assignment fills gaps
func TestPortAssignmentWithGaps(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.RunDual("init")

	// Create contexts with specific ports, leaving a gap
	h.CreateGitBranch("main")
	h.RunDual("context", "create", "main", "--base-port", "4100")

	h.CreateGitBranch("develop")
	h.RunDual("context", "create", "develop", "--base-port", "4300") // Skip 4200

	h.CreateGitBranch("feature")
	h.RunDual("context", "create", "feature", "--base-port", "4400")

	// Now create a context without specifying port - should fill the gap at 4200
	h.CreateGitBranch("bugfix")
	stdout, stderr, exitCode := h.RunDual("context", "create", "bugfix")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Auto-assigned base port: 4200")
}

// TestPortBoundaryValidation tests port validation at boundaries
func TestPortBoundaryValidation(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Test 1: Port below 1024 (privileged ports)
	t.Log("Test 1: Port below 1024 should fail")
	stdout, stderr, exitCode := h.RunDual("context", "create", "test1", "--base-port", "80")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "base port must be between 1024 and 65535")

	// Test 2: Port at 1024 (minimum valid)
	t.Log("Test 2: Port at 1024 should succeed")
	stdout, stderr, exitCode = h.RunDual("context", "create", "test2", "--base-port", "1024")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test 3: Port at 65535 (maximum valid)
	t.Log("Test 3: Port at 65535 should succeed")
	stdout, stderr, exitCode = h.RunDual("context", "create", "test3", "--base-port", "65535")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test 4: Port above 65535
	t.Log("Test 4: Port above 65535 should fail")
	stdout, stderr, exitCode = h.RunDual("context", "create", "test4", "--base-port", "65536")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "base port must be between 1024 and 65535")
}

// TestContextDuplicatePrevention tests that duplicate contexts are prevented
func TestContextDuplicatePrevention(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create a context
	stdout, stderr, exitCode := h.RunDual("context", "create", "main", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Try to create the same context again
	stdout, stderr, exitCode = h.RunDual("context", "create", "main", "--base-port", "4200")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stdout+stderr, "context \"main\" already exists")
	h.AssertOutputContains(stdout+stderr, "4100") // Should mention existing base port
}

// TestPortCalculationWithManyServices tests port calculation with large number of services
func TestPortCalculationWithManyServices(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create 50 services
	numServices := 50
	for i := 1; i <= numServices; i++ {
		serviceName := fmt.Sprintf("service%02d", i)
		servicePath := fmt.Sprintf("services/%s", serviceName)
		h.CreateDirectory(servicePath)
		h.RunDual("service", "add", serviceName, "--path", servicePath)
	}

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Query all ports
	stdout, stderr, exitCode := h.RunDual("ports")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Verify all services are listed with correct ports
	for i := 1; i <= numServices; i++ {
		serviceName := fmt.Sprintf("service%02d", i)
		expectedPort := 4100 + i // basePort + (i-1) + 1
		h.AssertOutputContains(stdout, serviceName)
		h.AssertOutputContains(stdout, fmt.Sprintf("%d", expectedPort))
	}

	// Test port query for specific service
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "services/service25"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4125") // 4100 + 24 + 1
}

// TestPortStability tests that ports remain stable after config changes
func TestPortStability(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Add initial services
	h.CreateDirectory("apps/api")
	h.CreateDirectory("apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")
	h.RunDual("service", "add", "web", "--path", "apps/web")

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Record initial ports
	stdout1, stderr1, exitCode := h.RunDual("ports")
	h.AssertExitCode(exitCode, 0, stdout1+stderr1)
	t.Logf("Initial ports:\n%s", stdout1)

	// api: 4101, web: 4102

	// Add a new service that would come first alphabetically
	h.CreateDirectory("apps/admin")
	h.RunDual("service", "add", "admin", "--path", "apps/admin")

	// Query ports again
	stdout2, stderr2, exitCode := h.RunDual("ports")
	h.AssertExitCode(exitCode, 0, stdout2+stderr2)
	t.Logf("Ports after adding admin:\n%s", stdout2)

	// Now: admin: 4101, api: 4102, web: 4103
	// Note: This is expected behavior - adding services changes the alphabetical order
	// This test documents this behavior

	h.AssertOutputContains(stdout2, "admin")
	h.AssertOutputContains(stdout2, "4101")
	h.AssertOutputContains(stdout2, "api")
	h.AssertOutputContains(stdout2, "4102")
	h.AssertOutputContains(stdout2, "web")
	h.AssertOutputContains(stdout2, "4103")

	// This demonstrates that port assignments are deterministic but not stable
	// when services are added/removed. This is by design - service order is alphabetical.
}

// TestMultiProjectPortIsolation tests that different projects have isolated port assignments
func TestMultiProjectPortIsolation(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Create first project
	project1 := filepath.Join(h.TempDir, "project1")
	if err := os.MkdirAll(project1, 0o755); err != nil {
		t.Fatalf("failed to create project1: %v", err)
	}

	// Initialize first project
	initGitRepoInDir(t, project1)
	createGitBranchInDir(t, project1, "main")
	h.RunDualInDir(project1, "init")

	// Create service directory for project1
	project1WebDir := filepath.Join(project1, "apps/web")
	if err := os.MkdirAll(project1WebDir, 0o755); err != nil {
		t.Fatalf("failed to create project1 web directory: %v", err)
	}
	h.RunDualInDir(project1, "service", "add", "web", "--path", "apps/web")
	h.RunDualInDir(project1, "context", "create", "main", "--base-port", "4100")

	// Create second project
	project2 := filepath.Join(h.TempDir, "project2")
	if err := os.MkdirAll(project2, 0o755); err != nil {
		t.Fatalf("failed to create project2: %v", err)
	}

	// Initialize second project
	initGitRepoInDir(t, project2)
	createGitBranchInDir(t, project2, "main")
	h.RunDualInDir(project2, "init")

	// Create service directory for project2
	project2ApiDir := filepath.Join(project2, "apps/api")
	if err := os.MkdirAll(project2ApiDir, 0o755); err != nil {
		t.Fatalf("failed to create project2 api directory: %v", err)
	}
	h.RunDualInDir(project2, "service", "add", "api", "--path", "apps/api")
	h.RunDualInDir(project2, "context", "create", "main", "--base-port", "4100")

	// Both projects can use the same base port independently
	// Query ports from project1
	stdout1, stderr1, exitCode := h.RunDualInDir(project1, "ports")
	h.AssertExitCode(exitCode, 0, stdout1+stderr1)
	h.AssertOutputContains(stdout1, "web")
	h.AssertOutputContains(stdout1, "4101")

	// Query ports from project2
	stdout2, stderr2, exitCode := h.RunDualInDir(project2, "ports")
	h.AssertExitCode(exitCode, 0, stdout2+stderr2)
	h.AssertOutputContains(stdout2, "api")
	h.AssertOutputContains(stdout2, "4101")

	// Verify registry contains both projects
	registry := h.ReadRegistryJSON()
	h.AssertOutputContains(registry, "project1")
	h.AssertOutputContains(registry, "project2")
}

// Helper functions for multi-project test
func initGitRepoInDir(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init git repo in %s: %v\n%s", dir, err, output)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	cmd.Run()
}

func createGitBranchInDir(t *testing.T, dir, branch string) {
	t.Helper()

	// Create initial commit if needed
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	if _, err := cmd.Output(); err != nil {
		readmePath := filepath.Join(dir, "README.md")
		if err := os.WriteFile(readmePath, []byte("# Test"), 0o644); err != nil {
			t.Fatalf("failed to write README: %v", err)
		}

		cmd = exec.Command("git", "add", "README.md")
		cmd.Dir = dir
		cmd.Run()

		cmd = exec.Command("git", "commit", "-m", "Initial commit")
		cmd.Dir = dir
		cmd.Run()
	}

	// Check if branch already exists
	cmd = exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = dir
	if output, err := cmd.Output(); err == nil && strings.TrimSpace(string(output)) != "" {
		// Branch exists, just check it out
		cmd = exec.Command("git", "checkout", branch)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout existing branch %s: %v", branch, err)
		}
		return
	}

	// Branch doesn't exist, create it
	cmd = exec.Command("git", "checkout", "-b", branch)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create branch %s: %v\n%s", branch, err, output)
	}
}
