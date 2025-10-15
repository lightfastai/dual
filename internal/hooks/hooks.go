package hooks

import (
	"fmt"
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
// Returns an error if any hook fails (unless continueOnError is true)
func (m *Manager) Execute(event HookEvent, ctx HookContext) error {
	if !event.IsValid() {
		return fmt.Errorf("invalid hook event: %s", event)
	}

	// Get hook scripts for this event from config
	scripts := m.config.GetHookScripts(event.String())
	if len(scripts) == 0 {
		// No hooks defined for this event, not an error
		return nil
	}

	fmt.Fprintf(os.Stderr, "[dual] Running %s hooks (%d scripts)...\n", event, len(scripts))

	// Execute each hook script in order
	for _, script := range scripts {
		if err := m.executeScript(script, ctx); err != nil {
			return fmt.Errorf("hook %s failed: %w", script, err)
		}
	}

	return nil
}

// executeScript executes a single hook script with the given context
func (m *Manager) executeScript(scriptName string, ctx HookContext) error {
	// Construct path to hook script
	hookPath := filepath.Join(m.projectRoot, ".dual", "hooks", scriptName)

	// Check if hook script exists
	info, err := os.Stat(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("hook script not found: %s", hookPath)
		}
		return fmt.Errorf("failed to stat hook script: %w", err)
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
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintf(os.Stderr, "[dual] Executing hook: %s\n", scriptName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook execution failed: %w", err)
	}

	return nil
}

// buildEnv constructs the environment variables to pass to the hook script
func (m *Manager) buildEnv(ctx HookContext) []string {
	env := []string{
		fmt.Sprintf("DUAL_EVENT=%s", ctx.Event),
		fmt.Sprintf("DUAL_CONTEXT_NAME=%s", ctx.ContextName),
		fmt.Sprintf("DUAL_CONTEXT_PATH=%s", ctx.ContextPath),
		fmt.Sprintf("DUAL_PROJECT_ROOT=%s", ctx.ProjectRoot),
		fmt.Sprintf("DUAL_BASE_PORT=%d", ctx.BasePort),
	}

	// Add service-specific port variables
	for serviceName, port := range ctx.ServicePorts {
		// Normalize service name for env var (replace hyphens with underscores, uppercase)
		envVarName := normalizeServiceName(serviceName)
		env = append(env, fmt.Sprintf("DUAL_PORT_%s=%d", envVarName, port))
	}

	return env
}

// normalizeServiceName converts a service name to a valid environment variable suffix
// Example: "my-api" -> "MY_API", "web" -> "WEB"
func normalizeServiceName(name string) string {
	// Replace hyphens with underscores
	normalized := strings.ReplaceAll(name, "-", "_")
	// Convert to uppercase
	normalized = strings.ToUpper(normalized)
	return normalized
}

// ExecuteWithFallback runs hooks but continues even if they fail, logging errors
// This is useful for non-critical hooks like postWorktreeDelete
func (m *Manager) ExecuteWithFallback(event HookEvent, ctx HookContext) {
	if err := m.Execute(event, ctx); err != nil {
		fmt.Fprintf(os.Stderr, "[dual] Warning: hook execution failed (continuing anyway): %v\n", err)
	}
}
