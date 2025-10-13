# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dual` is a CLI tool written in Go that manages port assignments across different development contexts (git branches, worktrees, or clones). It eliminates port conflicts when working on multiple features simultaneously by automatically detecting the context and service, then injecting the appropriate PORT environment variable.

## Core Concepts

### Architecture Components

1. **Command Wrapper**: Primary interface that intercepts commands and injects PORT environment variable
   - Detects current context (git branch, `.dual-context` file, or "default")
   - Detects service based on working directory matching against service paths
   - Calculates port: `port = basePort + serviceIndex + 1`
   - Executes wrapped command with PORT in environment

2. **Registry System** (`~/.dual/registry.json`)
   - Global, cross-project storage for context-to-port mappings
   - Structure: projects → contexts → basePort mappings
   - Tracks worktree paths for multi-location contexts
   - Local to user, never committed

3. **Configuration** (`dual.config.yml`)
   - Project-level service definitions
   - Maps service names to paths and env files
   - Can be committed for team sharing
   - Service order determines port offsets

### Port Calculation Logic

```
port = basePort + serviceIndex + 1

Example with basePort 4100:
  services[0] "web"    → 4101
  services[1] "api"    → 4102
  services[2] "worker" → 4103
```

### Context Detection Priority

1. Git branch name (`git branch --show-current`)
2. `.dual-context` file content (manual override)
3. Fallback to "default"

### Service Detection

Matches current working directory against configured service paths. Longest match wins to support nested structures.

## Commands

### Build & Test

```bash
# Build the binary
go build -o dual ./cmd/dual

# Run tests
go test ./...

# Install locally for testing
go install ./cmd/dual
```

### Core Command Structure

```bash
# Command wrapper (primary use case)
dual <command> [args...]
dual --service <name> <command> [args...]

# Initialization
dual init

# Service management
dual service add <name> --path <path> --env-file <file>

# Context management
dual context create [name] --base-port <port>
dual context

# Port queries
dual port [service]
dual ports

# Utilities
dual open <service>
dual sync
```

## Development Architecture

### Expected Directory Structure

```
cmd/dual/          - CLI entry point and command definitions
internal/
  config/          - dual.config.yml parsing and management
  registry/        - ~/.dual/registry.json operations
  context/         - Context detection logic
  service/         - Service detection and port calculation
  wrapper/         - Command execution with environment injection
pkg/               - Public/reusable packages if any
```

### Key Implementation Details

**Environment Variable Injection**:
- Never write PORT to files (Vercel-proof design)
- Inject PORT only into command execution environment
- Preserve all other environment variables from parent shell

**Registry Operations**:
- Thread-safe file I/O (multiple dual instances may run simultaneously)
- Atomic writes to prevent corruption
- Handle missing/corrupt registry gracefully

**Git Integration**:
- Use `git rev-parse --show-toplevel` to find project root
- Use `git branch --show-current` for context detection
- Support worktrees and multiple clones of same repository

**Service Path Matching**:
- Normalize paths (resolve symlinks, relative → absolute)
- Match against current working directory
- Choose longest matching path for nested services

## Integration Points

### Vercel Integration
- Never modify `.vercel/.env.development.local`
- Support it as envFile target for `sync` command fallback
- Command wrapper bypasses file-based PORT entirely

### Framework Compatibility
- Universal: works with any framework/tool that respects PORT env var
- Common targets: Next.js, Vite, Express, etc.
- No framework-specific logic in core

### CI/CD Considerations
- `dual sync` command writes PORT to env files as fallback
- Binary can be installed in CI for consistency
- Provide direct download URLs for CI installation scripts

## Configuration File Format

### dual.config.yml
```yaml
version: 1  # Config schema version
services:
  <name>:
    path: <relative-path>      # From project root
    envFile: <env-file-path>   # For sync command
```

### ~/.dual/registry.json
```json
{
  "projects": {
    "<absolute-project-path>": {
      "contexts": {
        "<context-name>": {
          "basePort": <integer>,
          "path": "<absolute-context-path>",  # Optional, for worktrees
          "created": "<ISO-8601-timestamp>"
        }
      }
    }
  }
}
```

## Testing Considerations

- Test context detection across different git states
- Test service matching with various directory structures
- Test port calculation with different service counts
- Test command wrapper with various shell commands
- Test concurrent registry access
- Mock filesystem operations for unit tests
- Integration tests with actual git repositories

## Error Handling Patterns

- Configuration not found → helpful init instructions
- Context not registered → suggest `dual context create`
- Service not detected → list available services, show CWD
- Registry corruption → attempt recovery, clear instructions
- Git commands fail → graceful fallback to "default" context
- Port already in use → informational only (don't prevent execution)

## Output Design

All dual-specific output should use prefix format:
```
[dual] Context: main | Service: web | Port: 4101
```

Show actual command being executed for transparency:
```
[dual] Executing: PORT=4101 pnpm dev
```
