package integration

import (
	"os"
	"path/filepath"
	"testing"
)

// TestServiceAutoDetection tests automatic service detection from current directory
func TestServiceAutoDetection(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create service directories
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.CreateDirectory("apps/worker")

	// Add services
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")
	h.RunDual("service", "add", "worker", "--path", "apps/worker")

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Test 1: Detection from service root directory
	t.Log("Test 1: Detection from service root directory")
	stdout, stderr, exitCode := h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4102") // web

	// Test 2: Detection from nested subdirectory
	t.Log("Test 2: Detection from nested subdirectory")
	h.CreateDirectory("apps/api/src/controllers")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/api/src/controllers"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101") // api

	// Test 3: Detection failure from outside service directories
	t.Log("Test 3: Detection failure from outside service directories")
	h.CreateDirectory("docs")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "docs"),
		"port",
	)
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "could not auto-detect service")
	h.AssertOutputContains(stderr, "Available services:")

	// Test 4: Explicit service argument to override auto-detection
	t.Log("Test 4: Explicit service argument override")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"port", "worker",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4103") // worker
}

// TestServiceDetectionLongestMatch tests the longest-match algorithm for nested services
func TestServiceDetectionLongestMatch(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create nested service structure
	// apps -> contains general app code
	// apps/web -> web-specific service
	h.CreateDirectory("apps")
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/web/client")

	// Add services (note: both start with "apps")
	h.RunDual("service", "add", "general", "--path", "apps")
	h.RunDual("service", "add", "web", "--path", "apps/web")

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Test 1: From apps root - should match "general"
	t.Log("Test 1: From apps root - should match 'general'")
	stdout, stderr, exitCode := h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101") // general (alphabetically first)

	// Test 2: From apps/web - should match "web" (longest match)
	t.Log("Test 2: From apps/web - should match 'web' (longest match)")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4102") // web (alphabetically second)

	// Test 3: From apps/web/client - should still match "web"
	t.Log("Test 3: From apps/web/client - should match 'web'")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web/client"),
		"port",
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4102") // web
}

// TestServiceDetectionWithSymlinks tests service detection with symbolic links
func TestServiceDetectionWithSymlinks(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create real service directory
	h.CreateDirectory("real/service/web")

	// Create symlink to the service
	symlinkPath := filepath.Join(h.ProjectDir, "apps/web")
	realPath := filepath.Join(h.ProjectDir, "real/service/web")

	// Create parent directory for symlink
	h.CreateDirectory("apps")

	// Create symlink
	if err := os.Symlink(realPath, symlinkPath); err != nil {
		t.Skipf("Skipping symlink test: %v", err)
	}

	// Add service using symlink path
	stdout, stderr, exitCode := h.RunDual("service", "add", "web", "--path", "apps/web")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Test 1: Detection from symlink path
	t.Log("Test 1: Detection from symlink path")
	stdout, stderr, exitCode = h.RunDualInDir(symlinkPath, "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101")

	// Test 2: Detection from real path
	t.Log("Test 2: Detection from real path")
	stdout, stderr, exitCode = h.RunDualInDir(realPath, "port")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "4101")
}

