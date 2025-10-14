package health

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lightfastai/dual/internal/config"
	"github.com/lightfastai/dual/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckGitRepository(t *testing.T) {
	// This test assumes we're running in a git repository
	// which should be true for the dual project itself
	check := CheckGitRepository()

	assert.Equal(t, "Git Repository", check.Name)
	// The status depends on whether we're in a git repo
	// In CI and development, we should be in a git repo
	if check.Status == StatusError {
		assert.Contains(t, check.Message, "Not in a git repository")
	} else {
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "Valid git repository")
	}
}

func TestCheckConfigFile(t *testing.T) {
	t.Run("No config", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: nil,
		}

		check := CheckConfigFile(ctx)
		assert.Equal(t, StatusError, check.Status)
		assert.Contains(t, check.Message, "No dual.config.yml found")
		assert.Contains(t, check.FixAction, "dual init")
	})

	t.Run("Valid config with services", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: &config.Config{
				Version: config.SupportedVersion,
				Services: map[string]config.Service{
					"api": {Path: "apps/api"},
					"web": {Path: "apps/web"},
				},
			},
			ProjectRoot: "/tmp/project",
		}

		check := CheckConfigFile(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "2 service(s)")
	})

	t.Run("Valid config with no services", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: &config.Config{
				Version:  config.SupportedVersion,
				Services: map[string]config.Service{},
			},
			ProjectRoot: "/tmp/project",
		}

		check := CheckConfigFile(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "no services defined")
		assert.Contains(t, check.FixAction, "dual service add")
	})

	t.Run("Unsupported version", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: &config.Config{
				Version: 999,
				Services: map[string]config.Service{
					"api": {Path: "apps/api"},
				},
			},
			ProjectRoot: "/tmp/project",
		}

		check := CheckConfigFile(ctx)
		assert.Equal(t, StatusError, check.Status)
		assert.Contains(t, check.Message, "Unsupported config version")
	})
}

func TestCheckRegistry(t *testing.T) {
	t.Run("No registry", func(t *testing.T) {
		ctx := &CheckerContext{
			Registry: nil,
		}

		check := CheckRegistry(ctx)
		assert.Equal(t, StatusError, check.Status)
		assert.Contains(t, check.Message, "Registry could not be loaded")
	})

	t.Run("Valid registry", func(t *testing.T) {
		reg := &registry.Registry{
			Projects: map[string]registry.Project{
				"/project1": {
					Contexts: map[string]registry.Context{
						"main": {BasePort: 4100, Created: time.Now()},
					},
				},
			},
		}

		ctx := &CheckerContext{
			Registry: reg,
		}

		check := CheckRegistry(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "1 project(s)")
		assert.Contains(t, check.Message, "1 context(s)")
	})
}

func TestCheckCurrentContext(t *testing.T) {
	t.Run("Context in registry", func(t *testing.T) {
		projectID := "/test/project"
		contextName := "test-branch"

		reg := &registry.Registry{
			Projects: map[string]registry.Project{
				projectID: {
					Contexts: map[string]registry.Context{
						contextName: {
							BasePort: 4100,
							Created:  time.Now(),
							Path:     "/test/path",
						},
					},
				},
			},
		}

		ctx := &CheckerContext{
			Registry:       reg,
			ProjectID:      projectID,
			CurrentContext: contextName,
		}

		check := CheckCurrentContext(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "is valid and registered")
	})

	t.Run("Context not in registry", func(t *testing.T) {
		reg := &registry.Registry{
			Projects: map[string]registry.Project{},
		}

		ctx := &CheckerContext{
			Registry:       reg,
			ProjectID:      "/test/project",
			CurrentContext: "missing-context",
		}

		check := CheckCurrentContext(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "not in registry")
		assert.Contains(t, check.FixAction, "dual context create")
	})
}

func TestCheckServicePaths(t *testing.T) {
	t.Run("No services", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: &config.Config{
				Services: map[string]config.Service{},
			},
		}

		check := CheckServicePaths(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "No services configured")
	})

	t.Run("Valid paths", func(t *testing.T) {
		// Create temp directories
		tmpDir := t.TempDir()
		service1Path := filepath.Join(tmpDir, "service1")
		service2Path := filepath.Join(tmpDir, "service2")

		require.NoError(t, os.MkdirAll(service1Path, 0o755))
		require.NoError(t, os.MkdirAll(service2Path, 0o755))

		ctx := &CheckerContext{
			Config: &config.Config{
				Services: map[string]config.Service{
					"service1": {Path: "service1"},
					"service2": {Path: "service2"},
				},
			},
			ProjectRoot: tmpDir,
		}

		check := CheckServicePaths(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "2 service path(s) are valid")
	})

	t.Run("Invalid paths", func(t *testing.T) {
		tmpDir := t.TempDir()

		ctx := &CheckerContext{
			Config: &config.Config{
				Services: map[string]config.Service{
					"missing": {Path: "does-not-exist"},
				},
			},
			ProjectRoot: tmpDir,
		}

		check := CheckServicePaths(ctx)
		assert.Equal(t, StatusError, check.Status)
		assert.Contains(t, check.Message, "invalid")
	})
}

