# Config Package

This package provides configuration file parsing and validation for the dual CLI tool.

## Overview

The config package handles loading and validating `dual.config.yml` files. It searches up the directory tree from the current working directory to find the configuration file, similar to how tools like `.git` or `package.json` are discovered.

The config layer is responsible for:
- Parsing YAML configuration with schema validation
- Validating service definitions and paths
- Validating worktree configuration (path, naming patterns)
- Validating hook definitions and script references
- Providing thread-safe file I/O with atomic writes
- Resolving project identifiers for registry normalization

## Usage

```go
import "github.com/lightfastai/dual/internal/config"

// Load config from current directory or parent directories
config, projectRoot, err := config.LoadConfig()
if err != nil {
    log.Fatal(err)
}

// Access services
for name, service := range config.Services {
    fmt.Printf("Service: %s, Path: %s\n", name, service.Path)
}

// Get worktree configuration
worktreePath := config.GetWorktreePath(projectRoot)
worktreeName := config.GetWorktreeName("feature/my-branch")

// Get hook scripts for an event
postCreateHooks := config.GetHookScripts("postWorktreeCreate")
```

## Configuration File Format

The configuration file must be named `dual.config.yml` and placed at the project root:

```yaml
version: 1

services:
  web:
    path: apps/web              # Required: relative path from project root
    envFile: apps/web/.env.local # Optional: env file path (for reference)

  api:
    path: apps/api
    envFile: apps/api/.env.local

  worker:
    path: apps/worker           # envFile is optional

worktrees:
  path: ../worktrees            # Optional: where to create worktrees (relative to project root)
  naming: "{branch}"            # Optional: directory naming pattern (default: "{branch}")

hooks:
  postWorktreeCreate:           # Optional: hooks to run after worktree creation
    - setup-database.sh
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:            # Optional: hooks to run before worktree deletion
    - backup-data.sh
    - cleanup-database.sh
  postWorktreeDelete:           # Optional: hooks to run after worktree deletion
    - notify-team.sh
```

## Configuration Schema

### Root Fields

- **`version`** (int, required): Config schema version. Must be `1`.
- **`services`** (map, optional): Service definitions (name → Service)
- **`worktrees`** (WorktreeConfig, optional): Worktree management configuration
- **`hooks`** (map, optional): Lifecycle hooks (event → script list)
- **`env`** (EnvConfig, optional): Environment configuration (reserved for future use)

### Service Definition

Each service in the `services` map has:

- **`path`** (string, required): Relative path from project root to service directory
- **`envFile`** (string, optional): Relative path to environment file (for reference)

### WorktreeConfig

- **`path`** (string, optional): Base directory for worktrees, relative to project root
  - Example: `../worktrees` creates a sibling directory
  - Default: `../worktrees` (if not specified)
  - Can be parent directory (`..`) or subdirectory (`./worktrees`)
- **`naming`** (string, optional): Directory naming pattern for worktrees
  - Supports `{branch}` placeholder (replaced with branch name)
  - Example: `{branch}` → `feature/my-branch`
  - Example: `wt-{branch}` → `wt-feature/my-branch`
  - Default: `{branch}`

### Hooks Configuration

Map of hook events to script lists. Valid events:

- **`postWorktreeCreate`**: Runs after creating worktree and registering context
- **`preWorktreeDelete`**: Runs before deleting worktree (files still exist)
- **`postWorktreeDelete`**: Runs after deleting worktree and removing from registry

Script paths are relative to `$PROJECT_ROOT/.dual/hooks/` directory.

## Validation Rules

### Version Validation
- Version field is required
- Must equal `1` (currently the only supported version)
- Error if missing or unsupported version