// TestServiceDetectionMultipleServices tests detection with many services
func TestServiceDetectionMultipleServices(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Create many services
	services := []string{
		"frontend/web",
		"frontend/admin",
		"frontend/mobile",
		"backend/api",
		"backend/auth",
		"backend/worker",
		"backend/scheduler",
	}

	for _, service := range services {
		h.CreateDirectory(service)
		// Extract service name from path (last component)
		parts := filepath.SplitList(service)
		if len(parts) == 0 {
			parts = []string{service}
		}
		serviceName := filepath.Base(service)
		h.RunDual("service", "add", serviceName, "--path", service)
	}

	// Create context
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Test detection from each service directory
	expectedPorts := map[string]string{
		"frontend/admin":       "4101", // admin (alphabetically 0)
		"frontend/mobile":      "4102", // mobile (alphabetically 1)
		"frontend/web":         "4103", // web (alphabetically 2)
		"backend/api":          "4104", // api (alphabetically 3)
		"backend/auth":         "4105", // auth (alphabetically 4)
		"backend/scheduler":    "4106", // scheduler (alphabetically 5)
		"backend/worker":       "4107", // worker (alphabetically 6)
	}

	// Wait, services are sorted alphabetically by NAME not path
	// So: admin, api, auth, mobile, scheduler, web, worker
	expectedPorts = map[string]string{
		"frontend/admin":    "4101", // admin (alphabetically 0)
		"backend/api":       "4102", // api (alphabetically 1)
		"backend/auth":      "4103", // auth (alphabetically 2)
		"frontend/mobile":   "4104", // mobile (alphabetically 3)
		"backend/scheduler": "4105", // scheduler (alphabetically 4)
		"frontend/web":      "4106", // web (alphabetically 5)
		"backend/worker":    "4107", // worker (alphabetically 6)
	}

	for servicePath, expectedPort := range expectedPorts {
		t.Logf("Testing service: %s (expected port: %s)", servicePath, expectedPort)
		stdout, stderr, exitCode := h.RunDualInDir(
			filepath.Join(h.ProjectDir, servicePath),
			"port",
		)
		h.AssertExitCode(exitCode, 0, stdout+stderr)
		h.AssertOutputContains(stdout, expectedPort)
	}

	// Test 'dual ports' command shows all services
	t.Log("Testing 'dual ports' command")
	stdout, stderr, exitCode := h.RunDual("ports")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Verify all services are listed
	for _, port := range expectedPorts {
		h.AssertOutputContains(stdout, port)
	}
}

// TestServiceDetectionErrorMessages tests error messages for service detection failures
func TestServiceDetectionErrorMessages(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	// Add some services
	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Test 1: No service detected from project root
	t.Log("Test 1: No service detected from project root")
	stdout, stderr, exitCode := h.RunDual("port")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "could not auto-detect service")
	h.AssertOutputContains(stderr, "Available services:")
	h.AssertOutputContains(stderr, "web")
	h.AssertOutputContains(stderr, "api")

	// Test 2: Invalid service with --service flag
	t.Log("Test 2: Invalid service with --service flag")
	stdout, stderr, exitCode = h.RunDual("--service", "nonexistent", "port")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "service \"nonexistent\" not found")
	h.AssertOutputContains(stderr, "Available services:")
}

// TestServiceDetectionWithCommandWrapper tests service detection in command wrapper
func TestServiceDetectionWithCommandWrapper(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Setup
	h.InitGitRepo()
	h.CreateGitBranch("main")
	h.RunDual("init")

	h.CreateDirectory("apps/web")
	h.CreateDirectory("apps/api")
	h.RunDual("service", "add", "web", "--path", "apps/web")
	h.RunDual("service", "add", "api", "--path", "apps/api")
	h.RunDual("context", "create", "main", "--base-port", "4100")

	// Create test script
	scriptContent := `#!/bin/sh
echo "Service detected: PORT=$PORT"
`
	h.WriteFile("test.sh", scriptContent)
	scriptPath := filepath.Join(h.ProjectDir, "test.sh")
	if err := makeExecutable(scriptPath); err != nil {
		t.Fatalf("failed to make script executable: %v", err)
	}

	// Test 1: Auto-detection from web directory
	t.Log("Test 1: Auto-detection from web directory")
	stdout, stderr, exitCode := h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"sh", scriptPath,
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Service: web")
	h.AssertOutputContains(stderr, "Port: 4102")
	h.AssertOutputContains(stdout, "PORT=4102")

	// Test 2: Auto-detection from api directory
	t.Log("Test 2: Auto-detection from api directory")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/api"),
		"sh", scriptPath,
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Service: api")
	h.AssertOutputContains(stderr, "Port: 4101")
	h.AssertOutputContains(stdout, "PORT=4101")

	// Test 3: Override with --service flag
	t.Log("Test 3: Override with --service flag")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "apps/web"),
		"--service", "api", "sh", scriptPath,
	)
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stderr, "Service: api")
	h.AssertOutputContains(stderr, "Port: 4101")
	h.AssertOutputContains(stdout, "PORT=4101")

	// Test 4: Failure from non-service directory without --service
	t.Log("Test 4: Failure from non-service directory without --service")
	h.CreateDirectory("docs")
	stdout, stderr, exitCode = h.RunDualInDir(
		filepath.Join(h.ProjectDir, "docs"),
		"sh", scriptPath,
	)
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "could not auto-detect service")
}
