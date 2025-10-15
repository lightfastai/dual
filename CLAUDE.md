# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dual` is a CLI tool written in Go that manages git worktree lifecycle (creation, deletion) with environment remapping via hooks. It enables developers to work on multiple features simultaneously by automating worktree setup and providing a flexible hook system for custom environment configuration (database branches, port assignment, dependency installation, etc.).

## Core Architecture

### Command Flow

The tool operates primarily in **Management Mode** with direct subcommands:
- `dual init` - Initialize dual configuration
- `dual service add/list/remove` - Manage service definitions
- `dual list` - List all contexts/worktrees
- `dual create <branch>` - Create a new worktree with lifecycle hooks
- `dual delete <context>` - Delete a worktree with cleanup hooks
- `dual doctor` - Diagnose configuration and registry health

The main entry point (cmd/dual/main.go) uses cobra for command routing and execution.

### Key Components

**Config Layer** (`internal/config/`)
- Loads and validates `dual.config.yml` from project root (or parent directories)
- Schema version 1 with services map (name → path, envFile)
- Supports optional worktrees configuration (path, naming pattern)
- Supports optional hooks configuration (lifecycle event → script list)
- Validates paths exist and are relative to project root
- Thread-safe file I/O with atomic writes

**Registry Layer** (`internal/registry/`)
- Project-local state in `$PROJECT_ROOT/.dual/.local/registry.json` (per-project, not committed)
- Structure: projects → contexts (name → path, created timestamp)
- All worktrees of a repository share the parent repo's registry via `GetProjectIdentifier()` normalization
- Thread-safe with sync.RWMutex for concurrent dual instances
- File locking with flock to prevent concurrent modifications
- Atomic writes via temp file + rename pattern
- Auto-recovers from corruption (returns empty registry)

**Hook System** (`internal/hooks/`)
- Executes lifecycle hooks at key worktree management points
- Hook events: `postWorktreeCreate`, `preWorktreeDelete`, `postWorktreeDelete`
- Scripts located in `$PROJECT_ROOT/.dual/hooks/` and configured in `dual.config.yml`
- Receives context via environment variables: `DUAL_EVENT`, `DUAL_CONTEXT_NAME`, `DUAL_CONTEXT_PATH`, `DUAL_PROJECT_ROOT`
- Non-zero exit codes fail the operation and halt execution

**Context Detection** (`internal/context/`)
- Priority: git branch → `.dual-context` file → "default"
- Simple, no external dependencies beyond git command

**Service Detection** (`internal/service/detector.go`)
- Matches current working directory against service paths from config
- Resolves symlinks for consistency across worktrees
- Uses longest path match for nested service structures
- Returns `ErrServiceNotDetected` if no match found

### Worktree Management Commands

**`dual create <branch>`** - Creates a new git worktree with integrated context setup:
1. Creates git worktree at configured location (from `worktrees.path` in config)
2. Names directory using configured pattern (from `worktrees.naming`, defaults to `{branch}`)
3. Registers context in project-local registry
4. Executes `postWorktreeCreate` hooks

**`dual delete <context>`** - Deletes a worktree with cleanup:
1. Validates context exists and is a worktree
2. Executes `preWorktreeDelete` hooks
3. Removes git worktree
4. Removes context from registry
5. Executes `postWorktreeDelete` hooks

Both commands integrate with the hook system to enable custom automation (database setup, environment configuration, port assignment, cleanup tasks, etc.).

## Hook System

The dual CLI supports lifecycle hooks that run at key points during worktree management. Hooks enable custom automation like database branch creation, environment setup, port assignment, and cleanup tasks.

### Hook Events

- **`postWorktreeCreate`**: Runs after creating a git worktree and registering the context
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
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - backup-data.sh
    - cleanup-database.sh
  postWorktreeDelete:
    - notify-team.sh
```

### Hook Environment Variables

Hook scripts receive the following environment variables:

- **`DUAL_EVENT`**: The hook event name (e.g., `postWorktreeCreate`)
- **`DUAL_CONTEXT_NAME`**: Context name (usually the branch name)
- **`DUAL_CONTEXT_PATH`**: Absolute path to the worktree directory
- **`DUAL_PROJECT_ROOT`**: Absolute path to the main repository

### Hook Script Examples

**Example 1: Custom Port Assignment**
```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh

