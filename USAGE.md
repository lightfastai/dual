# dual Usage Guide

Complete reference for all `dual` commands and their usage in v0.3.0.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Initialization Commands](#initialization-commands)
  - [dual init](#dual-init)
- [Service Management](#service-management)
  - [dual service add](#dual-service-add)
  - [dual service list](#dual-service-list)
  - [dual service remove](#dual-service-remove)
- [Worktree Management](#worktree-management)
  - [dual create](#dual-create)
  - [dual delete](#dual-delete)
- [Context Management](#context-management)
  - [dual context list](#dual-context-list)
- [Hook System](#hook-system)
  - [Lifecycle Events](#lifecycle-events)
  - [Hook Configuration](#hook-configuration)
  - [Hook Environment Variables](#hook-environment-variables)
  - [Hook Examples](#hook-examples)
- [Environment Management](#environment-management)
  - [dual env show](#dual-env-show)
  - [dual env set](#dual-env-set)
  - [dual env unset](#dual-env-unset)
  - [dual env export](#dual-env-export)
  - [dual env check](#dual-env-check)
  - [dual env diff](#dual-env-diff)
  - [dual env remap](#dual-env-remap)
- [Command Execution](#command-execution)
  - [dual run](#dual-run)
- [Utility Commands](#utility-commands)
  - [dual doctor](#dual-doctor)
- [Dotenv Compatibility Features](#dotenv-compatibility-features)
  - [Multiline Values](#multiline-values)
  - [Variable Expansion](#variable-expansion)
  - [Escape Sequences](#escape-sequences)
  - [Inline Comments](#inline-comments)
  - [Complex Quoting](#complex-quoting)
- [Environment Loading System](#environment-loading-system)
  - [Three-Layer Priority System](#three-layer-priority-system)
  - [Service Detection](#service-detection)
  - [Consistent Behavior](#consistent-behavior)
- [Configuration Reference](#configuration-reference)
- [Project-Local Registry](#project-local-registry)
- [Debug & Verbose Options](#debug--verbose-options)
- [Migration Guide](#migration-guide)
  - [Upgrading from Previous Versions](#upgrading-from-previous-versions)
  - [Breaking Changes](#breaking-changes)
  - [Migration Examples](#migration-examples)
- [Common Workflows](#common-workflows)

---

## Overview

`dual` is a CLI tool for managing git worktree lifecycle with environment remapping via hooks. It enables developers to work on multiple features simultaneously by automating worktree setup and providing a flexible hook system for custom environment configuration.

### Key Features

- **Worktree Lifecycle Management**: Create and delete git worktrees with integrated hooks
- **Hook-Based Customization**: Implement custom logic (ports, databases, dependencies) in hook scripts
- **Project-Local State**: Each project has its own isolated registry and hooks
- **Transparent Operations**: Always see what operations are being performed
- **Fail-Safe**: Helpful errors instead of crashes

### Architecture

In v0.3.0, `dual` operates in **Management Mode** with direct subcommands:
- `dual create <branch>` - Create a new worktree with lifecycle hooks
- `dual delete <context>` - Delete a worktree with cleanup hooks
- `dual init` - Initialize dual configuration
- `dual service add/list/remove` - Manage service definitions
- `dual context list` - List contexts
- `dual doctor` - Diagnose configuration and registry health

### What's New in v0.3.0

**Hook-Based Architecture**: Dual no longer manages ports automatically. Instead, implement custom logic in lifecycle hooks:
- Port assignment
- Database branch creation/deletion
- Environment configuration
- Dependency installation
- Team notifications
- Any custom automation

**Project-Local Registry**: Registry moved from `~/.dual/registry.json` to `$PROJECT_ROOT/.dual/.local/registry.json` for per-project isolation.

**Full Dotenv Compatibility**: Complete support for Node.js dotenv features:
- Multiline values for certificates, keys, and formatted text
- Variable expansion with `${VAR}` and `$VAR` syntax
- Escape sequences in double quotes (`\n`, `\t`, `\\`, `\"`)
- Inline comments support
- Complex quoting behaviors

**Unified Environment Loading**: Consistent three-layer environment system:
- Base environment layer for shared configuration
- Service-specific environment from `<service-path>/.env`
- Context-specific overrides for worktree isolation
- Unified `LoadLayeredEnv()` ensures consistent behavior across all commands

**Environment Management Commands**: New `dual env` command suite:
- `dual env show` - Display environment summary and layers
- `dual env set/unset` - Manage context-specific overrides
- `dual env export` - Export merged environment in multiple formats
- `dual env check` - Validate environment configuration
- `dual env diff` - Compare environments between contexts
- `dual env remap` - Regenerate service env files

**Command Execution**: New `dual run` command:
- Run commands with full environment injection
- Automatic service detection
- Support for base + service + override layers

**Enhanced Error Messages**: Better error handling with actionable hints and suggestions.

**Simplified Commands**: Focused on worktree lifecycle management.

---

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap lightfastai/tap
brew install dual
```

### Manual Installation

Download the latest release from [GitHub Releases](https://github.com/lightfastai/dual/releases) and add to your PATH.

### Verify Installation

```bash
dual --version
```

---

## Initialization Commands

### dual init

Initialize a new `dual` configuration in the current directory.

#### Syntax

```bash
dual init [--force]
```

#### Options

- `--force` - Overwrite existing configuration file if it exists

#### Examples

```bash
# Initialize in current directory
cd ~/Code/myproject
dual init

# Force overwrite existing config
dual init --force
```

#### What It Creates

Creates a `dual.config.yml` file:

```yaml
version: 1
services: {}
```

#### Output

```
[dual] Initialized configuration at /Users/dev/Code/myproject/dual.config.yml

Next steps:
  1. Add services with: dual service add <name> --path <path>
  2. Configure worktrees section for dual create/delete
  3. Add hook scripts to .dual/hooks/ for automation
```

---

## Service Management

Services define the different components of your project (web, api, worker, etc.). Service definitions are used for context and path validation.

### dual service add

Add a new service to your `dual` configuration.

#### Syntax

```bash
dual service add <name> --path <path> [--env-file <file>]
```

#### Arguments

- `<name>` - Service name (used in commands)

#### Options

- `--path <path>` - **Required.** Relative path from project root to service directory
- `--env-file <file>` - Optional. Relative path to env file (for reference)

#### Examples

##### Single Service Project

```bash
# Add single service at project root
dual service add app --path . --env-file .env.local
```

##### Monorepo with Multiple Services

```bash
# Add web frontend
dual service add web --path apps/web --env-file .env.local

# Add API backend
dual service add api --path apps/api --env-file .env

# Add worker service
dual service add worker --path apps/worker --env-file .env.local
```

##### Without Env File

```bash
# Add service without env file reference
dual service add docs --path docs
```

#### Output

```
[dual] Added service "web"
  Path: apps/web
  Env File: .env.local
```

#### Notes

- Paths must be relative to project root (where `dual.config.yml` is located)
- Paths must exist before adding the service
- Service names must be unique

---

### dual service list

List all services in your configuration.

#### Syntax

```bash
dual service list [--json] [--paths]
```

#### Options

- `--json` - Output in JSON format for machine-readable processing
- `--paths` - Show absolute paths instead of relative paths

#### Examples

##### Basic List

```bash
dual service list
```

Output:
```
Services in dual.config.yml:
  api     apps/api      .env
  web     apps/web      .env.local
  worker  apps/worker   .env.local

Total: 3 services
```

##### With Absolute Paths

```bash
dual service list --paths
```

Output:
```
Services in dual.config.yml:
  api     /Users/dev/Code/myproject/apps/api      .env
  web     /Users/dev/Code/myproject/apps/web      .env.local
  worker  /Users/dev/Code/myproject/apps/worker   .env.local

Total: 3 services
```

##### JSON Format

```bash
dual service list --json
```

Output:
```json
{
  "services": [
    {
      "name": "api",
      "path": "apps/api",
      "envFile": ".env"
    },
    {
      "name": "web",
      "path": "apps/web",
      "envFile": ".env.local"
    },
    {
      "name": "worker",
      "path": "apps/worker",
      "envFile": ".env.local"
    }
  ]
}
```

#### Use Cases

- **Quick Reference**: See all configured services at a glance
- **CI/CD Integration**: Use JSON output for automated scripts
- **Documentation**: Generate service documentation from configuration

---

### dual service remove

Remove a service from the configuration.

#### Syntax

```bash
dual service remove <name> [--force]
```

#### Arguments

- `<name>` - Name of the service to remove

#### Options

- `--force` or `-f` - Skip confirmation prompt

#### Examples

##### Interactive Removal

```bash
dual service remove worker
```

Output:
```
Remove service "worker" from configuration?
This will only remove the service definition, not any files.
Continue? (y/N): y
[dual] Service "worker" removed from config
```

##### Force Removal (No Confirmation)

```bash
dual service remove worker --force
```

Output:
```
[dual] Service "worker" removed from config
```

#### Behavior

**File Safety**: This command only removes the service from `dual.config.yml`. It does NOT delete any files or directories.

---

## Worktree Management

Worktree management is the core feature of `dual`. Create isolated development environments for each feature branch with automated setup and cleanup via hooks.

### dual create

Create a new git worktree with integrated context setup and lifecycle hooks.

#### Syntax

```bash
dual create <branch> [--from <base-branch>]
```

#### Arguments

- `<branch>` - Branch name for the new worktree

#### Options

- `--from <base-branch>` - Create branch from specified base branch (default: current branch)

#### Requirements

- Must be run from the main repository (not from within a worktree)
- `worktrees.path` must be configured in `dual.config.yml`
- Repository must not already have a branch with that name

#### Examples

##### Basic Worktree Creation

```bash
# Create worktree for feature branch
dual create feature-auth
```

Output:
```
[dual] Creating worktree for: feature-auth
[dual] Worktree path: /Users/dev/Code/myproject-wt/feature-auth
[dual] Creating git worktree...
[dual] Registering context in registry...
[dual] Executing postWorktreeCreate hooks...

Running hook: setup-environment.sh
Setting up environment for: feature-auth
Assigned port: 4237
✓ Created .env.local

Running hook: install-dependencies.sh
Installing dependencies for: feature-auth
✓ Dependencies installed

[dual] Successfully created worktree: feature-auth
  Path: /Users/dev/Code/myproject-wt/feature-auth
  Branch: feature-auth
```

##### Create from Specific Base Branch

```bash
# Create feature branch from develop instead of main
dual create feature-new-api --from develop
```

##### With Custom Naming Pattern

If your `dual.config.yml` has:
```yaml
worktrees:
  path: ../worktrees
  naming: "wt-{branch}"
```

Then:
```bash
dual create feature-x
# Creates: ../worktrees/wt-feature-x
```

#### What Happens

1. **Validation**: Checks if worktrees configuration exists, validates branch name
2. **Git Worktree Creation**: Runs `git worktree add <path> -b <branch>`
3. **Registry Update**: Adds context to project-local registry
4. **Hook Execution**: Runs `postWorktreeCreate` hooks sequentially

#### Configuration Required

Add to `dual.config.yml`:

```yaml
worktrees:
  path: ../worktrees           # Where to create worktrees
  naming: "{branch}"           # Directory naming pattern

hooks:
  postWorktreeCreate:
    - setup-environment.sh
    - install-dependencies.sh
```

#### Error Cases

```
Error: worktrees.path not configured in dual.config.yml
Hint: Add worktrees configuration to use 'dual create'
```

Solution: Add worktrees section to config.

```
Error: must be in main repository to create worktree
Hint: Run this command from the main repository, not from within a worktree
```

Solution: Change to main repository directory.

---

### dual delete

Delete a worktree with cleanup hooks.

#### Syntax

```bash
dual delete <context> [--force]
```

#### Arguments

- `<context>` - Context name (usually branch name) to delete

#### Options

- `--force` or `-f` - Skip confirmation prompt

#### Requirements

- Context must exist in registry
- Context must be a worktree (not main repository)
- Worktree directory must exist

#### Examples

##### Interactive Deletion

```bash
dual delete feature-old
```

Output:
```
About to delete worktree: feature-old
  Path: /Users/dev/Code/myproject-wt/feature-old
  Created: 2025-10-10T14:30:00Z

Are you sure you want to delete this worktree? (y/N): y

[dual] Executing preWorktreeDelete hooks...

Running hook: backup-data.sh
Backing up data for: feature-old
✓ Data backed up to ~/backups/feature-old.sql

Running hook: cleanup-database.sh
Cleaning up database for: feature-old
✓ Database branch deleted

[dual] Removing git worktree...
[dual] Removing from registry...
[dual] Executing postWorktreeDelete hooks...

Running hook: notify-team.sh
✓ Team notified

[dual] Successfully deleted worktree: feature-old
```

##### Force Deletion (No Confirmation)

```bash
dual delete feature-old --force
```

#### What Happens

1. **Validation**: Checks if context exists and is a worktree
2. **Pre-Delete Hooks**: Runs `preWorktreeDelete` hooks (files still exist)
3. **Git Worktree Removal**: Runs `git worktree remove <path>`
4. **Registry Update**: Removes context from registry
5. **Post-Delete Hooks**: Runs `postWorktreeDelete` hooks (files are gone)

#### Hook Timing

- **preWorktreeDelete**: Files still exist - good for backups, database cleanup
- **postWorktreeDelete**: Files are gone - good for notifications, external cleanup

#### Error Cases

```
Error: context "feature-x" not found in registry
Hint: Run 'dual context list' to see available contexts
```

Solution: Check context name with `dual context list`.

```
Error: context "main" is not a worktree (has no path)
Hint: Use 'dual delete' only for worktrees, not the main repository
```

Solution: Cannot delete main repository context.

---

## Context Management

### dual context list

List all contexts for the current project.

#### Syntax

```bash
dual context list [--json]
```

#### Options

- `--json` - Output in JSON format for machine-readable processing

#### Examples

##### Basic List

```bash
dual context list
```

Output:
```
Contexts for /Users/dev/Code/myproject:
NAME          PATH                                              CREATED
main          (main repository)                                 2025-10-01T10:00:00Z
feature-auth  /Users/dev/Code/myproject-wt/feature-auth        2025-10-10T14:30:00Z
feature-api   /Users/dev/Code/myproject-wt/feature-api         2025-10-12T09:15:00Z

Total: 3 contexts (1 main, 2 worktrees)
```

##### JSON Format

```bash
dual context list --json
```

Output:
```json
{
  "projectRoot": "/Users/dev/Code/myproject",
  "contexts": [
    {
      "name": "main",
      "path": "",
      "created": "2025-10-01T10:00:00Z"
    },
    {
      "name": "feature-auth",
      "path": "/Users/dev/Code/myproject-wt/feature-auth",
      "created": "2025-10-10T14:30:00Z"
    },
    {
      "name": "feature-api",
      "path": "/Users/dev/Code/myproject-wt/feature-api",
      "created": "2025-10-12T09:15:00Z"
    }
  ]
}
```

#### Use Cases

- **Context Overview**: See all contexts at a glance
- **CI/CD Integration**: Use JSON output for automated scripts
- **Worktree Management**: Track all active worktrees

---

## Hook System

The hook system is the core of `dual`'s automation capabilities. Hooks run at key points during worktree lifecycle to enable custom logic like port assignment, database setup, dependency installation, and cleanup.

### Lifecycle Events

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

**Script location**: All hook scripts must be in `$PROJECT_ROOT/.dual/hooks/`

**Script requirements**:
- Must be executable (`chmod +x .dual/hooks/script.sh`)
- Must use shebang line (`#!/bin/bash`)
- Should exit with non-zero code on failure

### Hook Environment Variables

Hook scripts receive the following environment variables:

- **`DUAL_EVENT`**: The hook event name (e.g., `postWorktreeCreate`)
- **`DUAL_CONTEXT_NAME`**: Context name (usually the branch name)
- **`DUAL_CONTEXT_PATH`**: Absolute path to the worktree directory
- **`DUAL_PROJECT_ROOT`**: Absolute path to the main repository

**Example usage in hook script**:
```bash
#!/bin/bash
echo "Event: $DUAL_EVENT"
echo "Context: $DUAL_CONTEXT_NAME"
echo "Worktree: $DUAL_CONTEXT_PATH"
echo "Project: $DUAL_PROJECT_ROOT"
```

### Hook Execution Rules

- Scripts must be executable (`chmod +x`)
- Scripts run in sequence (not parallel)
- Non-zero exit code halts execution and fails the operation
- stdout/stderr are streamed to the user in real-time
- Scripts run with the worktree directory as working directory (except `postWorktreeDelete`)
- Hook failure during `dual create` leaves the worktree in place but may be partially configured
- Hook failure during `dual delete` halts deletion - worktree and registry entry remain

### Hook Examples

#### Example 1: Port Assignment

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
NODE_ENV=development
EOF

echo "Assigned port: $PORT"
```

#### Example 2: Database Branch Setup (PlanetScale)

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

#### Example 3: Dependency Installation

```bash
#!/bin/bash
# .dual/hooks/install-dependencies.sh

set -e

echo "Installing dependencies for: $DUAL_CONTEXT_NAME"

cd "$DUAL_CONTEXT_PATH"

# Install npm dependencies
if [ -f "package.json" ]; then
  pnpm install
fi

# Install Python dependencies
if [ -f "requirements.txt" ]; then
  pip install -r requirements.txt
fi

echo "Dependencies installed"
```

#### Example 4: Pre-Delete Backup

```bash
#!/bin/bash
# .dual/hooks/backup-data.sh

set -e

echo "Backing up data for: $DUAL_CONTEXT_NAME"

# Create backup directory
mkdir -p ~/backups

# Backup database
pg_dump myapp_${DUAL_CONTEXT_NAME} > ~/backups/${DUAL_CONTEXT_NAME}_$(date +%Y%m%d).sql

echo "Data backed up to ~/backups/${DUAL_CONTEXT_NAME}_$(date +%Y%m%d).sql"
```

#### Example 5: Database Cleanup

```bash
#!/bin/bash
# .dual/hooks/cleanup-database.sh

set -e

echo "Cleaning up database for: $DUAL_CONTEXT_NAME"

# Drop local database
dropdb myapp_${DUAL_CONTEXT_NAME} --if-exists

# Delete PlanetScale branch
pscale branch delete myapp "$DUAL_CONTEXT_NAME" --force

echo "Database branch deleted"
```

#### Example 6: Team Notification

```bash
#!/bin/bash
# .dual/hooks/notify-team.sh

set -e

echo "Notifying team about: $DUAL_CONTEXT_NAME"

# Send Slack notification
curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
  -H 'Content-Type: application/json' \
  -d "{\"text\":\"Worktree deleted: $DUAL_CONTEXT_NAME\"}"

echo "Team notified"
```

#### Example 7: Multi-Service Port Assignment

```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh

set -e

echo "Setting up multi-service environment for: $DUAL_CONTEXT_NAME"

# Calculate base port from context hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

# Assign ports to each service
WEB_PORT=$((CONTEXT_BASE + 1))
API_PORT=$((CONTEXT_BASE + 2))
WORKER_PORT=$((CONTEXT_BASE + 3))

# Write web service env
cat > "$DUAL_CONTEXT_PATH/apps/web/.env.local" <<EOF
PORT=$WEB_PORT
API_URL=http://localhost:$API_PORT
EOF

# Write api service env
cat > "$DUAL_CONTEXT_PATH/apps/api/.env.local" <<EOF
PORT=$API_PORT
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
EOF

# Write worker service env
cat > "$DUAL_CONTEXT_PATH/apps/worker/.env.local" <<EOF
PORT=$WORKER_PORT
API_URL=http://localhost:$API_PORT
EOF

echo "Assigned ports:"
echo "  web:    $WEB_PORT"
echo "  api:    $API_PORT"
echo "  worker: $WORKER_PORT"
```

### Hook Best Practices

1. **Always use `set -e`**: Exit immediately on errors
2. **Provide feedback**: Echo progress messages for user visibility
3. **Check prerequisites**: Verify required tools are installed
4. **Handle failures gracefully**: Clean up partial state on error
5. **Use environment variables**: Access `DUAL_*` variables for context
6. **Keep scripts focused**: One script per concern (database, env, deps)
7. **Make scripts idempotent**: Safe to run multiple times
8. **Add comments**: Document what each section does

---

## Environment Management

The `dual env` command suite provides comprehensive environment variable management with support for layered configurations, service-specific overrides, and full dotenv compatibility.

### dual env show

Display environment summary and variable information for the current context.

#### Syntax

```bash
dual env show [--values] [--base-only] [--overrides-only] [--json] [--service <name>]
```

#### Options

- `--values` - Show all variable values (truncated for security by default)
- `--base-only` - Show only base environment variables
- `--overrides-only` - Show only context-specific overrides
- `--json` - Output as JSON for machine processing
- `--service <name>` - Show overrides for a specific service

#### Examples

##### Basic Summary

```bash
dual env show
```

Output:
```
Base:      .env.base (15 vars)
Service:   12 vars
Overrides: 3 vars
Effective: 28 vars total

Overrides for context 'feature-auth':
  DATABASE_URL=postgresql://localhost/myapp_featu...
  PORT=4237
  DEBUG=true
```

##### Show All Values

```bash
dual env show --values
```

Output:
```
Base:      .env.base (15 vars)
Service:   12 vars
Overrides: 3 vars
Effective: 28 vars total

Overrides for context 'feature-auth':
  DATABASE_URL=postgresql://localhost/myapp_feature-auth
  PORT=4237
  DEBUG=true
```

##### Show Only Base Variables

```bash
dual env show --base-only
```

Output:
```
Base environment (.env.base):
API_VERSION
APP_NAME
BASE_URL
DATABASE_HOST
LOG_LEVEL
...
```

##### Show Only Overrides

```bash
dual env show --overrides-only
```

Output:
```
Overrides for context 'feature-auth':
DATABASE_URL=postgresql://localhost/myapp_feature-auth
PORT=4237
DEBUG=true
```

##### JSON Output

```bash
dual env show --json
```

Output:
```json
{
  "context": "feature-auth",
  "baseFile": ".env.base",
  "stats": {
    "baseVars": 15,
    "serviceVars": 12,
    "overrideVars": 3,
    "totalVars": 28
  },
  "base": {
    "APP_NAME": "MyApp",
    "BASE_URL": "http://localhost:3000"
  },
  "service": {
    "PORT": "3000",
    "DATABASE_URL": "postgresql://localhost/myapp"
  },
  "overrides": {
    "DATABASE_URL": "postgresql://localhost/myapp_feature-auth",
    "PORT": "4237",
    "DEBUG": "true"
  }
}
```

##### Service-Specific Overrides

```bash
dual env show --service api
```

Shows overrides specific to the "api" service.

---

### dual env set

Set a context-specific environment variable override.

#### Syntax

```bash
dual env set <key> <value> [--service <name>]
```

#### Arguments

- `<key>` - Environment variable name
- `<value>` - Environment variable value

#### Options

- `--service <name>` - Set override for a specific service (otherwise global)

#### Examples

##### Set Global Override

```bash
dual env set DATABASE_URL "postgresql://localhost/myapp_feature"
```

Output:
```
Set DATABASE_URL=postgresql://localhost/myapp_feature for context 'feature-auth' (global)
Context 'feature-auth' now has 4 override(s) (4 global, 0 service-specific)
```

##### Set Service-Specific Override

```bash
dual env set --service api PORT 5000
dual env set --service web PORT 3000
```

Output:
```
Set PORT=5000 for service 'api' in context 'feature-auth'
Context 'feature-auth' now has 5 override(s) (3 global, 2 service-specific)
```

This allows different services to have different values for the same variable.

##### Override Base Variable

```bash
# If DATABASE_URL exists in base environment
dual env set DATABASE_URL "mysql://localhost/custom_db"
```

Output:
```
[dual] Warning: Overriding variable "DATABASE_URL" from base environment
Set DATABASE_URL=mysql://localhost/custom_db for context 'feature-auth' (global)
```

---

### dual env unset

Remove a context-specific environment variable override.

#### Syntax

```bash
dual env unset <key> [--service <name>]
```

#### Arguments

- `<key>` - Environment variable name to remove

#### Options

- `--service <name>` - Remove override for a specific service

#### Examples

##### Remove Global Override

```bash
dual env unset DATABASE_URL
```

Output:
```
Removed override for DATABASE_URL in context 'feature-auth'
Fallback to base value: DATABASE_URL=postgresql://localhost/myapp
```

##### Remove Service-Specific Override

```bash
dual env unset --service api PORT
```

Output:
```
Removed override for PORT in service 'api' for context 'feature-auth'
```

---

### dual env export

Export the complete merged environment to stdout.

#### Syntax

```bash
dual env export [--format <format>] [--service <name>]
```

#### Options

- `--format <format>` - Output format: `dotenv`, `json`, or `shell` (default: dotenv)
- `--service <name>` - Export for a specific service

#### Examples

##### Dotenv Format (Default)

```bash
dual env export
```

Output:
```
API_VERSION=v1
APP_NAME=MyApp
BASE_URL=http://localhost:3000
DATABASE_URL=postgresql://localhost/myapp_feature-auth
DEBUG=true
PORT=4237
```

##### JSON Format

```bash
dual env export --format json
```

Output:
```json
{
  "API_VERSION": "v1",
  "APP_NAME": "MyApp",
  "BASE_URL": "http://localhost:3000",
  "DATABASE_URL": "postgresql://localhost/myapp_feature-auth",
  "DEBUG": "true",
  "PORT": "4237"
}
```

##### Shell Export Format

```bash
dual env export --format shell
```

Output:
```bash
export API_VERSION='v1'
export APP_NAME='MyApp'
export BASE_URL='http://localhost:3000'
export DATABASE_URL='postgresql://localhost/myapp_feature-auth'
export DEBUG='true'
export PORT='4237'
```

You can source this output directly:
```bash
eval "$(dual env export --format shell)"
```

##### Save to File

```bash
dual env export > .env.local
```

##### Export Service Environment

```bash
dual env export --service api > apps/api/.env.local
```

---

### dual env check

Validate environment configuration for the current context.

#### Syntax

```bash
dual env check
```

#### Examples

##### Healthy Environment

```bash
dual env check
```

Output:
```
✓ Base environment file exists: .env.base (15 vars)
✓ Context detected: feature-auth
✓ Context has 5 environment override(s) (3 global, 2 service-specific)

✓ Environment configuration is valid
```

Exit code: 0

##### Environment with Issues

```bash
dual env check
```

Output:
```
Error: Base environment file (.env.base) is not readable: file not found
✓ Context detected: feature-auth
Error: Context 'feature-auth' not found in registry

❌ Environment configuration has issues
```

Exit code: 1

#### Use Cases

- **Pre-deployment validation**: Ensure environment is configured correctly
- **CI/CD checks**: Validate environment in automated pipelines
- **Troubleshooting**: Diagnose configuration problems
- **Health checks**: Verify setup after changes

---

### dual env diff

Compare environment variables between two contexts.

#### Syntax

```bash
dual env diff <context1> <context2>
```

#### Arguments

- `<context1>` - First context name
- `<context2>` - Second context name

#### Examples

##### Compare Main and Feature Branch

```bash
dual env diff main feature-auth
```

Output:
```
Comparing environments: main → feature-auth

Changed:
  DATABASE_URL: postgresql://localhost/myapp → postgresql://localhost/myapp_feature-auth
  PORT: 3000 → 4237

Added:
  DEBUG=true
  FEATURE_FLAG_AUTH=enabled

Removed:
  OLD_CONFIG=legacy_value
```

##### Compare Two Feature Branches

```bash
dual env diff feature-a feature-b
```

Output:
```
Comparing environments: feature-a → feature-b

Changed:
  PORT: 4001 → 4002
  DATABASE_URL: postgresql://localhost/myapp_a → postgresql://localhost/myapp_b

No variables added or removed
```

##### No Differences

```bash
dual env diff main staging
```

Output:
```
Comparing environments: main → staging

No differences found
```

#### Use Cases

- **Environment auditing**: Verify differences between contexts
- **Troubleshooting**: Identify why two contexts behave differently
- **Documentation**: Document environment variations
- **Validation**: Ensure staging matches production

---

### dual env remap

Regenerate service-specific .env files from registry.

#### Syntax

```bash
dual env remap
```

#### What It Does

Reads environment overrides from the registry and regenerates all service-specific environment files at `.dual/.local/service/<service>/.env`.

This command is automatically run when you use `dual env set` or `dual env unset`, so you typically don't need to run it manually.

#### When to Use

- After manually editing the registry file
- When env files are out of sync with registry
- After recovering from corruption
- When troubleshooting environment issues

#### Examples

```bash
dual env remap
```

Output:
```
[dual] Regenerating service env files for context 'feature-auth'...
[dual] Service env files regenerated successfully
  Files written to: /Users/dev/Code/myproject/.dual/.local/service/<service>/.env
```

---

## Command Execution

### dual run

Run a command with full environment injection from all layers.

#### Syntax

```bash
dual run [--service <name>] <command> [args...]
```

#### Options

- `--service <name>` - Explicitly specify service (auto-detected if not provided)

#### Arguments

- `<command>` - Command to execute
- `[args...]` - Command arguments

#### What It Does

The `dual run` command executes a command with complete environment variable injection from all three layers:

1. Base environment (.env.base if configured)
2. Service-specific environment (<service-path>/.env)
3. Context-specific overrides (.dual/.local/service/<service>/.env)

This enables running services with isolated environments per worktree without requiring applications to load dotenv files manually.

#### Examples

##### Run Node.js Server

```bash
dual run node server.js
```

Output:
```
[dual] Running: node [server.js]
[dual] Service: api
[dual] Context: feature-auth
[dual] Environment variables loaded: 28

Server listening on port 4237...
```

##### Run npm Start

```bash
dual run npm start
```

Output:
```
[dual] Running: npm [start]
[dual] Service: web
[dual] Context: feature-auth
[dual] Environment variables loaded: 32

> myapp@1.0.0 start
> react-scripts start
```

##### Run Python Application

```bash
dual run python app.py
```

##### Explicitly Specify Service

```bash
# Run command in API service context
dual run --service api npm start

# Run command in web service context
dual run --service web npm run dev
```

##### Run with Complex Arguments

```bash
dual run npm run test -- --coverage --watch
dual run node server.js --port 8080 --host 0.0.0.0
```

#### Service Detection

If `--service` is not specified, `dual run` automatically detects the service based on your current working directory:

```bash
cd apps/api
dual run npm start  # Automatically uses "api" service

cd apps/web
dual run npm start  # Automatically uses "web" service
```

#### Environment Injection

The command inherits your shell environment plus injected variables from dual. Variables from dual override shell variables with the same name.

Priority (lowest to highest):
1. Shell environment
2. Base environment
3. Service environment
4. Context overrides

#### Use Cases

- **Development servers**: Run dev servers with context-specific ports and configs
- **Scripts**: Execute build/test scripts with proper environment
- **Database migrations**: Run migrations with context-specific database URLs
- **Testing**: Run tests with isolated test databases
- **CLI tools**: Execute CLI tools with proper configuration

---

## Utility Commands

### dual doctor

Diagnose configuration and registry health.

#### Syntax

```bash
dual doctor
```

#### Examples

##### Healthy Configuration

```bash
dual doctor
```

Output:
```
Checking dual configuration...

✓ Config file found: /Users/dev/Code/myproject/dual.config.yml
✓ Config version: 1
✓ Services: 3 configured
  - api (apps/api)
  - web (apps/web)
  - worker (apps/worker)
✓ Worktrees configuration: ../worktrees
✓ Hooks configuration: 6 hooks configured
✓ Registry file found: /Users/dev/Code/myproject/.dual/.local/registry.json
✓ Registry is readable and valid
✓ Contexts: 3 registered
  - main (main repository)
  - feature-auth (worktree)
  - feature-api (worktree)

All checks passed!
```

##### Configuration Issues

```bash
dual doctor
```

Output:
```
Checking dual configuration...

✓ Config file found: /Users/dev/Code/myproject/dual.config.yml
✓ Config version: 1
✓ Services: 2 configured
  - web (apps/web)
  - api (apps/api)
✗ Worktrees configuration: not configured
  Hint: Add worktrees section to use 'dual create'
✗ Hook script not found: setup-database.sh
  Hint: Create .dual/hooks/setup-database.sh and make it executable
✗ Hook script not executable: cleanup-database.sh
  Hint: Run 'chmod +x .dual/hooks/cleanup-database.sh'
✓ Registry file found: /Users/dev/Code/myproject/.dual/.local/registry.json
✗ Registry corruption: context "feature-old" references non-existent worktree
  Hint: Run 'dual context list' and manually edit registry if needed

Issues found. Please resolve the issues above.
```

#### What It Checks

- **Config file**: Exists and is readable
- **Config version**: Supported version (currently 1)
- **Services**: Valid service definitions
- **Worktrees**: Configuration exists (if using dual create/delete)
- **Hooks**: Scripts exist and are executable
- **Registry**: File exists, is readable, and is valid JSON
- **Contexts**: Registered contexts are valid

#### Use Cases

- **Troubleshooting**: Diagnose configuration issues
- **Setup Verification**: Ensure everything is configured correctly
- **Pre-deployment Checks**: Validate before committing config changes
- **Team Onboarding**: Help new developers verify their setup

---

## Dotenv Compatibility Features

As of v0.3.x, `dual` provides full compatibility with the Node.js dotenv specification using the industry-standard godotenv library. This means your .env files support advanced features beyond simple KEY=value pairs.

### Multiline Values

Use quotes (double or single) to define values that span multiple lines. This is especially useful for certificates, keys, SQL queries, and formatted text.

#### Examples

##### SSL Certificate

```bash
# .env
TLS_CERT="-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKLdQVPy90WjMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMjQwMTAxMDAwMDAwWhcNMjUwMTAxMDAwMDAwWjBF
-----END CERTIFICATE-----"
```

##### SQL Query

```bash
# .env
SQL_QUERY="SELECT
  users.id,
  users.name,
  users.email
FROM users
WHERE users.active = true
ORDER BY users.created_at DESC"
```

##### JSON Configuration

```bash
# .env
CONFIG_JSON='{
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "myapp"
  },
  "cache": {
    "enabled": true,
    "ttl": 3600
  }
}'
```

##### Email Template

```bash
# .env
EMAIL_TEMPLATE="<html>
<body>
  <h1>Welcome!</h1>
  <p>Thank you for signing up.</p>
</body>
</html>"
```

---

### Variable Expansion

Reference other environment variables using `${VAR}` or `$VAR` syntax. This enables DRY (Don't Repeat Yourself) configuration.

#### Syntax

- `${VARIABLE}` - Standard expansion (recommended)
- `$VARIABLE` - Short form expansion

#### Examples

##### Basic Expansion

```bash
# .env
PROTOCOL=https
DOMAIN=example.com
PORT=3000

# Build complete URL from components
BASE_URL=${PROTOCOL}://${DOMAIN}:${PORT}
API_ENDPOINT=$BASE_URL/api/v1
```

When loaded:
```
PROTOCOL=https
DOMAIN=example.com
PORT=3000
BASE_URL=https://example.com:3000
API_ENDPOINT=https://example.com:3000/api/v1
```

##### Database Connection String

```bash
# .env
DB_USER=admin
DB_PASS=secret
DB_HOST=localhost
DB_PORT=5432
DB_NAME=myapp

# Build full connection URL
DATABASE_URL=postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}
```

Result: `postgresql://admin:secret@localhost:5432/myapp`

##### Nested Expansion

```bash
# .env
APP_NAME=MyApp
VERSION=1.0.0
FULL_NAME=${APP_NAME}-${VERSION}
USER_AGENT="${FULL_NAME} (${DOMAIN})"
```

##### Path Building

```bash
# .env
HOME=/users/developer
PROJECT_ROOT=$HOME/projects/myapp
CONFIG_PATH=${PROJECT_ROOT}/config
LOG_PATH=${PROJECT_ROOT}/logs
DATA_PATH=$PROJECT_ROOT/data
```

#### Quoting Behavior

- **Double quotes**: Variables are expanded
- **Single quotes**: Variables are NOT expanded (literal)

```bash
# .env
BASE_URL=http://localhost:3000

# Expanded
API_URL="${BASE_URL}/api"        # → http://localhost:3000/api

# Literal (not expanded)
TEMPLATE='Visit ${BASE_URL}'     # → Visit ${BASE_URL}
```

---

### Escape Sequences

In double-quoted strings, escape sequences are processed. This allows you to include special characters in values.

#### Supported Escape Sequences

- `\n` - Newline
- `\t` - Tab
- `\\` - Backslash
- `\"` - Double quote
- `\r` - Carriage return

#### Examples

##### Line Breaks

```bash
# .env
MESSAGE="Line 1\nLine 2\nLine 3"
```

When used, this creates:
```
Line 1
Line 2
Line 3
```

##### Escaped Quotes

```bash
# .env
QUOTE_MESSAGE="She said \"Hello\" to the world"
```

Result: `She said "Hello" to the world`

##### Windows Paths

```bash
# .env
WINDOWS_PATH="C:\\Program Files\\MyApp"
```

Result: `C:\Program Files\MyApp`

##### Escaped Variables (Literal $)

```bash
# .env
# Use single quotes for literal dollar signs
LITERAL_VAR='Price: $99.99'              # → Price: $99.99

# Or escape in double quotes
ESCAPED_VAR="Price: \$99.99"             # → Price: $99.99
```

#### Single vs Double Quotes

```bash
# .env
# Double quotes: Escape sequences ARE processed
DOUBLE="Hello\nWorld"    # → Hello
                        #    World

# Single quotes: Escape sequences are NOT processed (literal)
SINGLE='Hello\nWorld'    # → Hello\nWorld
```

---

### Inline Comments

Add comments after variable definitions using `#`. Comments are stripped from values.

#### Examples

##### Basic Comments

```bash
# .env
PORT=3000 # Application port
HOST=localhost # Development host
DEBUG=true # Enable debug mode
```

##### Section Organization

```bash
# .env
# Database configuration
DATABASE_HOST=localhost  # Primary database server
DATABASE_PORT=5432       # PostgreSQL default port
DATABASE_NAME=myapp      # Application database

# API configuration
API_VERSION=v1           # Current API version
API_TIMEOUT=30           # Request timeout in seconds
```

#### Important Notes

**Comments only work outside quoted values:**

```bash
# .env
# ✓ Comment is stripped
PORT=3000 # This is a comment

# ✓ Hash is preserved (inside quotes)
COLOR="#FF00FF"
CHANNEL="#general"

# ✓ Hash is part of value (inside quotes)
PASSWORD="p@ssw0rd#123"

# ✗ This is ambiguous - quote the value if it contains #
MESSAGE=Hello#World  # Could be interpreted as "Hello" with comment "World"

# ✓ Clearer - quote values with #
MESSAGE="Hello#World"  # No ambiguity
```

---

### Complex Quoting

Dual supports nested quotes and mixed quoting styles for complex configuration.

#### Examples

##### JSON in Environment Variables

```bash
# .env
JSON_CONFIG='{"port": 3000, "host": "localhost", "ssl": true}'
NESTED_JSON='{"users": [{"name": "Alice"}, {"name": "Bob"}]}'
```

##### Mixed Quotes

```bash
# .env
# Single quotes inside double quotes
MIXED_QUOTES="She said 'Hello' to me"

# Double quotes inside single quotes (less common)
ALTERNATIVE='He said "Goodbye"'
```

##### Complex Strings

```bash
# .env
# Regex patterns
REGEX_PATTERN='^[a-zA-Z0-9]+$'

# Math expressions
MATH_EXPR='2+2=4'

# URLs with parameters
API_URL='https://api.example.com/v1?key=value&format=json'
```

#### Quote Selection Guide

Use **single quotes** when:
- Value should be literal (no expansion)
- Value contains double quotes
- Value is JSON or complex data
- Value should preserve exact formatting

Use **double quotes** when:
- You need variable expansion
- You need escape sequences (\n, \t, etc.)
- Value contains single quotes
- Value has spaces

Use **no quotes** when:
- Value is simple (no spaces, no special characters)
- Value is a number or boolean
- You want trailing spaces trimmed

#### Examples by Use Case

```bash
# .env
# Simple values - no quotes needed
PORT=3000
DEBUG=true
MAX_CONNECTIONS=100

# Spaces - use quotes
APP_NAME="My Application"

# Variable expansion - use double quotes
API_URL="${BASE_URL}/api"

# Literal dollar signs - use single quotes
PRICE='$99.99'

# JSON - use single quotes (no expansion needed)
CONFIG='{"key": "value"}'

# Escape sequences - use double quotes
MULTILINE="Line 1\nLine 2"

# Paths with backslashes - use double quotes with escaping
WINDOWS_PATH="C:\\Program Files\\App"
```

---

## Environment Loading System

Dual implements a sophisticated three-layer environment system that provides consistent, predictable behavior across all commands while supporting both global and service-specific configurations.

### Three-Layer Priority System

Environment variables are loaded and merged from three layers, with higher layers overriding lower layers:

```
Layer 1: Base Environment (Lowest Priority)
    ↓
Layer 2: Service-Specific Environment
    ↓
Layer 3: Context-Specific Overrides (Highest Priority)
```

#### Layer 1: Base Environment

**Source**: Configured base file (e.g., `.env.base`)

**Purpose**: Shared variables across all services and contexts

**Configuration**: Set `env.baseFile` in `dual.config.yml`

```yaml
# dual.config.yml
version: 1
env:
  baseFile: .env.base
```

**Example `.env.base`**:
```bash
# Shared across all services and contexts
APP_NAME=MyApp
API_VERSION=v1
LOG_LEVEL=info
ENVIRONMENT=development
NODE_ENV=development
```

**When to use**:
- Variables shared by all services
- Default values for common configuration
- Environment-wide settings

#### Layer 2: Service-Specific Environment

**Source**: `.env` file in service directory (e.g., `apps/api/.env`)

**Purpose**: Configuration specific to a single service

**Auto-loaded**: Dual automatically loads `<service-path>/.env`

**Example `apps/api/.env`**:
```bash
# Specific to API service
PORT=3000
DATABASE_URL=postgresql://localhost/myapp
REDIS_URL=redis://localhost:6379
MAX_CONNECTIONS=100
```

**Example `apps/web/.env`**:
```bash
# Specific to web service
PORT=3001
API_URL=http://localhost:3000
PUBLIC_URL=http://localhost:3001
```

**When to use**:
- Service-specific configuration
- Default ports, URLs, connection strings
- Service dependencies

#### Layer 3: Context-Specific Overrides

**Source**: `.dual/.local/service/<service>/.env` (generated automatically)

**Purpose**: Override variables for a specific worktree context

**Management**: Use `dual env set/unset` commands

**Example `.dual/.local/service/api/.env`** (for feature-auth context):
```bash
# Overrides for feature-auth context
PORT=4237
DATABASE_URL=postgresql://localhost/myapp_feature_auth
DEBUG=true
```

**When to use**:
- Context-specific values (ports, database names)
- Feature-specific configuration
- Temporary overrides during development

#### Complete Example

**Setup**:
```
# .env.base (Layer 1 - base)
APP_NAME=MyApp
LOG_LEVEL=info

# apps/api/.env (Layer 2 - service)
PORT=3000
DATABASE_URL=postgresql://localhost/myapp

# .dual/.local/service/api/.env (Layer 3 - context override)
PORT=4237
DATABASE_URL=postgresql://localhost/myapp_feature_auth
```

**Effective environment** (when running `dual run` in api service, feature-auth context):
```
APP_NAME=MyApp                                              # From base
LOG_LEVEL=info                                              # From base
PORT=4237                                                   # From override (wins over service)
DATABASE_URL=postgresql://localhost/myapp_feature_auth    # From override (wins over service)
```

---

### Service Detection

Dual automatically detects which service you're working in based on your current directory.

#### How It Works

1. Gets your current working directory
2. Resolves any symlinks
3. Matches against service paths in `dual.config.yml`
4. Uses longest path match for nested structures

#### Example

```yaml
# dual.config.yml
services:
  api:
    path: ./apps/api
  web:
    path: ./apps/web
  shared:
    path: ./packages/shared
```

```bash
# Working in API service
cd /Users/dev/Code/myproject/apps/api
dual run npm start
# → Automatically detects "api" service
# → Loads apps/api/.env
# → Applies api-specific overrides

# Working in web service
cd /Users/dev/Code/myproject/apps/web
dual run npm start
# → Automatically detects "web" service
# → Loads apps/web/.env
# → Applies web-specific overrides
```

#### Explicit Override

You can always override auto-detection:

```bash
# Force use of api service even if in different directory
dual run --service api npm test

# Force use of web service
dual env show --service web
```

---

### Consistent Behavior

All environment-related commands use the unified `LoadLayeredEnv()` function, ensuring consistent behavior:

- **`dual env show`**: Shows all three layers
- **`dual env export`**: Exports merged environment
- **`dual env check`**: Validates all layers
- **`dual env diff`**: Compares merged environments
- **`dual run`**: Injects merged environment

#### Example: Consistency Across Commands

```bash
# Set an override
dual env set DATABASE_URL "postgresql://localhost/custom_db"

# View it
dual env show
# → Shows override in "Overrides" section

# Export it
dual env export | grep DATABASE_URL
# → DATABASE_URL=postgresql://localhost/custom_db

# Use it
dual run node server.js
# → Server receives DATABASE_URL from override

# Check it
dual env check
# → Validates all layers including override

# Compare it
dual env diff main feature-branch
# → Shows DATABASE_URL difference
```

All commands see the same environment, guaranteed.

---

## Configuration Reference

The `dual.config.yml` file defines your project's configuration.

### Complete Schema

```yaml
version: 1

# Required: Service definitions
services:
  <service-name>:
    path: <relative-path>      # Required: path from project root
    envFile: <relative-path>   # Optional: env file reference

# Optional: Base environment configuration
env:
  baseFile: <relative-path>    # Optional: shared base environment file

# Optional: Worktree management configuration
worktrees:
  path: <relative-path>        # Where to create worktrees
  naming: "{branch}"           # Directory naming pattern

# Optional: Lifecycle hooks
hooks:
  postWorktreeCreate:          # After creating worktree
    - script1.sh
    - script2.sh
  preWorktreeDelete:           # Before deleting worktree (files exist)
    - script3.sh
  postWorktreeDelete:          # After deleting worktree (files gone)
    - script4.sh
```

### Example Configuration

```yaml
version: 1

services:
  web:
    path: ./apps/web
    envFile: .env.local
  api:
    path: ./apps/api
    envFile: .env
  worker:
    path: ./apps/worker
    envFile: .env.local

env:
  baseFile: .env.base

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

### Configuration Notes

- **version**: Must be `1` (only supported version)
- **services**: At least one service is required
- **env.baseFile**: Optional. Path to shared base environment file (relative to project root)
- **worktrees.path**: Relative to project root (e.g., `../worktrees` creates sibling directory)
- **worktrees.naming**: Currently only supports `{branch}` placeholder
- **hooks**: All script paths are relative to `$PROJECT_ROOT/.dual/hooks/`
- **Hook scripts**: Must be executable (`chmod +x`)

### Worktree Naming Patterns

The `worktrees.naming` pattern controls how worktree directories are named:

**Default**: `{branch}`
```yaml
worktrees:
  naming: "{branch}"
```
Branch `feature/auth` → directory `feature-auth`

**Custom prefix**: `wt-{branch}`
```yaml
worktrees:
  naming: "wt-{branch}"
```
Branch `feature/auth` → directory `wt-feature-auth`

**Note**: Slashes in branch names are automatically replaced with dashes.

---

## Project-Local Registry

The registry tracks all contexts (main repository and worktrees) for a project.

### Registry Location

**v0.3.0**: `$PROJECT_ROOT/.dual/.local/registry.json` (project-local)

**Important changes from v0.2.x**:
- No longer stored in `~/.dual/registry.json` (global)
- Each project has its own registry
- All worktrees of a repository share the parent repo's registry
- Should be added to `.gitignore` (contains local paths)

### Registry Structure

```json
{
  "projects": {
    "/Users/dev/Code/myproject": {
      "contexts": {
        "main": {
          "path": "",
          "created": "2025-10-01T10:00:00Z"
        },
        "feature-auth": {
          "path": "/Users/dev/Code/myproject-wt/feature-auth",
          "created": "2025-10-10T14:30:00Z"
        }
      }
    }
  }
}
```

### Registry Operations

**Automatic management**: The registry is automatically updated by `dual create` and `dual delete`.

**Manual editing**: Generally not recommended, but you can edit the registry file if needed (be careful!).

**File locking**: The registry uses file locking to prevent corruption from concurrent dual operations.

**Auto-recovery**: If the registry is corrupted, dual will create a new empty registry.

### Add to .gitignore

```bash
echo "/.dual/.local/" >> .gitignore
```

The registry contains local paths that shouldn't be committed.

---

## Debug & Verbose Options

All `dual` commands support verbose and debug output flags for troubleshooting.

### Flags

#### --verbose or -v

Enable verbose output showing detailed operation steps.

```bash
dual --verbose create feature-x
dual -v context list
```

Output example:
```
[dual] Loading configuration from: /Users/dev/Code/myproject/dual.config.yml
[dual] Project root: /Users/dev/Code/myproject
[dual] Worktrees path: /Users/dev/Code/myproject-wt
[dual] Creating worktree for: feature-x
[dual] Target path: /Users/dev/Code/myproject-wt/feature-x
[dual] Executing git worktree add...
[dual] Registering context in registry...
[dual] Executing postWorktreeCreate hooks...
[dual] Successfully created worktree: feature-x
```

#### --debug or -d

Enable debug output showing maximum detail including internal state. Implies `--verbose`.

```bash
dual --debug create feature-x
```

Output example:
```
[dual] Debug mode enabled
[dual] Loading configuration from: /Users/dev/Code/myproject/dual.config.yml
[dual] Config loaded successfully
[dual] Services: map[api:{apps/api .env} web:{apps/web .env.local}]
[dual] Worktrees config: {../worktrees {branch}}
[dual] Hooks config: map[postWorktreeCreate:[setup-database.sh setup-environment.sh]]
[dual] Project root: /Users/dev/Code/myproject
[dual] Loading registry from: /Users/dev/Code/myproject/.dual/.local/registry.json
[dual] Registry loaded: 2 contexts
[dual] Creating worktree for: feature-x
[dual] Naming pattern: {branch}
[dual] Target directory: feature-x
[dual] Target path: /Users/dev/Code/myproject-wt/feature-x
[dual] Executing: git worktree add /Users/dev/Code/myproject-wt/feature-x -b feature-x
[dual] Git command succeeded
[dual] Updating registry...
[dual] Registry updated
[dual] Executing 2 postWorktreeCreate hooks...
[dual] Successfully created worktree: feature-x
```

### Environment Variable

You can also enable debug mode via environment variable:

```bash
export DUAL_DEBUG=1
dual create feature-x
```

This is useful for:
- Shell scripts that call dual
- CI/CD pipelines
- Persistent debugging sessions

### Examples

#### Debug Worktree Creation

```bash
dual --debug create feature-x
```

#### Verbose Context List

```bash
dual --verbose context list
```

#### Debug Doctor

```bash
dual --debug doctor
```

### Use Cases

- **Troubleshooting**: Diagnose configuration or execution issues
- **Development**: Debug dual itself during development
- **CI/CD**: Enable verbose output in pipelines for better logs
- **Learning**: Understand how dual makes decisions
- **Bug Reports**: Include debug output when reporting issues

---

## Migration Guide

### Upgrading from Previous Versions

If you're upgrading to v0.3.x (or later) from an earlier version, this section will help you understand the breaking changes and how to migrate your .env files.

#### What Changed

The v0.3.x release introduced full dotenv compatibility using the godotenv library. This brings powerful new features but also includes some breaking changes in how .env files are parsed.

**New Features**:
- Multiline values support
- Variable expansion (${VAR} and $VAR)
- Escape sequence processing (\n, \t, \\, \")
- Inline comments support
- Complex quoting behaviors

**Breaking Changes**:
1. Variable expansion is now enabled by default
2. Escape sequences are processed in double quotes
3. Inline comments are stripped from values

---

### Breaking Changes

#### 1. Variable Expansion (Now Enabled)

**Previously**: `${VAR}` was treated as literal text

**Now**: `${VAR}` is expanded to the value of VAR

##### Impact

```bash
# Old behavior (before v0.3.x):
BASE_URL=http://localhost:3000
API_URL=${BASE_URL}/api  # Result: "${BASE_URL}/api" (literal)

# New behavior (v0.3.x+):
BASE_URL=http://localhost:3000
API_URL=${BASE_URL}/api  # Result: "http://localhost:3000/api" (expanded)
```

##### Migration

If you need literal `${VAR}` values, use single quotes:

```bash
# If you want the literal string "${BASE_URL}/api"
LITERAL_VALUE='${BASE_URL}/api'

# Or escape the dollar sign
ESCAPED_VALUE="\${BASE_URL}/api"
```

#### 2. Escape Sequences (Now Processed)

**Previously**: `\n` in double quotes was literal text

**Now**: `\n` is processed as a newline character

##### Impact

```bash
# Old behavior (before v0.3.x):
MESSAGE="Hello\nWorld"  # Result: "Hello\nWorld" (literal)

# New behavior (v0.3.x+):
MESSAGE="Hello\nWorld"  # Result: "Hello
                        #          World" (newline)
```

##### Migration

If you need literal backslashes, use single quotes or double escaping:

```bash
# Single quotes (recommended for literals)
LITERAL_PATH='C:\Users\Name'

# Double escaping
ESCAPED_PATH="C:\\Users\\Name"
```

#### 3. Inline Comments (Now Stripped)

**Previously**: Everything after `=` was the value (including `#`)

**Now**: Content after `#` (outside quotes) is treated as a comment

##### Impact

```bash
# Old behavior (before v0.3.x):
VALUE=hello#world  # Result: "hello#world"

# New behavior (v0.3.x+):
VALUE=hello#world  # Result: "hello" (comment stripped)
```

##### Migration

If you need `#` in values, quote them:

```bash
# Quoted values preserve # characters
COLOR="#FF00FF"
CHANNEL="#general"
HASHTAG="hello#world"
```

---

### Migration Examples

#### Example 1: API URLs with Variable References

**Before (v0.2.x)**:
```bash
# .env
BASE_URL=http://localhost:3000
# This was literal text
API_URL=${BASE_URL}/api/v1
```

**After (v0.3.x+)**:

Option 1 - Use the expansion (recommended):
```bash
# .env
BASE_URL=http://localhost:3000
# Now this expands automatically
API_URL=${BASE_URL}/api/v1  # ← Becomes: http://localhost:3000/api/v1
```

Option 2 - Keep it literal (if needed):
```bash
# .env
BASE_URL=http://localhost:3000
# Use single quotes for literal
TEMPLATE_URL='${BASE_URL}/api/v1'  # ← Stays: ${BASE_URL}/api/v1
```

#### Example 2: Windows Paths

**Before (v0.2.x)**:
```bash
# .env
WINDOWS_PATH="C:\Program Files\MyApp"  # Worked as-is
```

**After (v0.3.x+)**:

Option 1 - Use single quotes:
```bash
# .env
WINDOWS_PATH='C:\Program Files\MyApp'  # No escaping needed
```

Option 2 - Escape backslashes in double quotes:
```bash
# .env
WINDOWS_PATH="C:\\Program Files\\MyApp"  # Backslashes escaped
```

#### Example 3: Messages with Literal \n

**Before (v0.2.x)**:
```bash
# .env
MESSAGE="Line 1\nLine 2"  # Result: "Line 1\nLine 2" (literal)
```

**After (v0.3.x+)**:

Option 1 - Use single quotes for literal \n:
```bash
# .env
MESSAGE='Line 1\nLine 2'  # Result: "Line 1\nLine 2" (literal)
```

Option 2 - Accept the newline (if desired):
```bash
# .env
MESSAGE="Line 1\nLine 2"  # Result: "Line 1
                          #          Line 2" (actual newline)
```

#### Example 4: Values with Hash Symbols

**Before (v0.2.x)**:
```bash
# .env
COLOR=#FF00FF  # Worked fine
TAG=version-1.0#beta  # Hash was part of value
```

**After (v0.3.x+)**:
```bash
# .env
# Quote values containing #
COLOR="#FF00FF"          # Preserve the hash
CHANNEL="#general"       # Preserve the hash
TAG="version-1.0#beta"   # Preserve the hash

# Or if comment is intentional, separate clearly
TAG=version-1.0  # beta tag
```

#### Example 5: Database URLs with Special Characters

**Before (v0.2.x)**:
```bash
# .env
DATABASE_URL=postgresql://user:p@ssw0rd@localhost/db
```

**After (v0.3.x+)**:
```bash
# .env
# Quote complex URLs to be safe
DATABASE_URL="postgresql://user:p@ssw0rd@localhost/db"

# Or use variable expansion for better organization
DB_USER=user
DB_PASS="p@ssw0rd"
DB_HOST=localhost
DB_NAME=myapp
DATABASE_URL="postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}/${DB_NAME}"
```

---

### Migration Checklist

Use this checklist when upgrading your .env files:

#### 1. Find Variable References

```bash
# Search for ${...} patterns in your .env files
grep '\${' .env* apps/*/.env
```

**Action**: Decide if you want expansion or literal values
- Keep unquoted or in double quotes → expansion
- Wrap in single quotes → literal

#### 2. Find Escape Sequences

```bash
# Search for backslashes in double-quoted strings
grep '\\' .env* apps/*/.env
```

**Action**: Check each occurrence
- `\n`, `\t`, etc. → decide if you want processing or literal
- Windows paths → use single quotes or double backslashes
- Regex patterns → use single quotes if they contain backslashes

#### 3. Find Hash Symbols in Values

```bash
# Search for values with # (excluding comments)
grep '=' .env* apps/*/.env | grep '#'
```

**Action**: Quote any values that contain `#` if it's not a comment

#### 4. Test Your Changes

```bash
# Show loaded environment to verify
dual env show --values

# Export and check for unexpected values
dual env export > test.env
cat test.env

# Run your application and test
dual run npm start
```

---

### Quick Migration Script

Use this script to help identify potential issues:

```bash
#!/bin/bash
# migration-check.sh - Check .env files for potential issues

echo "Checking for potential migration issues..."
echo

# Find all .env files
ENV_FILES=$(find . -name "*.env*" -o -name ".env")

for file in $ENV_FILES; do
  echo "Checking: $file"

  # Check for unquoted ${...}
  if grep -q '\${[^}]*}' "$file" 2>/dev/null; then
    echo "  ⚠ Found variable references (will be expanded)"
    grep -n '\${[^}]*}' "$file" | head -5
  fi

  # Check for \n or \t in double quotes
  if grep -q '".*\\[nt].*"' "$file" 2>/dev/null; then
    echo "  ⚠ Found escape sequences in double quotes (will be processed)"
    grep -n '".*\\[nt].*"' "$file" | head -5
  fi

  # Check for unquoted # in values
  if grep -E '^[^#]*=.*[^"]#[^"]*$' "$file" 2>/dev/null; then
    echo "  ⚠ Found potential inline comments (will be stripped)"
    grep -nE '^[^#]*=.*[^"]#[^"]*$' "$file" | head -5
  fi

  echo
done

echo "Migration check complete!"
echo "Review the warnings above and update your .env files as needed."
```

---

### Getting Help

If you encounter issues during migration:

1. **Check your environment loading**:
   ```bash
   dual env show --values
   dual env check
   ```

2. **Compare with expected values**:
   ```bash
   dual env export > actual.env
   # Compare with what you expect
   ```

3. **Use debug mode**:
   ```bash
   dual --debug env show
   ```

4. **Review the examples**: See the [Dotenv Compatibility Features](#dotenv-compatibility-features) section for detailed examples

5. **Check the test fixtures**: Look at `test/fixtures/*.env` in the dual repository for real examples

---

## Common Workflows

### First-Time Setup

```bash
# 1. Install dual
brew tap lightfastai/tap
brew install dual

# 2. Initialize project
cd ~/Code/myproject
dual init

# 3. Add services
dual service add web --path apps/web --env-file .env.local
dual service add api --path apps/api --env-file .env

# 4. Configure worktrees
# Edit dual.config.yml to add:
#   worktrees:
#     path: ../worktrees
#     naming: "{branch}"

# 5. Create hook scripts
mkdir -p .dual/hooks
cat > .dual/hooks/setup-environment.sh <<'EOF'
#!/bin/bash
set -e
echo "Setting up environment for: $DUAL_CONTEXT_NAME"
# Add your setup logic here
EOF
chmod +x .dual/hooks/setup-environment.sh

# 6. Verify setup
dual doctor

# 7. Add registry to .gitignore
echo "/.dual/.local/" >> .gitignore
```

### Creating a Feature Branch

```bash
# Using worktrees (recommended)
dual create feature-auth

# The worktree is created with:
# - Git worktree at configured location
# - Registered context
# - Lifecycle hooks executed (setup-environment.sh, install-dependencies.sh, etc.)

# Switch to the worktree
cd ../worktrees/feature-auth

# Start working
# Your hooks have already configured everything!
```

### Working on Multiple Features Simultaneously

```bash
# Terminal 1: Main branch
cd ~/Code/myproject/apps/web
npm run dev  # Runs on port from main context

# Terminal 2: Feature 1
cd ~/Code/myproject-wt/feature-auth/apps/web
npm run dev  # Runs on different port (assigned by hook)

# Terminal 3: Feature 2
cd ~/Code/myproject-wt/feature-api/apps/web
npm run dev  # Runs on another different port (assigned by hook)

# All three run simultaneously with isolated environments!
```

### Cleaning Up Old Worktrees

```bash
# List all contexts
dual context list

# Delete old worktrees
dual delete feature-old

# The deletion process:
# 1. Runs preWorktreeDelete hooks (backup data, cleanup databases)
# 2. Removes git worktree
# 3. Removes from registry
# 4. Runs postWorktreeDelete hooks (notify team, etc.)
```

### Implementing Custom Port Assignment

Create a hook script for port assignment:

```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh
set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate port based on context name hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Write to each service's env file
cat > "$DUAL_CONTEXT_PATH/apps/web/.env.local" <<EOF
PORT=$PORT
API_URL=http://localhost:$((PORT + 1))
EOF

cat > "$DUAL_CONTEXT_PATH/apps/api/.env.local" <<EOF
PORT=$((PORT + 1))
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
EOF

echo "Assigned ports:"
echo "  web: $PORT"
echo "  api: $((PORT + 1))"
```

Make it executable:
```bash
chmod +x .dual/hooks/setup-environment.sh
```

Configure in `dual.config.yml`:
```yaml
hooks:
  postWorktreeCreate:
    - setup-environment.sh
```

### Implementing Database Branch Management

Create hooks for database lifecycle:

```bash
# .dual/hooks/setup-database.sh
#!/bin/bash
set -e

echo "Creating database branch for: $DUAL_CONTEXT_NAME"

# Create PlanetScale branch
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main

# Get connection string
CONNECTION_URL=$(pscale connect myapp "$DUAL_CONTEXT_NAME" --format url)

# Write to env file
echo "DATABASE_URL=$CONNECTION_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "Database branch created"
```

```bash
# .dual/hooks/cleanup-database.sh
#!/bin/bash
set -e

echo "Cleaning up database for: $DUAL_CONTEXT_NAME"

# Delete PlanetScale branch
pscale branch delete myapp "$DUAL_CONTEXT_NAME" --force

echo "Database branch deleted"
```

Make executable and configure:
```bash
chmod +x .dual/hooks/setup-database.sh
chmod +x .dual/hooks/cleanup-database.sh
```

```yaml
hooks:
  postWorktreeCreate:
    - setup-database.sh
  preWorktreeDelete:
    - cleanup-database.sh
```

### CI/CD Integration

```bash
# In CI/CD pipeline

# 1. Install dual
brew install lightfastai/tap/dual

# 2. Initialize (if not committed)
dual init --force

# 3. List contexts (JSON output for parsing)
dual context list --json > contexts.json

# 4. Verify configuration
dual doctor

# 5. Use in build scripts
# (Access context information via dual context list --json)
```

---

## Next Steps

- See [EXAMPLES.md](EXAMPLES.md) for real-world usage scenarios
- See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- See [README.md](README.md) for project overview
- See [CLAUDE.md](CLAUDE.md) for development guidance
- See `.dual/hooks/README.md` for more hook examples

---

## Migration from v0.2.x

If you're upgrading from v0.2.x, see the migration guide:

### Removed Features

- **Command wrapper mode**: `dual <command>` no longer injects PORT
- **Port commands**: `dual port` and `dual ports` removed
- **`dual open` command**: Removed
- **`dual sync` command**: Removed
- **`dual env` commands**: Environment variable management removed
- **`dual context create`**: Deprecated in favor of `dual create <branch>`
- **Global registry**: Moved from `~/.dual/registry.json` to `$PROJECT_ROOT/.dual/.local/registry.json`

### New Features

- **Worktree lifecycle management**: `dual create` and `dual delete`
- **Hook system**: Lifecycle hooks for custom automation
- **Project-local registry**: Each project has its own registry

### Migration Steps

1. **Update configuration**: Add `worktrees` and `hooks` sections to `dual.config.yml`
2. **Create hook scripts**: Implement port assignment, database setup, etc. in hooks
3. **Recreate contexts**: Old contexts in global registry are not migrated automatically
4. **Update workflows**: Replace `dual <command>` with hook-based environment setup
5. **Add registry to .gitignore**: `/.dual/.local/`

See [MIGRATION.md](MIGRATION.md) for detailed migration instructions.
