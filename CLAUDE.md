# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dual` is a CLI tool written in Go that manages port assignments across different development contexts (git branches, worktrees, or clones). It eliminates port conflicts when working on multiple features simultaneously by automatically detecting the context and service, then injecting the appropriate PORT environment variable.

## Core Architecture

### Command Flow

The tool operates in two primary modes:

1. **Command Wrapper Mode** (primary): Intercepts arbitrary commands via `dual <command>` and injects PORT
2. **Management Mode**: Direct subcommands like `dual init`, `dual service add`, `dual context create`, `dual create`, `dual delete`

The main entry point (cmd/dual/main.go:160-234) implements custom argument parsing to distinguish between wrapper mode and management mode before delegating to cobra commands.

### Key Components

**Config Layer** (`internal/config/`)
- Loads and validates `dual.config.yml` from project root (or parent directories)
- Schema version 1 with services map (name → path, envFile)
- Supports optional worktrees configuration (path, naming pattern)
- Supports optional hooks configuration (lifecycle event → script list)
- Validates paths exist and are relative to project root
- Thread-safe file I/O with atomic writes

**Registry Layer** (`internal/registry/`)
- Project-local state in `$PROJECT_ROOT/.dual/registry.json` (per-project, not committed)
- Structure: projects → contexts → basePort mappings
- All worktrees of a repository share the parent repo's registry via `GetProjectIdentifier()` normalization
- Thread-safe with sync.RWMutex for concurrent dual instances
- Atomic writes via temp file + rename pattern
- Auto-recovers from corruption (returns empty registry)

**Hook System** (`internal/hooks/`)
- Executes lifecycle hooks at key worktree management points
- Hook events: `postWorktreeCreate`, `postPortAssign`, `preWorktreeDelete`, `postWorktreeDelete`
- Scripts located in `$PROJECT_ROOT/.dual/hooks/` and configured in `dual.config.yml`
- Receives context via environment variables: `DUAL_EVENT`, `DUAL_CONTEXT_NAME`, `DUAL_CONTEXT_PATH`, `DUAL_PROJECT_ROOT`, `DUAL_BASE_PORT`, `DUAL_PORT_<SERVICE>`
- Non-zero exit codes fail the operation and halt execution

**Context Detection** (`internal/context/`)
- Priority: git branch → `.dual-context` file → "default"
- Simple, no external dependencies beyond git command

**Service Detection** (`internal/service/detector.go`)
- Matches current working directory against service paths from config
- Resolves symlinks for consistency across worktrees
- Uses longest path match for nested service structures
- Returns `ErrServiceNotDetected` if no match found

**Port Calculation** (`internal/service/calculator.go`)
- Formula: `port = basePort + serviceIndex + 1`
- Services ordered alphabetically for determinism (not config order!)
- Example: basePort 4100 → services: api→4101, web→4102, worker→4103

### Command Wrapper Implementation

The wrapper (cmd/dual/main.go:78-157) follows this sequence:
1. Load config and detect context
2. Detect or validate service (supports `--service` override)
3. Calculate port from registry
4. Execute command with PORT injected into environment
5. Stream stdout/stderr in real-time, preserve exit codes

### Worktree Management Commands

**`dual create <branch>`** - Creates a new git worktree with integrated context setup:
1. Creates git worktree at configured location (from `worktrees.path` in config)
2. Names directory using configured pattern (from `worktrees.naming`, defaults to `{branch}`)
3. Registers context in project-local registry
4. Assigns base port for the context
5. Executes `postWorktreeCreate` hooks
6. Calculates and assigns ports for all services
7. Executes `postPortAssign` hooks

**`dual delete <context>`** - Deletes a worktree with cleanup:
1. Validates context exists and is a worktree
2. Executes `preWorktreeDelete` hooks
3. Removes git worktree
4. Removes context from registry
5. Executes `postWorktreeDelete` hooks

Both commands integrate with the hook system to enable custom automation (database setup, environment configuration, cleanup tasks, etc.).

## Hook System

The dual CLI supports lifecycle hooks that run at key points during worktree management. Hooks enable custom automation like database branch creation, environment setup, and cleanup tasks.

### Hook Events

