# Config Package

This package provides configuration file parsing for the dual CLI tool.

## Overview

The config package handles loading and validating `dual.config.yml` files. It searches up the directory tree from the current working directory to find the configuration file, similar to how tools like `.git` or `package.json` are discovered.

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
```

## Configuration File Format

The configuration file must be named `dual.config.yml` and placed at the project root:

```yaml
version: 1

services:
  web:
    path: apps/web              # Required: relative path from project root
    envFile: apps/web/.env.local # Optional: env file path for sync command

  api:
    path: apps/api
    envFile: apps/api/.env.local

  worker:
    path: apps/worker           # envFile is optional
```

## Validation

The package validates:

- **Version**: Must be `1` (currently the only supported version)
- **Services**: At least one service must be defined
- **Service paths**:
  - Must be relative (not absolute)
  - Must point to an existing directory
  - Path is resolved relative to project root
- **Service envFile** (optional):
  - Must be relative (not absolute)
  - The directory containing the file must exist
  - The file itself doesn't need to exist (will be created by sync command)

## API

### Types

```go
type Config struct {
    Version  int                `yaml:"version"`
    Services map[string]Service `yaml:"services"`
}

type Service struct {
    Path    string `yaml:"path"`
    EnvFile string `yaml:"envFile"`
}
```

### Functions

- `LoadConfig() (*Config, string, error)` - Searches for and loads config from current directory or parents. Returns config, project root path, and error.
- `LoadConfigFrom(path string) (*Config, error)` - Loads config from a specific file path (useful for testing).

### Constants

- `ConfigFileName` - The name of the config file (`"dual.config.yml"`)
- `SupportedVersion` - The currently supported config version (`1`)

## Error Handling

The package provides descriptive error messages:

- Missing config file: `"no dual.config.yml found in current directory or any parent directory"`
- Invalid version: `"unsupported config version X (expected 1)"`
- Missing services: `"at least one service must be defined"`
- Invalid service path: `"service \"web\": path does not exist: apps/web"`
- Absolute paths: `"service \"web\": path must be relative to project root"`

## Testing

Run tests with:

```bash
go test ./internal/config
go test ./internal/config -cover  # With coverage
```

The test suite includes:
- YAML parsing tests
- Config validation tests
- Service validation tests
- Directory tree walking tests
- Error handling tests

Current test coverage: ~91%
