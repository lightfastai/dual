package registry

import "time"

// Registry represents the global registry structure stored in ~/.dual/registry.json
type Registry struct {
	Projects map[string]Project `json:"projects"`
}

// Project represents a single project in the registry
type Project struct {
	Contexts map[string]Context `json:"contexts"`
}

// Context represents a development context (branch, worktree, etc.)
type Context struct {
	Created  time.Time `json:"created"`
	Path     string    `json:"path,omitempty"`
	BasePort int       `json:"basePort"`
}
