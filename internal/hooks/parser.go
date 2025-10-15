package hooks

import (
	"fmt"
	"strings"
)

// EnvOverrides represents environment variable overrides parsed from hook output
type EnvOverrides struct {
	// Global contains environment variables that apply to all services
	Global map[string]string

	// Services contains service-specific environment variable overrides
	// Map structure: serviceName -> (key -> value)
	Services map[string]map[string]string
}

// NewEnvOverrides creates a new empty EnvOverrides
func NewEnvOverrides() *EnvOverrides {
	return &EnvOverrides{
		Global:   make(map[string]string),
		Services: make(map[string]map[string]string),
	}
}

// ParseEnvOverrides parses hook stdout into structured environment variable overrides
// Format:
//   GLOBAL:KEY=VALUE -> global override that applies to all services
//   service:KEY=VALUE -> service-specific override
//
// Examples:
//   GLOBAL:DATABASE_URL=postgres://localhost/db
//   api:PORT=4201
//   web:PORT=4202
//   api:API_KEY=secret123
//
// Lines that don't match the format are silently ignored (allows hooks to print other output)
func ParseEnvOverrides(output string) (*EnvOverrides, error) {
	overrides := NewEnvOverrides()

	// Split output into lines
	lines := strings.Split(output, "\n")

	for lineNum, line := range lines {
		// Trim whitespace
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse line: scope:KEY=VALUE
		// Find first colon (separates scope from key=value)
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			// No colon found, skip line (not an env override)
			continue
		}

		scope := strings.TrimSpace(line[:colonIdx])
		keyValue := strings.TrimSpace(line[colonIdx+1:])

		// Parse KEY=VALUE
		equalsIdx := strings.Index(keyValue, "=")
		if equalsIdx == -1 {
			// No equals sign found, skip line
			continue
		}

		key := strings.TrimSpace(keyValue[:equalsIdx])
		value := keyValue[equalsIdx+1:] // Don't trim value - preserve whitespace

		// Validate key is not empty
		if key == "" {
			return nil, fmt.Errorf("line %d: empty key in override: %q", lineNum+1, line)
		}

		// Add to appropriate scope
		if strings.ToUpper(scope) == "GLOBAL" {
			overrides.Global[key] = value
		} else {
			// Service-specific override
			serviceName := scope
			if overrides.Services[serviceName] == nil {
				overrides.Services[serviceName] = make(map[string]string)
			}
			overrides.Services[serviceName][key] = value
		}
	}

	return overrides, nil
}

// Merge merges multiple EnvOverrides into one
// Later overrides take precedence over earlier ones
func (e *EnvOverrides) Merge(other *EnvOverrides) {
	// Merge global overrides
	for k, v := range other.Global {
		e.Global[k] = v
	}

	// Merge service-specific overrides
	for serviceName, serviceOverrides := range other.Services {
		if e.Services[serviceName] == nil {
			e.Services[serviceName] = make(map[string]string)
		}
		for k, v := range serviceOverrides {
			e.Services[serviceName][k] = v
		}
	}
}

// IsEmpty returns true if there are no overrides
func (e *EnvOverrides) IsEmpty() bool {
	return len(e.Global) == 0 && len(e.Services) == 0
}