func TestCheckEnvironmentFiles(t *testing.T) {
	t.Run("No env files configured", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: &config.Config{
				Services: map[string]config.Service{
					"api": {Path: "apps/api"},
				},
			},
		}

		check := CheckEnvironmentFiles(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "No environment files configured")
	})

	t.Run("Base env file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		envFile := filepath.Join(tmpDir, ".env")
		require.NoError(t, os.WriteFile(envFile, []byte("FOO=bar"), 0o644))

		ctx := &CheckerContext{
			Config: &config.Config{
				Env: config.EnvConfig{
					BaseFile: ".env",
				},
				Services: map[string]config.Service{},
			},
			ProjectRoot: tmpDir,
		}

		check := CheckEnvironmentFiles(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "1 environment file(s)")
	})

	t.Run("Env file missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		ctx := &CheckerContext{
			Config: &config.Config{
				Env: config.EnvConfig{
					BaseFile: ".env",
				},
			},
			ProjectRoot: tmpDir,
		}

		check := CheckEnvironmentFiles(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "not found")
	})
}

func TestCheckPortConflicts(t *testing.T) {
	t.Run("No conflicts", func(t *testing.T) {
		reg := &registry.Registry{
			Projects: map[string]registry.Project{
				"/project1": {
					Contexts: map[string]registry.Context{
						"main": {BasePort: 4100},
						"dev":  {BasePort: 4200},
					},
				},
			},
		}

		cfg := &config.Config{
			Services: map[string]config.Service{
				"api": {Path: "api"},
				"web": {Path: "web"},
			},
		}

		ctx := &CheckerContext{
			Registry:  reg,
			Config:    cfg,
			ProjectID: "/project1",
		}

		check := CheckPortConflicts(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "No port conflicts")
	})

	t.Run("Duplicate base ports", func(t *testing.T) {
		reg := &registry.Registry{
			Projects: map[string]registry.Project{
				"/project1": {
					Contexts: map[string]registry.Context{
						"main": {BasePort: 4100},
					},
				},
				"/project2": {
					Contexts: map[string]registry.Context{
						"main": {BasePort: 4100},
					},
				},
			},
		}

		cfg := &config.Config{
			Services: map[string]config.Service{
				"api": {Path: "api"},
			},
		}

		ctx := &CheckerContext{
			Registry:  reg,
			Config:    cfg,
			ProjectID: "/project1",
		}

		check := CheckPortConflicts(ctx)
		assert.Equal(t, StatusError, check.Status)
		assert.Contains(t, check.Message, "conflict")
	})
}

func TestCheckOrphanedContexts(t *testing.T) {
	t.Run("No orphaned contexts", func(t *testing.T) {
		tmpDir := t.TempDir()
		contextPath := filepath.Join(tmpDir, "worktree")
		require.NoError(t, os.MkdirAll(contextPath, 0o755))

		reg := &registry.Registry{
			Projects: map[string]registry.Project{
				"/project": {
					Contexts: map[string]registry.Context{
						"main": {
							BasePort: 4100,
							Path:     contextPath,
						},
					},
				},
			},
		}

		ctx := &CheckerContext{
			Registry: reg,
		}

		check := CheckOrphanedContexts(ctx)
		assert.Equal(t, StatusPass, check.Status)
		assert.Contains(t, check.Message, "No orphaned contexts")
	})

	t.Run("Orphaned context detected", func(t *testing.T) {
		reg := &registry.Registry{
			Projects: map[string]registry.Project{
				"/project": {
					Contexts: map[string]registry.Context{
						"main": {
							BasePort: 4100,
							Path:     "/non/existent/path",
						},
					},
				},
			},
		}

		ctx := &CheckerContext{
			Registry: reg,
			AutoFix:  false,
		}

		check := CheckOrphanedContexts(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "orphaned")
		assert.Contains(t, check.FixAction, "--fix")
	})
}

func TestCheckPermissions(t *testing.T) {
	ctx := &CheckerContext{
		ProjectRoot: t.TempDir(),
	}

	check := CheckPermissions(ctx)
	assert.Equal(t, "Permissions", check.Name)
	// Status can vary depending on system state
	// Just ensure it runs without panic
}

func TestCheckWorktrees(t *testing.T) {
	ctx := &CheckerContext{
		ProjectRoot: t.TempDir(),
	}

	check := CheckWorktrees(ctx)
	assert.Equal(t, "Worktrees", check.Name)
	// This will vary based on whether we're in a worktree
	// Just ensure it runs without panic
}

func TestCheckServiceDetection(t *testing.T) {
	t.Run("No services configured", func(t *testing.T) {
		ctx := &CheckerContext{
			Config: &config.Config{
				Services: map[string]config.Service{},
			},
		}

		check := CheckServiceDetection(ctx)
		assert.Equal(t, StatusWarn, check.Status)
		assert.Contains(t, check.Message, "No services configured")
	})
}
