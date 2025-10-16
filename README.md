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

## What's New in v0.3.0

### Full Dotenv Compatibility

Dual now uses the industry-standard godotenv library for complete compatibility with Node.js dotenv files:

- **Multiline Values**: Use quotes for certificates, keys, SQL queries, and formatted text
- **Variable Expansion**: Reference other variables with `${VAR}` or `$VAR` syntax for DRY configuration
- **Escape Sequences**: Process `\n`, `\t`, `\\`, `\"` in double-quoted strings
- **Inline Comments**: Document your configuration with `#` comments
- **Complex Quoting**: Mix single and double quotes for nested values

```bash
# .env - Full dotenv support
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=myapp

# Variable expansion - build URLs from components
DATABASE_URL=postgresql://${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_NAME}

# Multiline values - perfect for certificates
TLS_CERT="-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUA...
-----END CERTIFICATE-----"

# Escape sequences - actual newlines in values
WELCOME_MESSAGE="Hello!\nWelcome to our app"
```

### Unified Environment Loading

All environment-related commands now use the same three-layer loading system:

1. **Base Environment** (.env.base) - Shared across all services
2. **Service Environment** (apps/api/.env) - Service-specific defaults
3. **Context Overrides** (.dual/.local/service/api/.env) - Worktree-specific values

This ensures `dual env show`, `dual env export`, and `dual run` all see exactly the same environment.

### Enhanced Error Handling

Better error messages with actionable guidance:

```
Error: worktrees.path not configured in dual.config.yml
Hint: Add a worktrees section to use 'dual create':
  worktrees:
    path: ../worktrees
    naming: "{branch}"
```

### Migration Notes

If upgrading from v0.2.x, some .env files may need updates:
- Variable expansion is now enabled (quote with single quotes for literal `${VAR}`)
- Escape sequences are processed in double quotes (use single quotes for literal `\n`)
- Inline comments are stripped (quote values containing `#`)

