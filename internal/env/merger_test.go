package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lightfastai/dual/internal/config"
)

// TestLoadLayeredEnv_WorktreeInheritance tests that service .env files
// are properly loaded from both parent repo and worktree, with worktree overriding
func TestLoadLayeredEnv_WorktreeInheritance(t *testing.T) {
	// Create a temporary directory structure simulating a parent repo with a worktree
	tmpDir := t.TempDir()

	// Create parent repo structure
	parentRepo := filepath.Join(tmpDir, "parent")
	if err := os.MkdirAll(filepath.Join(parentRepo, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parentRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create worktree structure
	worktree := filepath.Join(tmpDir, "worktrees", "feature")
	if err := os.MkdirAll(filepath.Join(worktree, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .git file in worktree pointing to parent (simulates git worktree)
	gitFile := filepath.Join(worktree, ".git")
	gitContent := "gitdir: " + filepath.Join(parentRepo, ".git", "worktrees", "feature")
	if err := os.WriteFile(gitFile, []byte(gitContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create dual.config.yml in worktree
	configContent := `version: 1
services:
  web:
    path: apps/web
`
	configPath := filepath.Join(worktree, "dual.config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create parent repo's service .env file (23 variables as in the bug report)
	parentEnvContent := `PARENT_VAR1=parent1
PARENT_VAR2=parent2
SHARED_VAR=parent_value
DATABASE_URL=postgresql://localhost/parent_db
NEXT_PUBLIC_POSTHOG_KEY=parent_posthog_key
NEXT_PUBLIC_POSTHOG_HOST=parent_posthog_host
`
	parentEnvPath := filepath.Join(parentRepo, "apps", "web", ".env")
	if err := os.WriteFile(parentEnvPath, []byte(parentEnvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create worktree's service .env file (should override parent)
	worktreeEnvContent := `WORKTREE_VAR1=worktree1
SHARED_VAR=worktree_value
`
	worktreeEnvPath := filepath.Join(worktree, "apps", "web", ".env")
	if err := os.WriteFile(worktreeEnvPath, []byte(worktreeEnvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load config from worktree
	cfg, err := config.LoadConfigFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Load layered environment
	layeredEnv, err := LoadLayeredEnv(worktree, cfg, "web", "feature", nil)
	if err != nil {
		t.Fatalf("LoadLayeredEnv failed: %v", err)
	}

	// Verify that service layer contains variables from both parent and worktree
	expectedVars := map[string]string{
		// From parent repo
		"PARENT_VAR1":              "parent1",
		"PARENT_VAR2":              "parent2",
		"DATABASE_URL":             "postgresql://localhost/parent_db",
		"NEXT_PUBLIC_POSTHOG_KEY":  "parent_posthog_key",
		"NEXT_PUBLIC_POSTHOG_HOST": "parent_posthog_host",
		// From worktree (new var)
		"WORKTREE_VAR1": "worktree1",
		// From worktree (overriding parent)
		"SHARED_VAR": "worktree_value",
	}

	for key, expectedValue := range expectedVars {
		if actualValue, ok := layeredEnv.Service[key]; !ok {
			t.Errorf("missing expected key %q in service layer", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %q: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Verify the count - should have 7 variables total
	if len(layeredEnv.Service) != 7 {
		t.Errorf("expected 7 variables in service layer, got %d", len(layeredEnv.Service))
	}

	// Verify that SHARED_VAR is overridden by worktree value
	if layeredEnv.Service["SHARED_VAR"] != "worktree_value" {
		t.Errorf("SHARED_VAR should be overridden by worktree, got: %q", layeredEnv.Service["SHARED_VAR"])
	}
}

// TestLoadLayeredEnv_NonWorktree tests that in a non-worktree context,
// only the local service .env is loaded (backward compatible behavior)
func TestLoadLayeredEnv_NonWorktree(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a regular repo structure (not a worktree)
	repo := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create dual.config.yml
	configContent := `version: 1
services:
  web:
    path: apps/web
`
	configPath := filepath.Join(repo, "dual.config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create service .env file
	envContent := `LOCAL_VAR1=local1
LOCAL_VAR2=local2
DATABASE_URL=postgresql://localhost/local_db
`
	envPath := filepath.Join(repo, "apps", "web", ".env")
	if err := os.WriteFile(envPath, []byte(envContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.LoadConfigFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Load layered environment
	layeredEnv, err := LoadLayeredEnv(repo, cfg, "web", "", nil)
	if err != nil {
		t.Fatalf("LoadLayeredEnv failed: %v", err)
	}

	// Verify that service layer contains only local variables
	expectedVars := map[string]string{
		"LOCAL_VAR1":   "local1",
		"LOCAL_VAR2":   "local2",
		"DATABASE_URL": "postgresql://localhost/local_db",
	}

	for key, expectedValue := range expectedVars {
		if actualValue, ok := layeredEnv.Service[key]; !ok {
			t.Errorf("missing expected key %q in service layer", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %q: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Verify the count
	if len(layeredEnv.Service) != 3 {
		t.Errorf("expected 3 variables in service layer, got %d", len(layeredEnv.Service))
	}
}

// TestLoadLayeredEnv_WorktreeOnlyParent tests when worktree has no .env,
// parent repo's .env should still be loaded
func TestLoadLayeredEnv_WorktreeOnlyParent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create parent repo structure
	parentRepo := filepath.Join(tmpDir, "parent")
	if err := os.MkdirAll(filepath.Join(parentRepo, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(parentRepo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create worktree structure
	worktree := filepath.Join(tmpDir, "worktrees", "feature")
	if err := os.MkdirAll(filepath.Join(worktree, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .git file in worktree
	gitFile := filepath.Join(worktree, ".git")
	gitContent := "gitdir: " + filepath.Join(parentRepo, ".git", "worktrees", "feature")
	if err := os.WriteFile(gitFile, []byte(gitContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create dual.config.yml in worktree
	configContent := `version: 1
services:
  web:
    path: apps/web
`
	configPath := filepath.Join(worktree, "dual.config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create ONLY parent repo's service .env file
	parentEnvContent := `PARENT_VAR1=parent1
PARENT_VAR2=parent2
DATABASE_URL=postgresql://localhost/parent_db
`
	parentEnvPath := filepath.Join(parentRepo, "apps", "web", ".env")
	if err := os.WriteFile(parentEnvPath, []byte(parentEnvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// DO NOT create worktree's service .env file (simulates gitignored .env)

	// Load config from worktree
	cfg, err := config.LoadConfigFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Load layered environment
	layeredEnv, err := LoadLayeredEnv(worktree, cfg, "web", "feature", nil)
	if err != nil {
		t.Fatalf("LoadLayeredEnv failed: %v", err)
	}

	// Verify that service layer contains parent repo variables
	expectedVars := map[string]string{
		"PARENT_VAR1":  "parent1",
		"PARENT_VAR2":  "parent2",
		"DATABASE_URL": "postgresql://localhost/parent_db",
	}

	for key, expectedValue := range expectedVars {
		if actualValue, ok := layeredEnv.Service[key]; !ok {
			t.Errorf("missing expected key %q in service layer (should inherit from parent)", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %q: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	// Verify the count - should have 3 variables from parent
	if len(layeredEnv.Service) != 3 {
		t.Errorf("expected 3 variables in service layer (from parent), got %d", len(layeredEnv.Service))
	}
}

// TestLoadLayeredEnv_BaseFile tests that base file loading works correctly
func TestLoadLayeredEnv_BaseFile(t *testing.T) {
	tmpDir := t.TempDir()

	repo := filepath.Join(tmpDir, "repo")
	if err := os.MkdirAll(filepath.Join(repo, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create dual.config.yml with baseFile
	configContent := `version: 1
services:
  web:
    path: apps/web
env:
  baseFile: .env.base
`
	configPath := filepath.Join(repo, "dual.config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create base environment file
	baseEnvContent := `BASE_VAR1=base1
BASE_VAR2=base2
SHARED_VAR=base_value
`
	baseEnvPath := filepath.Join(repo, ".env.base")
	if err := os.WriteFile(baseEnvPath, []byte(baseEnvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create service .env file
	serviceEnvContent := `SERVICE_VAR1=service1
SHARED_VAR=service_value
`
	serviceEnvPath := filepath.Join(repo, "apps", "web", ".env")
	if err := os.WriteFile(serviceEnvPath, []byte(serviceEnvContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Load config
	cfg, err := config.LoadConfigFrom(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Load layered environment
	layeredEnv, err := LoadLayeredEnv(repo, cfg, "web", "", nil)
	if err != nil {
		t.Fatalf("LoadLayeredEnv failed: %v", err)
	}

	// Verify base layer
	if len(layeredEnv.Base) != 3 {
		t.Errorf("expected 3 variables in base layer, got %d", len(layeredEnv.Base))
	}
	if layeredEnv.Base["BASE_VAR1"] != "base1" {
		t.Errorf("BASE_VAR1 in base layer: expected 'base1', got %q", layeredEnv.Base["BASE_VAR1"])
	}

	// Verify service layer
	if len(layeredEnv.Service) != 2 {
		t.Errorf("expected 2 variables in service layer, got %d", len(layeredEnv.Service))
	}
	if layeredEnv.Service["SERVICE_VAR1"] != "service1" {
		t.Errorf("SERVICE_VAR1 in service layer: expected 'service1', got %q", layeredEnv.Service["SERVICE_VAR1"])
	}

	// Verify merged environment has service value overriding base
	merged := layeredEnv.Merge()
	if merged["SHARED_VAR"] != "service_value" {
		t.Errorf("SHARED_VAR in merged: expected 'service_value', got %q", merged["SHARED_VAR"])
	}
}

// TestLayeredEnv_Merge tests the merge priority
func TestLayeredEnv_Merge(t *testing.T) {
	env := &LayeredEnv{
		Base: map[string]string{
			"VAR1": "base",
			"VAR2": "base",
			"VAR3": "base",
		},
		Service: map[string]string{
			"VAR2": "service",
			"VAR3": "service",
			"VAR4": "service",
		},
		Overrides: map[string]string{
			"VAR3": "override",
			"VAR5": "override",
		},
	}

	merged := env.Merge()

	// Verify merge priority: Base < Service < Overrides
	expected := map[string]string{
		"VAR1": "base",     // Only in base
		"VAR2": "service",  // Base overridden by service
		"VAR3": "override", // Base and service overridden by override
		"VAR4": "service",  // Only in service
		"VAR5": "override", // Only in override
	}

	for key, expectedValue := range expected {
		if actualValue, ok := merged[key]; !ok {
			t.Errorf("missing key %q in merged environment", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %q: expected %q, got %q", key, expectedValue, actualValue)
		}
	}

	if len(merged) != 5 {
		t.Errorf("expected 5 variables in merged environment, got %d", len(merged))
	}
}

// TestLayeredEnv_Stats tests the stats calculation
func TestLayeredEnv_Stats(t *testing.T) {
	env := &LayeredEnv{
		Base:      map[string]string{"BASE1": "v1", "BASE2": "v2"},
		Service:   map[string]string{"SVC1": "v1", "SVC2": "v2", "SVC3": "v3"},
		Overrides: map[string]string{"OVER1": "v1"},
	}

	stats := env.Stats()

	if stats.BaseVars != 2 {
		t.Errorf("expected 2 base vars, got %d", stats.BaseVars)
	}
	if stats.ServiceVars != 3 {
		t.Errorf("expected 3 service vars, got %d", stats.ServiceVars)
	}
	if stats.OverrideVars != 1 {
		t.Errorf("expected 1 override var, got %d", stats.OverrideVars)
	}
	if stats.TotalVars != 6 {
		t.Errorf("expected 6 total vars, got %d", stats.TotalVars)
	}
}