set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate port based on context name hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Write to .env file
cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
PORT=$PORT
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
EOF

echo "Assigned port: $PORT"
```

**Example 2: Database Branch Setup**
```bash
#!/bin/bash
# .dual/hooks/setup-database.sh

set -e

echo "Creating database branch for: $DUAL_CONTEXT_NAME"

# Create PlanetScale branch
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main

# Get connection string
CONNECTION_URL=$(pscale connect myapp "$DUAL_CONTEXT_NAME" --format url)

# Update .env file
echo "DATABASE_URL=$CONNECTION_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "Database branch created"
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
go test -v -run TestCreateWorktree ./test/integration/...

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
- Registry operations use file locking (flock) to prevent concurrent modifications
- Config saves use atomic write pattern: write temp file → rename
- Multiple dual instances can run concurrently safely

### Dependency Injection in Tests
Detectors accept function parameters for git commands, getwd, evalSymlinks to enable mocking in tests. See internal/service/detector.go.

### Security Notes
- Commands executed via exec.Command are intentional (gosec G204 suppressed)
- File reads from config paths are trusted (gosec G304 suppressed)
- Test files excluded from most linters (see .golangci.yml)

### Registry Location and Sharing
The registry is **project-local** at `$PROJECT_ROOT/.dual/.local/registry.json`, not global. This means:
- Each project has its own registry file
- All worktrees of a repository share the parent repo's registry (normalized via `GetProjectIdentifier()`)
- The registry should be added to `.gitignore` to avoid committing context mappings
- File locking ensures concurrent dual operations don't corrupt the registry

## Configuration Schema

The `dual.config.yml` file supports the following structure:

```yaml
version: 1

services:
  # Required: Map of service name to configuration
  web:
    path: ./apps/web           # Required: relative path from project root
    envFile: .env.local        # Optional: env file path (for reference)
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
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - backup-data.sh
    - cleanup-database.sh
  postWorktreeDelete:
    - notify-team.sh
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
- Registry: `$PROJECT_ROOT/.dual/.local/registry.json` (project-local, should not be committed - add to .gitignore)
- Registry lock: `$PROJECT_ROOT/.dual/.local/registry.json.lock` (temporary file during operations)
- Config: `dual.config.yml` (project root, can be committed)
- Hooks: `$PROJECT_ROOT/.dual/hooks/` (project-local, can be committed)
- Context override: `.dual-context` (worktree/clone specific)

### Code Structure
```
cmd/dual/          Command definitions (cobra commands)
internal/config/   Config file parsing and validation
internal/registry/ Project-local registry operations with file locking
internal/context/  Context detection logic
internal/service/  Service detection
internal/hooks/    Hook execution and lifecycle management
internal/worktree/ Worktree operations (creation, deletion)
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

1. **Hook-Based Customization**: Core tool manages worktree lifecycle; users implement custom logic (ports, databases, env) in hooks
2. **Transparent**: Always show user what operations are being performed
3. **Fail-safe**: Corrupt registry or missing config produce helpful errors, not crashes
4. **Isolated Contexts**: Each worktree is an independent environment
5. **Project-Local State**: Registry and hooks are project-specific, not global

## Migration Notes (v0.2.2 → v0.3.0)

**Port Management Removed**: Dual no longer manages ports automatically. If you need port assignment:
1. Implement custom port logic in a `postWorktreeCreate` hook
2. Calculate ports based on context name, hash, sequential assignment, etc.
3. Write PORT to `.env` file or use your preferred method

**Registry Migration**: The registry has moved from `~/.dual/registry.json` to `$PROJECT_ROOT/.dual/.local/registry.json`:
- Old global registry is no longer used
- Contexts must be recreated with `dual create`
- Each project now has its own isolated registry

**Removed Commands**:
- `dual port` - Port querying (no longer applicable)
- `dual ports` - List all ports (no longer applicable)
- Command wrapper mode `dual <command>` - PORT injection (no longer applicable)

**Removed Hook Event**:
- `postPortAssign` - Port assignment hooks (implement port logic in `postWorktreeCreate` instead)

See `.dual/hooks/README.md` for examples of implementing custom port assignment in hooks.
