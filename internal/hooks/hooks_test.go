package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lightfastai/dual/internal/config"
)

func TestHookEvent_String(t *testing.T) {
	tests := []struct {
		name  string
		event HookEvent
		want  string
	}{
		{"PostWorktreeCreate", PostWorktreeCreate, "postWorktreeCreate"},
		{"PreWorktreeDelete", PreWorktreeDelete, "preWorktreeDelete"},
		{"PostWorktreeDelete", PostWorktreeDelete, "postWorktreeDelete"},
		{"PostEnvChange", PostEnvChange, "postEnvChange"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.String(); got != tt.want {
				t.Errorf("HookEvent.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHookEvent_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		event HookEvent
		want  bool
	}{
		{"Valid: PostWorktreeCreate", PostWorktreeCreate, true},
		{"Valid: PreWorktreeDelete", PreWorktreeDelete, true},
		{"Valid: PostWorktreeDelete", PostWorktreeDelete, true},
		{"Valid: PostEnvChange", PostEnvChange, true},
		{"Invalid: empty", HookEvent(""), false},
		{"Invalid: unknown", HookEvent("unknownEvent"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.IsValid(); got != tt.want {
				t.Errorf("HookEvent.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
	}
	projectRoot := "/test/project"

	manager := NewManager(cfg, projectRoot)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}
	if manager.config != cfg {
		t.Error("Manager config not set correctly")
	}
	if manager.projectRoot != projectRoot {
		t.Error("Manager projectRoot not set correctly")
	}
}

func TestManager_Execute_NoHooks(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		Hooks:   map[string][]string{},
	}

	manager := NewManager(cfg, "/test/project")
	ctx := HookContext{
		Event:       PostWorktreeCreate,
		ContextName: "test-context",
	}

	// Should not error when no hooks are defined
	err := manager.Execute(PostWorktreeCreate, ctx)
	if err != nil {
		t.Errorf("Execute() with no hooks should not error, got: %v", err)
	}
}

func TestManager_Execute_InvalidEvent(t *testing.T) {
	cfg := &config.Config{Version: 1}
	manager := NewManager(cfg, "/test/project")
	ctx := HookContext{Event: HookEvent("invalid")}

	err := manager.Execute(HookEvent("invalid"), ctx)
	if err == nil {
		t.Error("Execute() with invalid event should error")
	}
}

func TestManager_buildEnv(t *testing.T) {
	cfg := &config.Config{Version: 1}
	manager := NewManager(cfg, "/test/project")

	ctx := HookContext{
		Event:       PostWorktreeCreate,
		ContextName: "feature-branch",
		ContextPath: "/test/worktree",
		ProjectRoot: "/test/project",
	}

	env := manager.buildEnv(ctx)

	expectedVars := map[string]bool{
		"DUAL_EVENT=postWorktreeCreate":    false,
		"DUAL_CONTEXT_NAME=feature-branch": false,
		"DUAL_CONTEXT_PATH=/test/worktree": false,
		"DUAL_PROJECT_ROOT=/test/project":  false,
	}

	for _, envVar := range env {
		if _, exists := expectedVars[envVar]; exists {
			expectedVars[envVar] = true
		}
	}

	for expectedVar, found := range expectedVars {
		if !found {
			t.Errorf("Expected environment variable not found: %s", expectedVar)
		}
	}
}

func TestManager_Execute_WithHookScript(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create .dual/hooks directory
	hooksDir := filepath.Join(tempDir, ".dual", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatalf("Failed to create hooks directory: %v", err)
	}

	// Create a simple test hook script
	hookScript := filepath.Join(hooksDir, "test-hook.sh")
	scriptContent := `#!/bin/bash
echo "Hook executed successfully"
exit 0
`
	if err := os.WriteFile(hookScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	// Create config with hook
	cfg := &config.Config{
		Version: 1,
		Hooks: map[string][]string{
			"postWorktreeCreate": {"test-hook.sh"},
		},
	}

	manager := NewManager(cfg, tempDir)
	ctx := HookContext{
		Event:       PostWorktreeCreate,
		ContextName: "test",
		ContextPath: tempDir,
		ProjectRoot: tempDir,
	}

	err := manager.Execute(PostWorktreeCreate, ctx)
	if err != nil {
		t.Errorf("Execute() with valid hook script failed: %v", err)
	}
}