- **`postWorktreeCreate`**: Runs after creating a git worktree, before port assignment
- **`postPortAssign`**: Runs after assigning ports to a context
- **`preWorktreeDelete`**: Runs before deleting a worktree, while files still exist
- **`postWorktreeDelete`**: Runs after deleting a worktree and removing from registry

### Hook Configuration

Hook scripts are stored in `$PROJECT_ROOT/.dual/hooks/` and configured in `dual.config.yml`:

```yaml
version: 1

services:
  web:
    path: ./apps/web
  api:
    path: ./apps/api

worktrees:
  path: ../worktrees
  naming: "{branch}"

hooks:
  postWorktreeCreate:
    - setup-database.sh
    - install-dependencies.sh
  postPortAssign:
    - update-env-files.sh
  preWorktreeDelete:
    - backup-data.sh
  postWorktreeDelete:
    - cleanup-database.sh
```

### Hook Environment Variables

Hook scripts receive the following environment variables:

- **`DUAL_EVENT`**: The hook event name (e.g., `postWorktreeCreate`)
- **`DUAL_CONTEXT_NAME`**: Context name (usually the branch name)
- **`DUAL_CONTEXT_PATH`**: Absolute path to the worktree directory
- **`DUAL_PROJECT_ROOT`**: Absolute path to the main repository
- **`DUAL_BASE_PORT`**: Base port assigned to this context
- **`DUAL_PORT_<SERVICE>`**: Port for each service (e.g., `DUAL_PORT_WEB=4101`, `DUAL_PORT_API=4102`)

### Hook Script Example

```bash
#!/bin/bash
# .dual/hooks/setup-database.sh

set -e

echo "Setting up database for context: $DUAL_CONTEXT_NAME"
echo "Context path: $DUAL_CONTEXT_PATH"
echo "Base port: $DUAL_BASE_PORT"

# Create a database branch
createdb "myapp_${DUAL_CONTEXT_NAME}"

# Update connection string in .env file
cat > "$DUAL_CONTEXT_PATH/.env" <<EOF
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
PORT=$DUAL_PORT_WEB
API_PORT=$DUAL_PORT_API
EOF

echo "Database setup complete"
```

### Hook Execution Rules

- Scripts must be executable (`chmod +x`)
- Scripts run in sequence (not parallel)
- Non-zero exit code halts execution and fails the operation
- stdout/stderr are streamed to the user in real-time
- Scripts run with the worktree directory as working directory (except `postWorktreeDelete`)
- Hook failure during `dual create` leaves the worktree in place but may be partially configured
- Hook failure during `dual delete` halts deletion - worktree and registry entry remain

## Build and Test Commands

### Build
```bash
# Build binary
go build -o dual ./cmd/dual

# Install to GOPATH/bin
go install ./cmd/dual

# Build with version info (used in CI/releases)
go build -ldflags="-X main.version=1.0.0 -X main.commit=abc123 -X main.date=2024-01-01" -o dual ./cmd/dual
```

### Testing
```bash
# Unit tests (internal/ packages)
go test -v -race -coverprofile=coverage.out ./internal/...

# Integration tests (test/integration/)
go test -v -timeout=10m ./test/integration/...

# All tests
go test ./...

# Run specific test
go test -v -run TestCalculatePort ./internal/service/...

# Run tests with coverage
go test -cover ./...
```

### Linting
```bash
# Run golangci-lint (requires golangci-lint installed)
golangci-lint run

# Run with auto-fix
golangci-lint run --fix

# Specific linters (configured in .golangci.yml)
golangci-lint run --enable-only=gosec
```

### Git Configuration for Integration Tests
Integration tests require git user config:
```bash
git config --global user.email "test@example.com"
git config --global user.name "Test User"
```

## Development Patterns

### Error Handling
- Use sentinel errors for expected conditions (e.g., `ErrServiceNotDetected`, `ErrContextNotFound`)
- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Check sentinel errors with `errors.Is(err, sentinel)`
- Provide actionable hints in error messages (e.g., "Run 'dual init' to create...")

### Thread Safety
- Registry uses `sync.RWMutex` - always lock before accessing Projects map
- Config saves use atomic write pattern: write temp file → rename
- Multiple dual instances can run concurrently

### Dependency Injection in Tests
Detectors accept function parameters for git commands, getwd, evalSymlinks to enable mocking in tests. See internal/service/detector.go:20-27.