See [USAGE.md - Migration Guide](USAGE.md#migration-guide) for detailed migration instructions.

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
- **Full Dotenv Compatibility**: Multiline values, variable expansion, escape sequences - works with any Node.js dotenv file
- **Three-Layer Environment System**: Base → Service → Context with automatic merging and override support
- **Unified Environment Loading**: Consistent behavior across all commands with layered configuration
- **Enhanced Error Handling**: Actionable error messages with helpful suggestions for fixing issues
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

### 4. Create base environment (optional)

Create `.env.base` for shared configuration:

```bash
# .env.base - Shared across all services and contexts
APP_NAME=MyApp
API_VERSION=v1
LOG_LEVEL=info
NODE_ENV=development

# Database configuration (will be overridden per context)
DATABASE_HOST=localhost
DATABASE_PORT=5432
```

Configure in `dual.config.yml`:

```yaml
env:
  baseFile: .env.base
```

### 5. Create service environments

Create service-specific `.env` files using modern dotenv features:

```bash
# apps/web/.env
PORT=3000

# Use variable expansion for DRY configuration
API_HOST=localhost
API_PORT=3001
API_URL=http://${API_HOST}:${API_PORT}

# Multi-service URLs
PUBLIC_URL=http://localhost:${PORT}
```

```bash
# apps/api/.env
PORT=3001

# Variable expansion from base environment
DATABASE_URL=postgresql://${DATABASE_HOST}:${DATABASE_PORT}/myapp

# Multiline configuration
CORS_ORIGINS="http://localhost:3000
http://localhost:3001"
```

### 6. Create hook scripts

Create `.dual/hooks/setup-environment.sh` to customize per worktree:

```bash
#!/bin/bash
set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate unique port based on context name
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Use dual env to set context-specific overrides
cd "$DUAL_CONTEXT_PATH"
dual env set PORT $PORT
dual env set DATABASE_URL "postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}"

echo "Assigned port: $PORT"
```

Make it executable:

```bash
chmod +x .dual/hooks/setup-environment.sh
```

### 7. Use dual to manage worktrees

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

# Environment management
dual env show                     # Display environment summary
dual env set KEY value            # Set context-specific override
dual env unset KEY                # Remove override
dual env export                   # Export merged environment
dual env check                    # Validate configuration
dual env diff ctx1 ctx2           # Compare environments

# Command execution
dual run <command>                # Run with full environment injection

# Health check
dual doctor                       # Diagnose configuration issues
```

### Environment Management

Dual provides a three-layer environment system that automatically merges configuration:

#### Layer 1: Base Environment (Shared)

Create `.env.base` for variables shared across all services:

```bash
# .env.base
APP_NAME=MyApp
API_VERSION=v1
LOG_LEVEL=info
DATABASE_HOST=localhost
DATABASE_PORT=5432
```

Configure in `dual.config.yml`:

```yaml
env:
  baseFile: .env.base
```

#### Layer 2: Service Environment (Defaults)

Each service has its own `.env` file with service-specific defaults:

```bash
# apps/api/.env
PORT=3001

# Use variable expansion from base environment
DATABASE_URL=postgresql://${DATABASE_HOST}:${DATABASE_PORT}/myapp

# Service-specific configuration
MAX_CONNECTIONS=100
REDIS_URL=redis://localhost:6379
```

```bash
# apps/web/.env
PORT=3000

# Build URLs using variable expansion
API_HOST=localhost
API_PORT=3001
API_URL=http://${API_HOST}:${API_PORT}

PUBLIC_URL=http://localhost:${PORT}
```

#### Layer 3: Context Overrides (Per-Worktree)

Set context-specific values that override the lower layers:

```bash
# Set global override for current context
dual env set DATABASE_URL "postgresql://localhost/myapp_feature_auth"

# Set service-specific override
dual env set --service api PORT 5000
dual env set --service web PORT 4000

# View current environment
dual env show --values

# Export for use in other tools
dual env export > .env.local
```

#### Environment Features

**Variable Expansion** - Build complex values from simple parts:

```bash
# .env
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=myapp

# Expansion happens automatically
DATABASE_URL=postgresql://${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_NAME}
```

**Multiline Values** - Perfect for certificates and formatted text:

```bash
# .env
TLS_CERT="-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUA...
-----END CERTIFICATE-----"

SQL_QUERY="SELECT users.id, users.name
FROM users
WHERE users.active = true"
```

**Escape Sequences** - Process special characters in double quotes:

```bash
# .env
WELCOME_MESSAGE="Hello!\nWelcome to our application"  # Actual newline
WINDOWS_PATH="C:\\Program Files\\MyApp"                # Escaped backslashes
```

**Inline Comments** - Document your configuration:

```bash
# .env
PORT=3000           # Application port
DEBUG=true          # Enable debug mode
MAX_WORKERS=4       # CPU core count
```

#### Running Commands with Environment

Use `dual run` to execute commands with the full merged environment:

```bash
# Service auto-detected from current directory
cd apps/api
dual run npm start

# Or explicitly specify service
dual run --service api npm start

# Run with full environment injection
dual run node server.js
# Server receives merged variables from all three layers
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

# Optional: Base environment file
env:
  baseFile: .env.base

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

**New in v0.3.0**: The `env.baseFile` option enables a base environment layer shared across all services and contexts.

### Project-Local State (`.dual/`)

Each project has its own state directory:

```
.dual/
├── .local/
│   ├── registry.json          # Context-to-worktree mappings
│   └── registry.json.lock     # Lock file for concurrent access
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

# Use dual env to set context-specific overrides
cd "$DUAL_CONTEXT_PATH"
dual env set PORT $PORT

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

# Use dual env to set database URL
cd "$DUAL_CONTEXT_PATH"
dual env set DATABASE_URL "$CONNECTION_URL"

echo "Database branch created: $DUAL_CONTEXT_NAME"
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

### New Features in v0.3.0

1. **Full Dotenv Compatibility**:
   - Multiline values, variable expansion, escape sequences
   - Use modern Node.js dotenv syntax in all .env files
   - **Breaking**: Variable expansion now enabled by default

2. **Three-Layer Environment System**:
   - Base environment (.env.base)
   - Service environments (apps/api/.env)
   - Context overrides (managed via `dual env set`)
   - **New**: `dual env` command suite for environment management

3. **Unified Environment Loading**:
   - All commands now use consistent environment loading
   - Fixed bugs where service .env files weren't loaded
   - **New**: `dual run` command for executing with full environment

4. **Enhanced Error Handling**:
   - Actionable error messages with hints
   - Better diagnostics for configuration issues
   - Helpful suggestions for fixing problems

### Breaking Changes

#### 1. Dotenv Parsing Changes

**Variable Expansion** (now enabled):
```bash
# Old behavior: literal text
API_URL=${BASE_URL}/api  # Result: "${BASE_URL}/api"

# New behavior: expanded
API_URL=${BASE_URL}/api  # Result: "http://localhost:3000/api"

# To keep literal, use single quotes:
TEMPLATE='${BASE_URL}/api'
```

**Escape Sequences** (now processed in double quotes):
```bash
# Old behavior: literal
MESSAGE="Hello\nWorld"  # Result: "Hello\nWorld"

# New behavior: processed
MESSAGE="Hello\nWorld"  # Result: "Hello
                        #          World"

# To keep literal, use single quotes:
LITERAL='Hello\nWorld'
```

**Inline Comments** (now stripped):
```bash
# Old behavior: included in value
COLOR=#FF00FF  # Result: "#FF00FF"

# New behavior: stripped
COLOR=#FF00FF  # Result: "" (comment stripped!)

# Quote values with #:
COLOR="#FF00FF"
```

#### 2. Architecture Changes

- **Port Management Removed**: Implement in hooks instead
- **Registry Location**: Moved to `$PROJECT_ROOT/.dual/.local/registry.json`
- **Command Changes**: Simplified to focus on worktree lifecycle

### Migration Steps

#### Step 1: Update .env Files

Check your .env files for these patterns:

```bash
# Find potential issues
grep '\${' .env* apps/*/.env      # Variable references
grep '\\' .env* apps/*/.env       # Escape sequences
grep '=' .env* apps/*/.env | grep '#'  # Hash symbols
```

**Fix variable references:**
```bash
# If you want expansion (recommended):
DATABASE_URL=postgresql://${DATABASE_HOST}:${DATABASE_PORT}/myapp

# If you want literal:
TEMPLATE_URL='${BASE_URL}/api'
```

**Fix escape sequences:**
```bash
# Single quotes for Windows paths:
PATH='C:\Program Files\App'

# Or escape in double quotes:
PATH="C:\\Program Files\\App"
```

**Fix hash symbols:**
```bash
# Quote values containing #:
COLOR="#FF00FF"
CHANNEL="#general"
```

#### Step 2: Update Configuration

Add environment configuration to `dual.config.yml`:

```yaml
version: 1

# Optional: Base environment
env:
  baseFile: .env.base

# Required for dual create/delete
worktrees:
  path: ../worktrees
  naming: "{branch}"

# Update hooks
hooks:
  postWorktreeCreate:
    - setup-environment.sh
```

#### Step 3: Update Hook Scripts

Modernize hooks to use `dual env`:

```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh
set -e

# Calculate unique port
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Use dual env instead of writing files directly
cd "$DUAL_CONTEXT_PATH"
dual env set PORT $PORT
dual env set DATABASE_URL "postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}"
```

#### Step 4: Test Environment Loading

```bash
# Check loaded environment
dual env show --values

# Verify variable expansion works
dual env export | grep DATABASE_URL

# Test with your application
dual run npm start
```

#### Step 5: Clean Up

```bash
# Add new registry location to .gitignore
echo "/.dual/.local/" >> .gitignore

# Remove old files if migrating
rm -f ~/.dual/registry.json  # Old global registry (no longer used)
```

### Getting Help

- **Verify setup**: `dual doctor`
- **Check environment**: `dual env show --values`
- **Debug issues**: `dual --debug env check`
- **Detailed docs**: See [USAGE.md - Migration Guide](USAGE.md#migration-guide)

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
- ✅ Worktree lifecycle management
- ✅ Hook-based customization
- ✅ Full dotenv compatibility (multiline, expansion, escape sequences)
- ✅ Three-layer environment system
- ✅ Unified environment loading
- ✅ Enhanced error handling with actionable hints
- ✅ Project-local registry
- ✅ Service detection
- ✅ Context management
- ✅ `dual env` command suite
- ✅ `dual run` command execution

### v0.4.0 (Planned)
- Environment variable validation and schema
- Template-based environment generation
- Hook templates and examples library
- Improved `dual doctor` diagnostics
- Performance optimizations

### Future
- Visual dashboard (`dual ui`)
- Windows support
- Integration with tmux/terminal multiplexers
- Git hook integration (pre-commit, pre-push)
- Cloud service integrations (PlanetScale, Supabase, etc.)

## Credits

Built by [Lightfast](https://github.com/lightfastai) to automate our multi-context development workflow. Open-sourced to help other developers working with git worktrees.

## Support

- [Report a bug](https://github.com/lightfastai/dual/issues/new?template=bug_report.md)
- [Request a feature](https://github.com/lightfastai/dual/issues/new?template=feature_request.md)
- [Join discussions](https://github.com/lightfastai/dual/discussions)
- [Read the docs](https://github.com/lightfastai/dual/wiki)

---

**Made with care by developers who love git worktrees.**
