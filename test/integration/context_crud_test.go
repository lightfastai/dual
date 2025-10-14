package integration

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestContextList tests the context list command
func TestContextList(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
  web:
    path: services/web
    envFile: services/web/.env
`)
	h.CreateDirectory("services/api")
	h.CreateDirectory("services/web")

	// Create a few contexts
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	h.CreateGitBranch("feature-a")
	stdout, stderr, exitCode = h.RunDual("context", "create", "feature-a")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	h.CreateGitBranch("feature-b")
	stdout, stderr, exitCode = h.RunDual("context", "create", "feature-b")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test basic list (should be on feature-b branch)
	stdout, stderr, exitCode = h.RunDual("context", "list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "main")
	h.AssertOutputContains(stdout, "feature-a")
	h.AssertOutputContains(stdout, "feature-b")
	h.AssertOutputContains(stdout, "(current)") // feature-b is current
	h.AssertOutputContains(stdout, "Total: 3 contexts")
}

// TestContextListJSON tests the context list command with JSON output
func TestContextListJSON(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")

	// Create a context
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test JSON output
	stdout, stderr, exitCode = h.RunDual("context", "list", "--json")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	// Verify structure
	if result["projectRoot"] == nil {
		t.Error("JSON output missing projectRoot")
	}
	if result["currentContext"] == nil {
		t.Error("JSON output missing currentContext")
	}
	if result["contexts"] == nil {
		t.Error("JSON output missing contexts")
	}

	contexts := result["contexts"].([]interface{})
	if len(contexts) == 0 {
		t.Error("contexts array is empty")
	}

	// Check first context has expected fields
	ctx := contexts[0].(map[string]interface{})
	if ctx["name"] == nil {
		t.Error("context missing name field")
	}
	if ctx["basePort"] == nil {
		t.Error("context missing basePort field")
	}
	if ctx["created"] == nil {
		t.Error("context missing created field")
	}
}

// TestContextListWithPorts tests the context list command with --ports flag
func TestContextListWithPorts(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
  web:
    path: services/web
    envFile: services/web/.env
`)
	h.CreateDirectory("services/api")
	h.CreateDirectory("services/web")

	// Create a context
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test list with ports
	stdout, stderr, exitCode = h.RunDual("context", "list", "--ports")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "api:4101")
	h.AssertOutputContains(stdout, "web:4102")
}

// TestContextListNoContexts tests listing when no contexts exist
func TestContextListNoContexts(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")
	h.CreateGitBranch("main")

	// List contexts (none created yet)
	stdout, stderr, exitCode := h.RunDual("context", "list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "No contexts found")
	h.AssertOutputContains(stdout, "dual context create")
}

// TestContextDelete tests the context delete command
func TestContextDelete(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")

	// Create two contexts
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	h.CreateGitBranch("feature-a")
	stdout, stderr, exitCode = h.RunDual("context", "create", "feature-a")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Switch back to main
	h.CreateGitBranch("main")

	// Delete feature-a context with --force
	stdout, stderr, exitCode = h.RunDual("context", "delete", "feature-a", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Deleted context")
	h.AssertOutputContains(stdout, "feature-a")

	// Verify it's deleted by listing
	stdout, stderr, exitCode = h.RunDual("context", "list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "main")
	h.AssertOutputNotContains(stdout, "feature-a")
	h.AssertOutputContains(stdout, "Total: 1 context")
}

// TestContextDeleteCurrent tests that deleting the current context fails
func TestContextDeleteCurrent(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")

	// Create a context
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Try to delete current context (should fail)
	stdout, stderr, exitCode = h.RunDual("context", "delete", "main", "--force")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "cannot delete current context")
}

// TestContextDeleteNonExistent tests deleting a non-existent context
func TestContextDeleteNonExistent(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")
	h.CreateGitBranch("main")

	// Try to delete non-existent context
	stdout, stderr, exitCode := h.RunDual("context", "delete", "nonexistent", "--force")
	h.AssertExitCode(exitCode, 1, stdout+stderr)
	h.AssertOutputContains(stderr, "not found")
}

// TestContextListAll tests listing contexts from all projects
func TestContextListAll(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Create first project
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// List all projects (should show current project)
	stdout, stderr, exitCode = h.RunDual("context", "list", "--all")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "Project:")
	h.AssertOutputContains(stdout, "main")
	h.AssertOutputContains(stdout, "Total: 1 contexts across 1 projects")
}

// TestContextListWithJSONAndPorts tests combining --json and --ports flags
func TestContextListWithJSONAndPorts(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
  web:
    path: services/web
    envFile: services/web/.env
`)
	h.CreateDirectory("services/api")
	h.CreateDirectory("services/web")

	// Create a context
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Test JSON output with ports
	stdout, stderr, exitCode = h.RunDual("context", "list", "--json", "--ports")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("failed to parse JSON output: %v\nOutput: %s", err, stdout)
	}

	contexts := result["contexts"].([]interface{})
	ctx := contexts[0].(map[string]interface{})

	// Verify ports field exists
	if ctx["ports"] == nil {
		t.Error("context missing ports field with --ports flag")
	}

	ports := ctx["ports"].(map[string]interface{})
	if ports["api"] == nil || ports["web"] == nil {
		t.Error("ports field missing service entries")
	}
}

// TestContextDeleteShowsInfo tests that delete shows context info before deletion
func TestContextDeleteShowsInfo(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")

	// Create two contexts
	h.CreateGitBranch("main")
	stdout, stderr, exitCode := h.RunDual("context", "create", "main", "--base-port", "4100")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	h.CreateGitBranch("feature-a")
	stdout, stderr, exitCode = h.RunDual("context", "create", "feature-a", "--base-port", "4200")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Switch back to main
	h.CreateGitBranch("main")

	// Delete feature-a with --force (should show info)
	stdout, stderr, exitCode = h.RunDual("context", "delete", "feature-a", "--force")
	h.AssertExitCode(exitCode, 0, stdout+stderr)
	h.AssertOutputContains(stdout, "About to delete context: feature-a")
	h.AssertOutputContains(stdout, "Base Port: 4200")
}

// TestContextListSorting tests that contexts are listed in alphabetical order
func TestContextListSorting(t *testing.T) {
	h := NewTestHelper(t)
	defer h.RestoreHome()

	// Initialize git repo and config
	h.InitGitRepo()
	h.WriteFile("dual.config.yml", `version: 1
services:
  api:
    path: services/api
    envFile: services/api/.env
`)
	h.CreateDirectory("services/api")

	// Create contexts in non-alphabetical order
	h.CreateGitBranch("zebra")
	stdout, stderr, exitCode := h.RunDual("context", "create", "zebra")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	h.CreateGitBranch("alpha")
	stdout, stderr, exitCode = h.RunDual("context", "create", "alpha")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	h.CreateGitBranch("beta")
	stdout, stderr, exitCode = h.RunDual("context", "create", "beta")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// List contexts
	stdout, stderr, exitCode = h.RunDual("context", "list")
	h.AssertExitCode(exitCode, 0, stdout+stderr)

	// Verify alphabetical order (alpha should come before beta, and beta before zebra)
	alphaIdx := strings.Index(stdout, "alpha")
	betaIdx := strings.Index(stdout, "beta")
	zebraIdx := strings.Index(stdout, "zebra")

	if alphaIdx == -1 || betaIdx == -1 || zebraIdx == -1 {
		t.Fatal("not all contexts found in output")
	}

	if !(alphaIdx < betaIdx && betaIdx < zebraIdx) {
		t.Errorf("contexts not in alphabetical order\nOutput: %s", stdout)
	}
}