### Service Validation
- Services map can be empty (for initial setup)
- For each service:
  - Name cannot be empty
  - `path` is required
  - `path` must be relative (not absolute)
  - `path` must point to an existing directory
  - `envFile` (if provided) must be relative
  - `envFile` directory must exist (file itself doesn't need to exist)

### Worktree Validation
- `worktrees.path` (if provided) must be relative (not absolute)
- Directory doesn't need to exist (created by `dual create`)
- `worktrees.naming` can be any string with `{branch}` placeholder

### Hook Validation
- Hook events must be one of: `postWorktreeCreate`, `preWorktreeDelete`, `postWorktreeDelete`
- Invalid hook events produce an error with valid event list
- Hook scripts are validated for existence (warning only, not error)
- Script paths resolved as `$PROJECT_ROOT/.dual/hooks/{script}`

## API

### Types

```go
type Config struct {
    Version   int                 `yaml:"version"`
    Services  map[string]Service  `yaml:"services"`
    Worktrees WorktreeConfig      `yaml:"worktrees,omitempty"`
    Hooks     map[string][]string `yaml:"hooks,omitempty"`
    Env       EnvConfig           `yaml:"env,omitempty"`
}

type Service struct {
    Path    string `yaml:"path"`
    EnvFile string `yaml:"envFile"`
}

type WorktreeConfig struct {
    Path   string `yaml:"path,omitempty"`
    Naming string `yaml:"naming,omitempty"`
}

type EnvConfig struct {
    BaseFile string `yaml:"baseFile,omitempty"`
}
```

### Functions

- **`LoadConfig() (*Config, string, error)`** - Searches for and loads config from current directory or parents. Returns config, project root path, and error.
- **`LoadConfigFrom(path string) (*Config, error)`** - Loads config from a specific file path (useful for testing).
- **`SaveConfig(config *Config, path string) error`** - Writes config to path atomically (temp file + rename).
- **`GetProjectIdentifier(projectRoot string) (string, error)`** - Returns normalized project identifier for registry. For worktrees, returns parent repo path so all worktrees share the same registry entry.
- **`(c *Config) GetWorktreePath(projectRoot string) string`** - Returns absolute path to worktrees directory.
- **`(c *Config) GetWorktreeName(branchName string) string`** - Returns worktree directory name for a branch using naming pattern.
- **`(c *Config) GetHookScripts(event string) []string`** - Returns hook scripts for an event, or nil if none.

### Constants

- `ConfigFileName` - The name of the config file (`"dual.config.yml"`)
- `SupportedVersion` - The currently supported config version (`1`)

## Error Handling

The package provides descriptive error messages:

- Missing config file: `"no dual.config.yml found in current directory or any parent directory"`
- Invalid version: `"unsupported config version X (expected 1)"`
- Invalid version (missing): `"version field is required"`
- Invalid service path: `"service \"web\": path does not exist: apps/web"`
- Absolute service path: `"service \"web\": path must be relative to project root"`
- Absolute worktree path: `"worktrees.path must be relative to project root, got absolute path: /foo"`
- Invalid hook event: `"hooks: invalid hook event: badEvent (valid events: postWorktreeCreate, preWorktreeDelete, postWorktreeDelete)"`
- Missing hook script: `"[dual] Warning: hook script not found: /path/to/script"` (warning, not error)

## Project Root and Worktree Handling

The config loader automatically handles both main repositories and worktrees:

1. **Config Discovery**: Walks up directory tree from current directory
2. **Project Root**: Directory where `dual.config.yml` is found
3. **Service Path Resolution**: All service paths resolved relative to project root
4. **Worktree Support**: Same config file used in both main repo and worktrees
5. **Registry Normalization**: `GetProjectIdentifier()` ensures all worktrees of a repo share the same registry entry

This design allows a single config file to work correctly in both the main repository and all its worktrees.

## Thread Safety

- Config loading is read-only and inherently thread-safe
- `SaveConfig()` uses atomic write pattern: write to temp file → rename
- Atomic rename ensures no partial writes visible to readers
- Multiple readers can safely load config concurrently
- Writers should coordinate externally (config rarely written)

## Migration Notes (v0.2.x → v0.3.0)

### Removed Fields
- Port management configuration (no longer supported)
- Global registry references (registry is now project-local)

### Added Fields
- `worktrees` section for worktree management configuration
- `hooks` section for lifecycle hook definitions
- `env` section (reserved for future use)

### Breaking Changes
- Services can now be empty (previously required at least one)
- Hook validation warnings instead of errors for missing scripts
- Registry moved from global to project-local (`$PROJECT_ROOT/.dual/.local/registry.json`)

### Migration Steps
1. Add `worktrees` section to config if using `dual create`
2. Add `hooks` section if using lifecycle hooks
3. Remove any port-related configuration (no longer used)
4. Ensure `.dual/` directory in `.gitignore` (registry should not be committed)

## Testing

Run tests with:

```bash
go test ./internal/config
go test ./internal/config -cover  # With coverage
```

The test suite includes:
- YAML parsing tests
- Config validation tests (version, services, worktrees, hooks)
- Service path validation tests
- Worktree configuration validation tests
- Hook validation tests
- Directory tree walking tests
- Error handling tests
- Atomic write tests

Current test coverage: ~91%
