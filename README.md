<div align="center">

# dual

**Git worktree lifecycle management with hook-based customization**

[![Go Version](https://img.shields.io/github/go-mod/go-version/lightfastai/dual)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/lightfastai/dual)](https://github.com/lightfastai/dual/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/lightfastai/dual/test.yml)](https://github.com/lightfastai/dual/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/lightfastai/dual)](https://goreportcard.com/report/github.com/lightfastai/dual)

[![GitHub stars](https://img.shields.io/github/stars/lightfastai/dual)](https://github.com/lightfastai/dual/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/lightfastai/dual)](https://github.com/lightfastai/dual/issues)

`dual` is a CLI tool that automates git worktree lifecycle management with a flexible hook system for custom environment configuration. Work on multiple features simultaneously with automated setup for databases, ports, dependencies, and more.

[Features](#key-features) • [Installation](#installation) • [Quick Start](#quick-start) • [Documentation](#documentation) • [Examples](#real-world-workflows)

</div>

---

## The Problem

When working on multiple features using git worktrees:

```bash
# Create a worktree manually
git worktree add ../myproject-feature feature-branch

# Now you need to manually:
# 1. Create a database branch for isolated testing
# 2. Assign a unique port to avoid conflicts
# 3. Copy and customize environment files
# 4. Install dependencies
# 5. Run any other project-specific setup

# And when you're done:
# 1. Remember to clean up the database branch
# 2. Remove the worktree
# 3. Clean up any other resources
```

**Common pain points:**
- Manual worktree setup is repetitive and error-prone
- Forgetting cleanup steps leads to resource leaks
- Port conflicts when running multiple dev servers
- Database branch management across worktrees
- Inconsistent environment configuration
- No automated dependency installation

## The Solution

`dual` automates the entire worktree lifecycle with customizable hooks:

```bash
# Create a worktree with automated setup
dual create feature-auth
# [dual] Creating worktree: feature-auth
# [dual] Running hook: create-database-branch.sh
# [Hook] Created database branch: feature-auth
# [dual] Running hook: setup-environment.sh
# [Hook] Assigned port: 4237
# [Hook] Environment file created: .env.local
# [dual] Running hook: install-dependencies.sh
# [Hook] Dependencies installed
# [dual] Worktree ready at: /Users/dev/worktrees/feature-auth

# Delete with automated cleanup
dual delete feature-auth
# [dual] Running hook: cleanup-database.sh
# [Hook] Deleted database branch: feature-auth
# [dual] Removing worktree: feature-auth
# [dual] Worktree deleted successfully
```

## Key Features

- **Automated Worktree Lifecycle**: Create and delete worktrees with a single command
- **Hook-Based Customization**: Implement custom logic for ports, databases, environments, and more
- **Project-Local State**: Each project has its own registry and hooks
- **Service Detection**: Auto-detect which service you're working on in monorepos
- **Context Management**: Track worktrees and their configurations
- **Transparent**: See exactly what operations are being performed
- **Fast**: Native Go binary, instant startup
- **Safe**: Fail-safe error handling prevents partial cleanup
- **Flexible**: Configure hooks for your specific workflow

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap lightfastai/tap
brew install dual
```

### Direct Download

```bash
# Download and install latest release
curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_$(uname -s)_$(uname -m).tar.gz" | \
  sudo tar -xzf - -C /usr/local/bin dual

# Verify installation
dual --version
```

### Build from Source

```bash
git clone https://github.com/lightfastai/dual.git
cd dual
go build -o dual ./cmd/dual
mv dual /usr/local/bin/
```

### Shell Completions

Enable auto-completion for commands, flags, and dynamic values:

#### Bash

```bash
# Install completions permanently
# Linux:
dual completion bash | sudo tee /etc/bash_completion.d/dual > /dev/null

# macOS (with Homebrew bash-completion):
dual completion bash > $(brew --prefix)/etc/bash_completion.d/dual
```

#### Zsh

```bash
# Install completions
dual completion zsh > "${fpath[1]}/_dual"

# Reload shell
exec zsh
```

#### Fish

```bash
# Install completions permanently
dual completion fish > ~/.config/fish/completions/dual.fish
```

## Quick Start

### 1. Initialize your project

```bash
cd ~/Code/myproject
dual init
```

This creates `dual.config.yml`:

```yaml
version: 1
services: {}
```

### 2. Register your services

```bash
# For a monorepo with multiple apps
dual service add web --path apps/web --env-file .env.local
dual service add api --path apps/api --env-file .env

# For a single-service project
dual service add app --path . --env-file .env.local

# List services to verify
dual service list
```

### 3. Configure worktrees and hooks

Add to your `dual.config.yml`:

```yaml
version: 1

services:
  web:
    path: apps/web
    envFile: .env.local
  api:
    path: apps/api
    envFile: .env

worktrees:
  path: ../worktrees        # Where to create worktrees
  naming: "{branch}"        # Directory naming pattern

hooks:
  postWorktreeCreate:
    - setup-database.sh
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - cleanup-database.sh
```

### 4. Create hook scripts

Create `.dual/hooks/setup-environment.sh`:

```bash
#!/bin/bash
set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate unique port based on context name
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

Make it executable:

```bash
chmod +x .dual/hooks/setup-environment.sh
```

### 5. Use dual to manage worktrees

```bash
# Create a worktree with automated setup
dual create feature-auth

# Work on your feature
cd ../worktrees/feature-auth
npm run dev  # Uses the auto-configured port

# Delete with automated cleanup
dual delete feature-auth
```

## Usage

### Core Commands

```bash
# Initialize project
dual init

# Service management
dual service add <name> --path <path> --env-file <file>
dual service list
dual service remove <name>

# Worktree lifecycle
dual create <branch>              # Create worktree with hooks
dual delete <context>             # Delete worktree with cleanup

# Context management
dual context list                 # List all contexts
dual context                      # Show current context

# Health check
dual doctor                       # Diagnose configuration issues
```

### Hook System

Hooks are shell scripts that run at key lifecycle points:

**Hook Events:**
- `postWorktreeCreate` - After creating a worktree
- `preWorktreeDelete` - Before deleting a worktree
- `postWorktreeDelete` - After deleting a worktree

**Hook Environment Variables:**
- `DUAL_EVENT` - Hook event name
- `DUAL_CONTEXT_NAME` - Context name (usually branch name)
- `DUAL_CONTEXT_PATH` - Absolute path to worktree
- `DUAL_PROJECT_ROOT` - Absolute path to main repository

**Hook Example** (`.dual/hooks/create-database-branch.sh`):

```bash
#!/bin/bash
set -e

echo "Creating database branch for: $DUAL_CONTEXT_NAME"

# Create PlanetScale branch
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main

echo "Database branch created: $DUAL_CONTEXT_NAME"
```

For detailed hook documentation, see the [Hook System](#hook-system-details) section below.

## Configuration

### Project Config (`dual.config.yml`)

Lives at your project root:

```yaml
version: 1

services:
  web:
    path: apps/web
    envFile: .env.local
  api:
    path: apps/api
    envFile: .env

worktrees:
  path: ../worktrees          # Relative to project root
  naming: "{branch}"          # Supports {branch} placeholder

hooks:
  postWorktreeCreate:
    - setup-database.sh       # Relative to .dual/hooks/
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - cleanup-database.sh
  postWorktreeDelete:
    - notify-team.sh
```

**Can be committed** to share configuration with your team.

### Project-Local State (`.dual/`)

Each project has its own state directory:

```
.dual/
├── registry.json              # Context-to-worktree mappings
├── registry.json.lock         # Lock file for concurrent access
└── hooks/                     # Hook scripts
    ├── setup-environment.sh
    ├── setup-database.sh
    └── cleanup-database.sh
```

**Should be added to `.gitignore`** (except hooks directory).

The registry is project-local and shared across all worktrees of the same repository:

```json
{
  "projects": {
    "/Users/dev/Code/myproject": {
      "contexts": {
        "main": {
          "created": "2025-10-14T10:00:00Z"
        },
        "feature-auth": {
          "path": "/Users/dev/worktrees/feature-auth",
          "created": "2025-10-14T11:30:00Z"
        }
      }
    }
  }
}
```

## Hook System Details

### Hook Configuration

Hooks are configured in `dual.config.yml` and stored in `.dual/hooks/`:

```yaml
hooks:
  postWorktreeCreate:
    - script1.sh
    - script2.sh
  preWorktreeDelete:
    - cleanup.sh
  postWorktreeDelete:
    - notify.sh
```

### Hook Execution Rules

- Scripts must be executable (`chmod +x`)
- Scripts run sequentially (not parallel)
- Non-zero exit code halts execution and fails the operation
- stdout/stderr are streamed to the user in real-time
- Scripts run with the worktree directory as working directory
- Hook failure during `dual create` leaves worktree in place but may be partially configured
- Hook failure during `dual delete` halts deletion - worktree remains

### Common Hook Patterns

#### Custom Port Assignment

```bash
#!/bin/bash
set -e

# Hash-based port assignment
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

echo "PORT=$PORT" > "$DUAL_CONTEXT_PATH/.env.local"
echo "Assigned port: $PORT"
```

#### Database Branch Creation

```bash
#!/bin/bash
set -e

# PlanetScale example
pscale branch create mydb "$DUAL_CONTEXT_NAME" --from main

# Get connection string
CONNECTION_URL=$(pscale connect mydb "$DUAL_CONTEXT_NAME" --format url)
echo "DATABASE_URL=$CONNECTION_URL" >> "$DUAL_CONTEXT_PATH/.env.local"
```

#### Dependency Installation

```bash
#!/bin/bash
set -e

cd "$DUAL_CONTEXT_PATH"

if [ -f "package.json" ]; then
  echo "Installing npm dependencies..."
  npm install
fi

if [ -f "go.mod" ]; then
  echo "Installing Go dependencies..."
  go mod download
fi
```

#### Database Cleanup

```bash
#!/bin/bash
set -e

# Delete PlanetScale branch
pscale branch delete mydb "$DUAL_CONTEXT_NAME" --force

echo "Deleted database branch: $DUAL_CONTEXT_NAME"
```

## How It Works

### Context Detection

Priority order:

1. **Git branch name** (primary, auto-detected)
   ```bash
   git branch --show-current  # → "feature-auth"
   ```

2. **`.dual-context` file** (fallback if not in git repo)
   ```bash
   echo "custom-context" > .dual-context
   ```

3. **Fallback to "default"**

### Service Detection

Matches current working directory against service paths:

```bash
# Config: web: { path: "apps/web" }
# CWD: /Users/dev/Code/myproject/apps/web/src/components
# Match: "web"
```

Uses longest path match for nested service structures.

## Real-World Workflows

### Monorepo with Multiple Features

```bash
# Setup once
cd ~/Code/myproject
dual init
dual service add web --path apps/web --env-file .env.local
dual service add api --path apps/api --env-file .env

# Create worktrees for different features
dual create feature-auth      # Worktree at ../worktrees/feature-auth
dual create feature-payments  # Worktree at ../worktrees/feature-payments

# Work on both simultaneously
# Terminal 1
cd ../worktrees/feature-auth/apps/web
npm run dev  # Port 4237 (auto-assigned by hook)

# Terminal 2
cd ../worktrees/feature-payments/apps/web
npm run dev  # Port 4891 (auto-assigned by hook)

# No conflicts! Both run with isolated environments.

# Clean up when done
dual delete feature-auth
dual delete feature-payments
```

### Database Branch Management

Create hooks for automated database branch lifecycle:

**.dual/hooks/setup-database.sh:**
```bash
#!/bin/bash
set -e

echo "Creating database branch: $DUAL_CONTEXT_NAME"

# Create PlanetScale branch
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main --wait

# Get connection string
CONNECTION_URL=$(pscale connect myapp "$DUAL_CONTEXT_NAME" --format url)

# Write to environment file
echo "DATABASE_URL=$CONNECTION_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "Database branch ready: $DUAL_CONTEXT_NAME"
```

**.dual/hooks/cleanup-database.sh:**
```bash
#!/bin/bash
set -e

echo "Deleting database branch: $DUAL_CONTEXT_NAME"

# Delete PlanetScale branch
pscale branch delete myapp "$DUAL_CONTEXT_NAME" --force

echo "Database branch deleted"
```

Then use:

```bash
dual create feature-db-migration
# [dual] Creating worktree: feature-db-migration
# [Hook] Creating database branch: feature-db-migration
# [Hook] Database branch ready: feature-db-migration

# ... do work ...

dual delete feature-db-migration
# [Hook] Deleting database branch: feature-db-migration
# [Hook] Database branch deleted
# [dual] Worktree deleted successfully
```

### Multi-Service Development

Run multiple services with isolated configurations:

```bash
# Start web service in main worktree
cd ~/Code/myproject/apps/web
npm run dev  # Port 4101 (from main context)

# Start web service in feature worktree
cd ~/worktrees/feature-auth/apps/web
npm run dev  # Port 4237 (from feature-auth context)

# Start API in both contexts
cd ~/Code/myproject/apps/api
npm run dev  # Port 4102 (web=4101, api=4102)

cd ~/worktrees/feature-auth/apps/api
npm run dev  # Port 4238 (web=4237, api=4238)

# All services run simultaneously without conflicts
```

## Migration from v0.2.x

If you're upgrading from v0.2.x, here are the key changes:

### What Changed

1. **Port Management Removed**: Dual no longer manages ports automatically
   - Implement port assignment in hooks instead
   - See hook examples above for custom port logic

2. **Registry Location**: Moved from `~/.dual/registry.json` to `$PROJECT_ROOT/.dual/registry.json`
   - Registry is now project-local, not global
   - Contexts must be recreated with `dual create`

3. **Command Structure**:
   - `dual context create` → `dual create` (simpler worktree creation)
   - `dual port` - Removed (implement in hooks if needed)
   - `dual ports` - Removed (implement in hooks if needed)
   - Command wrapper `dual <command>` - Removed (use hooks for env setup)

4. **Hook Events**:
   - `postPortAssign` - Removed (use `postWorktreeCreate` instead)

### Migration Steps

1. **Update your config** - Add `worktrees` section to `dual.config.yml`:
   ```yaml
   worktrees:
     path: ../worktrees
     naming: "{branch}"
   ```

2. **Implement port hooks** (if you need port management):
   ```bash
   # Create .dual/hooks/setup-environment.sh
   # See examples above for port assignment logic
   ```

3. **Update commands**:
   ```bash
   # Old
   dual context create feature-x --base-port 4200

   # New
   dual create feature-x
   # (port is assigned by hook instead)
   ```

4. **Recreate contexts**:
   ```bash
   # Old contexts in global registry won't work
   # Recreate them with dual create
   dual create main
   dual create feature-branch
   ```

5. **Add `.dual/` to `.gitignore`**:
   ```gitignore
   .dual/registry.json
   .dual/registry.json.lock
   ```

For detailed migration assistance, run `dual doctor` to diagnose configuration issues.

## Documentation

- **[USAGE.md](USAGE.md)** - Comprehensive command reference
- **[EXAMPLES.md](EXAMPLES.md)** - Real-world usage patterns
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technical architecture details
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Development guidelines
- **[CLAUDE.md](CLAUDE.md)** - Project context for AI assistants

## Comparison

### vs Manual Worktree Management

```bash
# Before dual
git worktree add ../feature-auth -b feature-auth
cd ../feature-auth
# ... manual setup: database, env files, ports, dependencies ...

# With dual
dual create feature-auth
# All setup automated via hooks
```

### vs Other Tools

| Feature | dual | git worktree | tmuxinator |
|---------|------|-------------|------------|
| Worktree creation | Automated | Manual | N/A |
| Database branch mgmt | Via hooks | Manual | Manual |
| Environment setup | Via hooks | Manual | Via config |
| Port management | Via hooks | Manual | Manual |
| Cleanup automation | Via hooks | Manual | N/A |
| Project-local state | Yes | Yes | No |

## Development

### Building

```bash
# Build binary
go build -o dual ./cmd/dual

# Install to GOPATH/bin
go install ./cmd/dual

# Build with version info
go build -ldflags="-X main.version=1.0.0" -o dual ./cmd/dual
```

### Testing

```bash
# Unit tests
go test -v -race ./internal/...

# Integration tests
go test -v -timeout=10m ./test/integration/...

# All tests with coverage
go test -cover ./...
```

### Linting

```bash
# Run golangci-lint
golangci-lint run

# Run with auto-fix
golangci-lint run --fix
```

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## Roadmap

### v0.3.0 (Current)
- Worktree lifecycle management
- Hook-based customization
- Project-local registry
- Service detection
- Context management

### Planned
- Visual dashboard (`dual ui`)
- Hook templates and examples library
- Windows support
- Integration with tmux/terminal multiplexers
- Environment variable templates
- Git hook integration (pre-commit, pre-push)

## Credits

Built by [Lightfast](https://github.com/lightfastai) to automate our multi-context development workflow. Open-sourced to help other developers working with git worktrees.

## Support

- [Report a bug](https://github.com/lightfastai/dual/issues/new?template=bug_report.md)
- [Request a feature](https://github.com/lightfastai/dual/issues/new?template=feature_request.md)
- [Join discussions](https://github.com/lightfastai/dual/discussions)
- [Read the docs](https://github.com/lightfastai/dual/wiki)

---

**Made with care by developers who love git worktrees.**
