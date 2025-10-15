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
- [Utility Commands](#utility-commands)
  - [dual doctor](#dual-doctor)
- [Configuration Reference](#configuration-reference)
- [Project-Local Registry](#project-local-registry)
- [Debug & Verbose Options](#debug--verbose-options)
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
