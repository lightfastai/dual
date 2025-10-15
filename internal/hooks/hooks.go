package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lightfastai/dual/internal/config"
)

// Manager handles the execution of lifecycle hooks
type Manager struct {
	config      *config.Config
	projectRoot string
}

// NewManager creates a new hook manager
func NewManager(cfg *config.Config, projectRoot string) *Manager {
	return &Manager{
		config:      cfg,
		projectRoot: projectRoot,
	}
}

// Execute runs all hooks for a given event with the provided context
// Returns an error if any hook fails
// Also returns parsed environment variable overrides from hook output
func (m *Manager) Execute(event HookEvent, ctx HookContext) (*EnvOverrides, error) {
	if !event.IsValid() {
		return nil, fmt.Errorf("invalid hook event: %s", event)
	}

	// Get hook scripts for this event from config
	scripts := m.config.GetHookScripts(event.String())
	if len(scripts) == 0 {
		// No hooks defined for this event, not an error
		return NewEnvOverrides(), nil
	}

	fmt.Fprintf(os.Stderr, "[dual] Running %s hooks (%d scripts)...\n", event, len(scripts))

	// Accumulate env overrides from all hooks
	allOverrides := NewEnvOverrides()

	// Execute each hook script in order
	for _, script := range scripts {
		overrides, err := m.executeScript(script, ctx)
		if err != nil {
			return nil, fmt.Errorf("hook %s failed: %w", script, err)
		}
		// Merge overrides (later scripts can override earlier ones)
		allOverrides.Merge(overrides)
	}

	return allOverrides, nil
}

// executeScript executes a single hook script with the given context
// Returns parsed environment variable overrides from stdout
func (m *Manager) executeScript(scriptName string, ctx HookContext) (*EnvOverrides, error) {
	// Construct path to hook script
	hookPath := filepath.Join(m.projectRoot, ".dual", "hooks", scriptName)

	// Check if hook script exists
	info, err := os.Stat(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("hook script not found: %s", hookPath)
		}
		return nil, fmt.Errorf("failed to stat hook script: %w", err)
	}

	// Check if hook is executable (Unix-like systems)
	if info.Mode()&0o111 == 0 {
		fmt.Fprintf(os.Stderr, "[dual] Warning: hook script %s is not executable, attempting to run anyway\n", scriptName)
	}

	// Prepare environment variables
	env := m.buildEnv(ctx)

	// Execute the hook script
	// #nosec G204 - Script path is controlled by config file (trusted source)
	cmd := exec.Command(hookPath)
	cmd.Env = append(os.Environ(), env...)
	cmd.Dir = ctx.ContextPath // Run hook in context directory

	// Capture stdout for parsing env overrides
	var stdout strings.Builder
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdout)
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "[dual] Executing hook: %s\n", scriptName)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("hook execution failed: %w", err)
	}

	// Parse environment variable overrides from stdout
	overrides, err := ParseEnvOverrides(stdout.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse env overrides from hook output: %w", err)
	}

	// Log parsed overrides if any
	if !overrides.IsEmpty() {
		fmt.Fprintf(os.Stderr, "[dual] Hook %s produced %d global and %d service-specific env overrides\n",
			scriptName, len(overrides.Global), len(overrides.Services))
	}

	return overrides, nil
}

// buildEnv constructs the environment variables to pass to the hook script
func (m *Manager) buildEnv(ctx HookContext) []string {
	env := []string{
		fmt.Sprintf("DUAL_EVENT=%s", ctx.Event),
		fmt.Sprintf("DUAL_CONTEXT_NAME=%s", ctx.ContextName),
		fmt.Sprintf("DUAL_CONTEXT_PATH=%s", ctx.ContextPath),
		fmt.Sprintf("DUAL_PROJECT_ROOT=%s", ctx.ProjectRoot),
	}

	return env
}

// ExecuteWithFallback runs hooks but continues even if they fail, logging errors
// This is useful for non-critical hooks like postWorktreeDelete
// Returns parsed environment variable overrides (empty if hooks failed)
func (m *Manager) ExecuteWithFallback(event HookEvent, ctx HookContext) *EnvOverrides {
	overrides, err := m.Execute(event, ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: hook execution failed (continuing anyway): %v\n", err)
		return NewEnvOverrides()
	}
	return overrides
}
