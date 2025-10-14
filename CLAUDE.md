# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`dual` is a CLI tool written in Go that manages port assignments across different development contexts (git branches, worktrees, or clones). It eliminates port conflicts when working on multiple features simultaneously by automatically detecting the context and service, then injecting the appropriate PORT environment variable.

## Core Architecture

### Command Flow

The tool operates in two primary modes:

1. **Command Wrapper Mode** (primary): Intercepts arbitrary commands via `dual <command>` and injects PORT
2. **Management Mode**: Direct subcommands like `dual init`, `dual service add`, `dual context create`

The main entry point (cmd/dual/main.go:160-234) implements custom argument parsing to distinguish between wrapper mode and management mode before delegating to cobra commands.

### Key Components

**Config Layer** (`internal/config/`)
- Loads and validates `dual.config.yml` from project root (or parent directories)
- Schema version 1 with services map (name → path, envFile)
- Validates paths exist and are relative to project root
- Thread-safe file I/O with atomic writes

**Registry Layer** (`internal/registry/`)
- Global state in `~/.dual/registry.json` (per-user, cross-project)
- Structure: projects → contexts → basePort mappings
- Thread-safe with sync.RWMutex for concurrent dual instances
- Atomic writes via temp file + rename pattern
- Auto-recovers from corruption (returns empty registry)

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

## File Locations

### User Data
- Registry: `~/.dual/registry.json` (global, never committed)
- Config: `dual.config.yml` (project root, can be committed)
- Context override: `.dual-context` (worktree/clone specific)

### Code Structure
```
cmd/dual/          Command definitions (cobra commands)
internal/config/   Config file parsing and validation
internal/registry/ Global registry operations
internal/context/  Context detection logic
internal/service/  Service detection and port calculation
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