### Security Notes
- Commands executed via exec.Command are intentional (gosec G204 suppressed)
- File reads from config paths are trusted (gosec G304 suppressed)
- Test files excluded from most linters (see .golangci.yml)

### Port Calculation Gotcha
Services are sorted **alphabetically** for port assignment, not by order in config file. This ensures deterministic ports even if config order changes.

### Registry Location and Sharing
The registry is now **project-local** at `$PROJECT_ROOT/.dual/registry.json`, not global. This means:
- Each project has its own registry file
- All worktrees of a repository share the parent repo's registry (normalized via `GetProjectIdentifier()`)
- The registry should be added to `.gitignore` to avoid committing port assignments
- Migration from global `~/.dual/registry.json` is not automatic - contexts must be recreated

## Configuration Schema

The `dual.config.yml` file supports the following structure:

```yaml
version: 1

services:
  # Required: Map of service name to configuration
  web:
    path: ./apps/web           # Required: relative path from project root
    envFile: .env.local        # Optional: env file to update (deprecated - use hooks)
  api:
    path: ./apps/api

worktrees:
  # Optional: Worktree management configuration
  path: ../worktrees           # Where to create worktrees (relative to project root)
  naming: "{branch}"           # Directory naming pattern ({branch} is replaced)

hooks:
  # Optional: Lifecycle hooks (scripts relative to .dual/hooks/)
  postWorktreeCreate:
    - setup-database.sh
    - install-dependencies.sh
  postPortAssign:
    - update-env-files.sh
  preWorktreeDelete:
    - backup-data.sh
  postWorktreeDelete:
    - cleanup-database.sh
```

### Configuration Notes

- The `services` section is **required** and must have at least one service
- The `worktrees` section is **optional** - if omitted, `dual create` will fail with a helpful error
- The `worktrees.path` is relative to the project root (e.g., `../worktrees` creates a sibling directory)
- The `worktrees.naming` pattern currently only supports `{branch}` placeholder
- The `hooks` section is **optional** - if omitted, no hooks will run
- Hook script paths are relative to `$PROJECT_ROOT/.dual/hooks/` directory
- Hook scripts must be executable and are run in the order listed

## GitHub Workflow

### Issue Labels
All issues must be labeled with appropriate combinations from these categories:

**Type** (required): `enhancement`, `bug`, `documentation`, `testing`, `refactor`
**Area** (required): `core`, `command`, `infrastructure`, `environment`, `config`
**Priority** (recommended): `critical`, `high`, `medium`, `low`
**Status** (optional): `blocked`, `help-wanted`, `good-first-issue`, `epic`

Example issue labeling:
- New command → `enhancement`, `command`, `medium`
- Registry corruption bug → `bug`, `core`, `critical`
- Release automation → `enhancement`, `infrastructure`, `high`
- Tracking issue → `epic` + relevant area labels

Use `gh issue create` or `gh issue edit` to manage labels via CLI.

## File Locations

### User Data
- Registry: `$PROJECT_ROOT/.dual/registry.json` (project-local, should not be committed - add to .gitignore)
- Config: `dual.config.yml` (project root, can be committed)
- Hooks: `$PROJECT_ROOT/.dual/hooks/` (project-local, can be committed)
- Context override: `.dual-context` (worktree/clone specific)

### Code Structure
```
cmd/dual/          Command definitions (cobra commands)
internal/config/   Config file parsing and validation
internal/registry/ Project-local registry operations
internal/context/  Context detection logic
internal/service/  Service detection and port calculation
internal/hooks/    Hook execution and lifecycle management
test/integration/  End-to-end workflow tests
```

## CI/CD

The project uses GitHub Actions (.github/workflows/test.yml):
- Unit tests with race detector and coverage
- Integration tests with 10m timeout
- golangci-lint with 5m timeout
- Build verification

All jobs must pass for successful CI run (test-summary job).

## Design Principles

1. **Vercel-proof**: Never write PORT to files that might be overwritten by external tools
2. **Transparent**: Always show user what command will execute and what port is used
3. **Universal**: Works with any command/framework that respects PORT env var
4. **Deterministic**: Same context + service always yields same port
5. **Fail-safe**: Corrupt registry or missing config produce helpful errors, not crashes
