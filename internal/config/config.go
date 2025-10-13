package config

// Config represents the dual.config.yml structure
type Config struct {
	Services map[string]Service `yaml:"services"`
	Version  int                `yaml:"version"`
}

// Service represents a single service configuration
type Service struct {
	Path    string `yaml:"path"`
	EnvFile string `yaml:"envFile"`
}
