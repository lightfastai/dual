package env

import (
	"fmt"
	"path/filepath"

	"github.com/lightfastai/dual/internal/config"
)

// LayeredEnv represents a layered environment with multiple sources
type LayeredEnv struct {
	Base      map[string]string // Base environment from file
	Service   map[string]string // Service-specific environment from <service-path>/.env
	Overrides map[string]string // Context-specific overrides
}

// Merge merges all layers into a single environment map
// Priority (lowest to highest): Base → Service → Overrides
func (e *LayeredEnv) Merge() map[string]string {
	result := make(map[string]string)

	// Layer 1: Base environment
	for k, v := range e.Base {
		result[k] = v
	}

	// Layer 2: Service-specific environment
	for k, v := range e.Service {
		result[k] = v
	}

	// Layer 3: Context overrides
	for k, v := range e.Overrides {
		result[k] = v
	}

	return result
}

// ToSlice converts the merged environment to a slice of KEY=value strings
func (e *LayeredEnv) ToSlice() []string {
	merged := e.Merge()
	result := make([]string, 0, len(merged))

	for k, v := range merged {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}

// Stats returns statistics about the environment layers
func (e *LayeredEnv) Stats() EnvStats {
	return EnvStats{
		BaseVars:     len(e.Base),
		ServiceVars:  len(e.Service),
		OverrideVars: len(e.Overrides),
		TotalVars:    len(e.Merge()),
	}
}

// EnvStats contains statistics about environment layers
type EnvStats struct {
	BaseVars     int
	ServiceVars  int
	OverrideVars int
	TotalVars    int
}

// LoadLayeredEnv loads a layered environment for a given context with all three layers:
// 1. Base environment from the configured base file
// 2. Service-specific environment from the service's .env file
// 3. Context-specific overrides (from registry or filesystem)
//
// Parameters:
//   - projectRoot: The root directory of the project
//   - cfg: The dual configuration
//   - serviceName: The name of the service (empty string for no service)
//   - contextName: The name of the current context (empty string for no context)
//   - overrides: Context-specific overrides from registry (can be nil)
func LoadLayeredEnv(projectRoot string, cfg *config.Config, serviceName string, contextName string, overrides map[string]string) (*LayeredEnv, error) {
	loader := NewLoader()
	env := &LayeredEnv{
		Base:      make(map[string]string),
		Service:   make(map[string]string),
		Overrides: make(map[string]string),
	}

	// Layer 1: Load base environment file if configured
	if cfg.Env.BaseFile != "" {
		baseFilePath := filepath.Join(projectRoot, cfg.Env.BaseFile)
		baseEnv, err := loader.LoadEnvFile(baseFilePath)
		if err != nil {
			// Non-fatal: The file might not exist yet, which is OK
			// Just continue with empty base environment
		} else {
			env.Base = baseEnv
		}
	}

	// Layer 2: Load service-specific environment file
	// In worktrees, load from both parent repo and worktree, with worktree overriding
	if serviceName != "" {
		if service, ok := cfg.Services[serviceName]; ok {
			serviceEnv := make(map[string]string)

			// Determine relative env file path
			var relativeEnvPath string
			if service.EnvFile != "" {
				relativeEnvPath = service.EnvFile
			} else {
				relativeEnvPath = filepath.Join(service.Path, ".env")
			}

			// First, try to load from parent repo (if we're in a worktree)
			projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
			if err == nil && projectIdentifier != projectRoot {
				// We're in a worktree, load parent repo's service env first
				parentEnvPath := filepath.Join(projectIdentifier, relativeEnvPath)
				parentEnv, err := loader.LoadEnvFile(parentEnvPath)
				if err == nil {
					// Merge parent repo env into service env (lowest priority)
					for k, v := range parentEnv {
						serviceEnv[k] = v
					}
				}
			}

			// Then, load from worktree (overrides parent repo)
			worktreeEnvPath := filepath.Join(projectRoot, relativeEnvPath)
			worktreeEnv, err := loader.LoadEnvFile(worktreeEnvPath)
			if err == nil {
				// Merge worktree env into service env (higher priority, overrides parent)
				for k, v := range worktreeEnv {
					serviceEnv[k] = v
				}
			}

			env.Service = serviceEnv
		}
	}

	// Layer 3: Add context-specific overrides
	// First try to use provided overrides (from registry)
	if overrides != nil {
		env.Overrides = overrides
	} else if contextName != "" && serviceName != "" {
		// If no overrides provided but we have context and service,
		// try to load from filesystem (.dual/.local/service/<service>/.env)
		// Get the parent repo root for override files (worktrees share parent's .dual/)
		projectIdentifier, err := config.GetProjectIdentifier(projectRoot)
		if err != nil {
			// Fallback to projectRoot if not a git repo
			projectIdentifier = projectRoot
		}

		overridesPath := filepath.Join(projectIdentifier, ".dual", ".local", "service", serviceName, ".env")
		overridesEnv, err := loader.LoadEnvFile(overridesPath)
		if err == nil {
			env.Overrides = overridesEnv
		}
		// Non-fatal: if overrides file doesn't exist, continue with empty overrides
	}

	return env, nil
}
