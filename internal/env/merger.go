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

// LoadLayeredEnv loads a layered environment for a given context
// projectRoot: The root directory of the project
// cfg: The dual configuration
// contextName: The name of the current context
// overrides: Context-specific overrides from registry
// Note: This function does NOT load service-specific .env files. Use the manual approach in run.go instead.
func LoadLayeredEnv(projectRoot string, cfg *config.Config, contextName string, overrides map[string]string) (*LayeredEnv, error) {
	loader := NewLoader()
	env := &LayeredEnv{
		Base:      make(map[string]string),
		Service:   make(map[string]string),
		Overrides: make(map[string]string),
	}

	// Load base environment file if configured
	if cfg.Env.BaseFile != "" {
		baseFilePath := filepath.Join(projectRoot, cfg.Env.BaseFile)
		baseEnv, err := loader.LoadEnvFile(baseFilePath)
		if err != nil {
			// Non-fatal: log warning but continue
			// The file might not exist yet, which is OK
		} else {
			env.Base = baseEnv
		}
	}

	// Add context overrides from registry
	if overrides != nil {
		env.Overrides = overrides
	}

	return env, nil
}
