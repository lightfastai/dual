# dual Usage Guide

Complete reference for all `dual` commands and their usage.

## Table of Contents

- [Command Wrapper](#command-wrapper)
- [Initialization Commands](#initialization-commands)
  - [dual init](#dual-init)
- [Service Management](#service-management)
  - [dual service add](#dual-service-add)
  - [dual service list](#dual-service-list)
  - [dual service remove](#dual-service-remove)
- [Context Management](#context-management)
  - [dual context](#dual-context)
  - [dual context create](#dual-context-create)
  - [dual context list](#dual-context-list)
  - [dual context delete](#dual-context-delete)
- [Environment Management](#environment-management)
  - [dual env / dual env show](#dual-env--dual-env-show)
  - [dual env set](#dual-env-set)
  - [dual env unset](#dual-env-unset)
  - [dual env export](#dual-env-export)
  - [dual env check](#dual-env-check)
  - [dual env diff](#dual-env-diff)
- [Port Queries](#port-queries)
  - [dual port](#dual-port)
  - [dual ports](#dual-ports)
- [Utility Commands](#utility-commands)
  - [dual open](#dual-open)
  - [dual sync](#dual-sync)
- [Debug & Verbose Options](#debug--verbose-options)
- [Common Workflows](#common-workflows)

---

## Command Wrapper

The primary interface for `dual`. Wraps any command and injects the `PORT` environment variable.

### Syntax

```bash
dual [--service <name>] <command> [args...]
```

### Options

- `--service <name>` - Override automatic service detection and use the specified service

### Examples

#### Basic Usage

```bash
# Run development server with auto-detected port
dual pnpm dev

# Run with npm
dual npm start

# Run with bun
dual bun run dev

# Run with yarn
dual yarn dev
```

#### Building Applications

```bash
# Next.js build with correct port
dual pnpm build

# Vite build
dual npm run build
```

#### Running Scripts

```bash
# Run database migrations
dual node scripts/migrate.js

# Run tests
dual npm test

# Run custom scripts
dual python manage.py runserver
```

#### Service Override

When working from outside a service directory or when auto-detection picks the wrong service:

```bash
# Force using the "api" service
dual --service api pnpm dev

# Force using "web" service from project root
cd ~/Code/myproject
dual --service web npm start
```

### What Happens

When you run `dual <command>`:

1. Detects current context (git branch, falls back to `.dual-context` file, then "default")
2. Detects current service (from working directory or `--service` flag)
3. Calculates the port: `basePort + serviceIndex + 1` (services sorted alphabetically)
4. Executes the command with `PORT` in the environment
5. Prints context info to stderr: `[dual] Context: main | Service: web | Port: 4101`

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
  2. Create a context with: dual context create
  3. Run your commands with: dual <command>
```

---

## Service Management

### dual service add

Add a new service to your `dual` configuration.

#### Syntax

```bash
dual service add <name> --path <path> [--env-file <file>]
```

#### Arguments

- `<name>` - Service name (used in commands and port calculations)

#### Options

- `--path <path>` - **Required.** Relative path from project root to service directory
- `--env-file <file>` - Optional. Relative path to env file for `dual sync` command

#### Examples

##### Single Service Project

```bash
# Add single service at project root
dual service add app --path . --env-file .env.local
```

##### Monorepo with Multiple Services

```bash
# Add web frontend
dual service add web --path apps/web --env-file .vercel/.env.development.local

# Add API backend
dual service add api --path apps/api --env-file .env

# Add worker service
dual service add worker --path apps/worker --env-file .env.local
```

##### Without Env File

```bash
# Add service without env file (won't be updated by `dual sync`)
dual service add docs --path docs
```

#### Output

```
[dual] Added service "web"
  Path: apps/web
  Env File: .vercel/.env.development.local
```

#### Notes

- Paths must be relative to project root (where `dual.config.yml` is located)
- Paths must exist before adding the service
- Services are sorted **alphabetically** for port calculation (not by config order!)
- Service names must be unique

---

### dual service list

List all services in your configuration.

#### Syntax

```bash
dual service list [--json] [--ports] [--paths]
```

#### Options

- `--json` - Output in JSON format for machine-readable processing
- `--ports` - Show port assignments for each service in the current context
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
  web     apps/web      .vercel/.env.development.local
  worker  apps/worker   .env.local

Total: 3 services
```

##### With Port Assignments

```bash
dual service list --ports
```

Output:
```
Services (context: main, base: 4100):
  api     apps/api      Port: 4101
  web     apps/web      Port: 4102
  worker  apps/worker   Port: 4103

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
  web     /Users/dev/Code/myproject/apps/web      .vercel/.env.development.local
  worker  /Users/dev/Code/myproject/apps/worker   .env.local

Total: 3 services
```

##### JSON Format

```bash
dual service list --json --ports
```

Output:
```json
{
  "services": [
    {
      "name": "api",
      "path": "apps/api",
      "envFile": ".env",
      "port": 4101
    },
    {
      "name": "web",
      "path": "apps/web",
      "envFile": ".vercel/.env.development.local",
      "port": 4102
    },
    {
      "name": "worker",
      "path": "apps/worker",
      "envFile": ".env.local",
      "port": 4103
    }
  ]
}
```

#### Use Cases

- **Quick Reference**: See all configured services at a glance
- **Port Planning**: Check port assignments before starting services
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
Warning: Removing "worker" will change port assignments:
  api: 4102 → 4101 (will move to index 0)
  web: 4103 → 4102 (will move to index 1)

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

**Port Assignment Changes**: Removing a service changes port assignments for all services that come after it alphabetically, since ports are calculated based on sorted order.

**Confirmation**: By default, shows which services will be affected and prompts for confirmation. Use `--force` to skip.

**File Safety**: This command only removes the service from `dual.config.yml`. It does NOT delete any files or directories.

#### Warning

After removing a service, any running instances of affected services will need to be restarted to use their new ports.

---

## Context Management

### dual context

Show information about the current context.

#### Syntax

```bash
dual context [--json]
```

#### Options

- `--json` - Output as JSON instead of human-readable format

#### Examples

##### Human-Readable Output

```bash
dual context
```

Output:
```
Context: main
Base Port: 4100
```

##### JSON Output

```bash
dual context --json
```

Output:
```json
{
  "name": "main",
  "basePort": 4100
}
```

##### With Worktree Path

```bash
cd ~/Code/myproject-wt/feature-auth
dual context
```

Output:
```
Context: feature-auth
Base Port: 4200
Path: /Users/dev/Code/myproject-wt/feature-auth
```

---

### dual context create

Create a new development context with an assigned base port.

#### Syntax

```bash
dual context create [name] [--base-port <port>]
```

#### Arguments

- `[name]` - Optional. Context name. If omitted, auto-detects from git branch or uses "default"

#### Options

- `--base-port <port>` - Optional. Base port for this context (1024-65535). If omitted, auto-assigns next available port

#### Examples

##### Auto-Detected Context Name

```bash
# On branch "main"
git checkout main
dual context create
# Creates context "main" with auto-assigned port
```

##### Explicit Context Name

```bash
# Create context with specific name
dual context create staging --base-port 5100
```

##### Auto-Assigned Port

```bash
# Let dual assign next available port
dual context create feature-auth
```

Output:
```
[dual] Auto-assigned base port: 4200
[dual] Created context "feature-auth"
  Project: /Users/dev/Code/myproject
  Base Port: 4200

Services will be assigned ports starting from: 4201
```

##### Specific Base Port

```bash
# Use custom base port
dual context create main --base-port 4100
```

#### Notes

- Base ports are typically assigned in increments of 100 (4100, 4200, 4300, etc.)
- Each context must have a unique base port
- Contexts are stored in `~/.dual/registry.json`
- Multiple worktrees of the same project can have different contexts

---

### dual context list

List all contexts for the current project or all projects.

#### Syntax

```bash
dual context list [--json] [--ports] [--all]
```

#### Options

- `--json` - Output in JSON format for machine-readable processing
- `--ports` - Show calculated port assignments for each service in each context
- `--all` - Include contexts from all projects in the registry

#### Examples

##### List Contexts for Current Project

```bash
dual context list
```

Output:
```
Contexts for /Users/dev/Code/myproject:
NAME          BASE PORT  CREATED     CURRENT
main          4100       2024-01-15  (current)
feature-auth  4200       2024-01-16
staging       5100       2024-01-10

Total: 3 contexts
```

##### List with Port Assignments

```bash
dual context list --ports
```

Output:
```
Contexts for /Users/dev/Code/myproject:
NAME          BASE PORT  CREATED     PORTS                              CURRENT
main          4100       2024-01-15  api:4101, web:4102, worker:4103   (current)
feature-auth  4200       2024-01-16  api:4201, web:4202, worker:4203
staging       5100       2024-01-10  api:5101, web:5102, worker:5103

Total: 3 contexts
```

##### List All Projects

```bash
dual context list --all
```

Output:
```
Project: /Users/dev/Code/myproject
NAME          BASE PORT  CREATED     CURRENT
main          4100       2024-01-15
feature-auth  4200       2024-01-16

Project: /Users/dev/Code/otherproject
NAME          BASE PORT  CREATED     CURRENT
main          4300       2024-01-12  (current)

Total: 3 contexts across 2 projects
```

##### JSON Format

```bash
dual context list --json --ports
```

Output:
```json
{
  "projectRoot": "/Users/dev/Code/myproject",
  "currentContext": "main",
  "contexts": [
    {
      "name": "main",
      "basePort": 4100,
      "created": "2024-01-15T10:30:00Z",
      "ports": {
        "api": 4101,
        "web": 4102,
        "worker": 4103
      }
    },
    {
      "name": "feature-auth",
      "basePort": 4200,
      "created": "2024-01-16T14:20:00Z",
      "ports": {
        "api": 4201,
        "web": 4202,
        "worker": 4203
      }
    }
  ]
}
```

#### Use Cases

- **Context Overview**: See all contexts and their port ranges at a glance
- **Port Planning**: Check port assignments across contexts to avoid conflicts
- **CI/CD Integration**: Use JSON output for automated deployment scripts
- **Multi-Project Management**: Track contexts across all projects with `--all`

---

### dual context delete

Delete a context from the registry.

#### Syntax

```bash
dual context delete <name> [--force]
```

#### Arguments

- `<name>` - Name of the context to delete

#### Options

- `--force` or `-f` - Skip confirmation prompt

#### Examples

##### Interactive Deletion

```bash
dual context delete feature-old
```

Output:
```
About to delete context: feature-old
  Project: /Users/dev/Code/myproject
  Base Port: 4300
  Environment Overrides: 5

Are you sure you want to delete this context? (y/N): y
[dual] Deleted context "feature-old"
```

##### Force Deletion (No Confirmation)

```bash
dual context delete feature-old --force
```

Output:
```
About to delete context: feature-old
  Project: /Users/dev/Code/myproject
  Base Port: 4300
[dual] Deleted context "feature-old"
```

#### Behavior

**Current Context Protection**: Cannot delete the current context. Switch to a different branch or context first.

**Environment Overrides**: Deleting a context also removes all environment variable overrides associated with it.

**Confirmation**: By default, shows context details and prompts for confirmation. Use `--force` to skip.

**Port Reclamation**: The base port is freed up and can be reused by future contexts.

#### Safety

This operation is permanent and cannot be undone. Make sure you're deleting the correct context.

#### Error Cases

```bash
# Attempting to delete current context
dual context delete main
```

Output:
```
Error: cannot delete current context "main"
Hint: Switch to a different branch or context first
```

---

## Environment Management

The environment management system allows you to set context-specific and service-specific environment variable overrides. Variables are layered in priority order:

1. **Runtime values** (highest priority) - PORT injected by dual
2. **Context-specific overrides** - Set via `dual env set`
3. **Base environment file** (lowest priority) - Loaded from `dual.config.yml`

### dual env / dual env show

Display environment variable summary and overrides for the current context.

#### Syntax

```bash
dual env [show] [--values] [--base-only] [--overrides-only] [--json] [--service <name>]
```

#### Options

- `--values` - Show actual variable values (by default, values are truncated for security)
- `--base-only` - Show only variables from base environment file
- `--overrides-only` - Show only context-specific overrides
- `--json` - Output in JSON format
- `--service <name>` - Show overrides for specific service

#### Examples

##### Basic Summary

```bash
dual env
```

Output:
```
Base:      .env (12 vars)
Overrides: 3 vars
Effective: 15 vars total

Overrides for context 'main':
  DATABASE_URL=mysql://localhost/mydb_main...
  DEBUG=true
  LOG_LEVEL=debug
```

##### Show All Values

```bash
dual env --values
```

Output:
```
Base:      .env (12 vars)
Overrides: 3 vars
Effective: 15 vars total

Overrides for context 'main':
  DATABASE_URL=mysql://localhost/mydb_main
  DEBUG=true
  LOG_LEVEL=debug
```

##### Show Only Base Variables

```bash
dual env --base-only
```

Output:
```
Base environment (.env):
NODE_ENV
API_KEY
DATABASE_HOST
DATABASE_PORT
REDIS_URL
...
```

##### Show Only Overrides

```bash
dual env --overrides-only --values
```

Output:
```
Overrides for context 'main':
DATABASE_URL=mysql://localhost/mydb_main
DEBUG=true
LOG_LEVEL=debug
```

##### Show Service-Specific Overrides

```bash
dual env --service api --values
```

Output:
```
Base:      .env (12 vars)
Overrides: 5 vars (including 2 service-specific for 'api')
Effective: 17 vars total

Overrides for context 'main':
  DATABASE_URL=mysql://localhost/mydb_main (global)
  DEBUG=true (global)
  LOG_LEVEL=debug (global)
  API_TIMEOUT=30s (api)
  API_MAX_CONNECTIONS=100 (api)
```

##### JSON Output

```bash
dual env --json
```

Output:
```json
{
  "context": "main",
  "baseFile": ".env",
  "stats": {
    "baseVars": 12,
    "overrideVars": 3,
    "totalVars": 15
  },
  "base": {
    "NODE_ENV": "development",
    "API_KEY": "...",
    "DATABASE_HOST": "localhost"
  },
  "overrides": {
    "DATABASE_URL": "mysql://localhost/mydb_main",
    "DEBUG": "true",
    "LOG_LEVEL": "debug"
  }
}
```

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

- `--service <name>` - Set override only for a specific service (instead of globally for the context)

#### Examples

##### Set Global Override

```bash
dual env set DATABASE_URL "mysql://localhost/mydb_main"
```

Output:
```
[dual] Warning: Overriding variable "DATABASE_URL" from base environment
Set DATABASE_URL=mysql://localhost/mydb_main for context 'main' (global)
Context 'main' now has 3 override(s) (3 global, 0 service-specific)
```

##### Set Service-Specific Override

```bash
dual env set --service api API_TIMEOUT "30s"
```

Output:
```
Set API_TIMEOUT=30s for service 'api' in context 'main'
Context 'main' now has 4 override(s) (3 global, 1 service-specific)
```

##### Override Multiple Variables

```bash
dual env set DEBUG "true"
dual env set LOG_LEVEL "debug"
dual env set CACHE_TTL "3600"
```

##### Service-Specific Database URLs

```bash
# Different database for each service
dual env set --service api DATABASE_URL "mysql://localhost/api_db"
dual env set --service web DATABASE_URL "mysql://localhost/web_db"
dual env set --service worker DATABASE_URL "mysql://localhost/worker_db"
```

#### Behavior

**Override Priority**: Service-specific overrides take precedence over global overrides, which take precedence over base environment file.

**Warning on Conflict**: Shows a warning if overriding a variable from the base environment file.

**Multiple Contexts**: Each context maintains its own set of overrides. Switching contexts automatically applies the correct overrides.

#### Use Cases

- **Context-Specific Databases**: Use different database names for each branch/context
- **Debug Flags**: Enable debug mode only in development contexts
- **API Endpoints**: Point to different backend URLs per context
- **Feature Flags**: Enable experimental features in specific contexts
- **Service Isolation**: Give each service its own configuration overrides

---

### dual env unset

Remove a context-specific environment variable override.

#### Syntax

```bash
dual env unset <key> [--service <name>]
```

#### Arguments

- `<key>` - Environment variable name to unset

#### Options

- `--service <name>` - Unset service-specific override (instead of global)

#### Examples

##### Unset Global Override

```bash
dual env unset DATABASE_URL
```

Output:
```
Removed override for DATABASE_URL in context 'main'
Fallback to base value: DATABASE_URL=mysql://localhost/defaultdb
```

##### Unset Service-Specific Override

```bash
dual env unset --service api API_TIMEOUT
```

Output:
```
Removed override for API_TIMEOUT in service 'api' for context 'main'
```

##### Unset Multiple Variables

```bash
dual env unset DEBUG
dual env unset LOG_LEVEL
dual env unset CACHE_TTL
```

#### Behavior

**Fallback to Base**: If the variable exists in the base environment file, shows the base value that will be used.

**Error on Missing**: Returns an error if the override doesn't exist.

**Service vs Global**: Must specify `--service` to unset service-specific overrides.

---

### dual env export

Export the complete merged environment to stdout.

#### Syntax

```bash
dual env export [--format <format>] [--service <name>]
```

#### Options

- `--format <format>` - Output format: `dotenv` (default), `json`, or `shell`
- `--service <name>` - Export environment for specific service (includes service-specific overrides)

#### Examples

##### Export as Dotenv Format (Default)

```bash
dual env export
```

Output:
```
NODE_ENV=development
API_KEY=abc123
DATABASE_URL=mysql://localhost/mydb_main
DEBUG=true
LOG_LEVEL=debug
PORT=0
...
```

##### Export as JSON

```bash
dual env export --format=json
```

Output:
```json
{
  "NODE_ENV": "development",
  "API_KEY": "abc123",
  "DATABASE_URL": "mysql://localhost/mydb_main",
  "DEBUG": "true",
  "LOG_LEVEL": "debug",
  "PORT": "0"
}
```

##### Export as Shell Format

```bash
dual env export --format=shell
```

Output:
```bash
export NODE_ENV='development'
export API_KEY='abc123'
export DATABASE_URL='mysql://localhost/mydb_main'
export DEBUG='true'
export LOG_LEVEL='debug'
export PORT='0'
```

##### Export Service-Specific Environment

```bash
dual env export --service api --format=dotenv
```

Output:
```
NODE_ENV=development
API_KEY=abc123
DATABASE_URL=mysql://localhost/api_db
API_TIMEOUT=30s
API_MAX_CONNECTIONS=100
DEBUG=true
PORT=0
...
```

##### Save to File

```bash
dual env export > .env.local
dual env export --format=json > env.json
dual env export --format=shell > env.sh
```

#### Use Cases

- **CI/CD Integration**: Export environment for deployment pipelines
- **Docker Builds**: Generate env files for container builds
- **Debugging**: Inspect complete merged environment
- **Team Sharing**: Share environment configuration (be careful with secrets!)
- **Shell Sourcing**: `source <(dual env export --format=shell)`

#### Notes

- PORT is included but set to 0 (actual port is determined at runtime)
- All layers are merged: base → overrides → runtime
- Values with spaces or special characters are properly quoted

---

### dual env check

Validate environment configuration for the current context.

#### Syntax

```bash
dual env check
```

#### Examples

##### Valid Configuration

```bash
dual env check
```

Output:
```
✓ Base environment file exists: .env (12 vars)
✓ Context detected: main
✓ Context has 3 environment override(s) (3 global, 0 service-specific)

✓ Environment configuration is valid
```

Exit code: `0`

##### Configuration Issues

```bash
dual env check
```

Output:
```
Error: Base environment file (.env) is not readable: file not found
✓ Context detected: main
✓ Context has 3 environment override(s) (3 global, 0 service-specific)

❌ Environment configuration has issues
```

Exit code: `1`

##### No Base File Configured

```bash
dual env check
```

Output:
```
ℹ No base environment file configured
✓ Context detected: main
ℹ Context has no environment overrides

✓ Environment configuration is valid
```

Exit code: `0`

#### What It Checks

- **Base Environment File**: Exists and is readable (if configured)
- **Context Detection**: Current context can be detected
- **Registry**: Context exists in registry
- **Override Counts**: Shows global and service-specific override counts

#### Use Cases

- **Pre-deployment Checks**: Validate environment before deploying
- **CI/CD Pipelines**: Ensure environment is properly configured
- **Troubleshooting**: Diagnose environment configuration issues
- **Team Onboarding**: Verify new developer setup

---

### dual env diff

Compare environment variables between two contexts.

#### Syntax

```bash
dual env diff <context1> <context2>
```

#### Arguments

- `<context1>` - First context name
- `<context2>` - Second context name to compare against

#### Examples

##### Compare Two Contexts

```bash
dual env diff main feature-auth
```

Output:
```
Comparing environments: main → feature-auth

Changed:
  DATABASE_URL: mysql://localhost/mydb_main → mysql://localhost/mydb_auth
  LOG_LEVEL: info → debug

Added:
  AUTH_DEBUG=true
  JWT_SECRET=test123

Removed:
  LEGACY_MODE=true
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

##### Typical Use Case

```bash
# Before merging feature branch, check environment differences
dual env diff main feature-new-api
```

#### Behavior

- **Changed**: Variables with different values between contexts
- **Added**: Variables only in context2
- **Removed**: Variables only in context1
- **PORT Excluded**: PORT is always excluded from comparison
- **Complete Merge**: Compares fully merged environments (base + overrides)

#### Use Cases

- **Pre-merge Review**: Check environment differences before merging branches
- **Configuration Audit**: Verify environment consistency across contexts
- **Debugging**: Identify why behavior differs between contexts
- **Documentation**: Document environment differences for deployment

---

## Port Queries

### dual port

Get the port number for a service in the current context.

#### Syntax

```bash
dual port [service] [--verbose]
```

#### Arguments

- `[service]` - Optional. Service name. If omitted, auto-detects from current directory

#### Options

- `--verbose` or `-v` - Show context and service information along with port

#### Examples

##### Auto-Detect Service

```bash
# From within service directory
cd ~/Code/myproject/apps/web
dual port
```

Output:
```
4101
```

##### Specific Service

```bash
# Query any service by name
dual port api
```

Output:
```
4102
```

##### Verbose Output

```bash
dual port web --verbose
```

Output:
```
[dual] Context: main | Service: web | Port: 4101
```

#### Use Cases

##### Integration with Scripts

```bash
# Use in shell scripts
PORT=$(dual port web)
echo "Web service running on port $PORT"

# Use in other commands
curl http://localhost:$(dual port api)/health
```

##### Environment Files

```bash
# Write to env file
echo "API_URL=http://localhost:$(dual port api)" >> .env.local
```

---

### dual ports

List all service ports for the current context.

#### Syntax

```bash
dual ports [--json]
```

#### Options

- `--json` - Output as JSON

#### Examples

##### Table Format

```bash
dual ports
```

Output:
```
Context: main (base: 4100)
api:    4102
web:    4101
worker: 4103
```

##### JSON Format

```bash
dual ports --json
```

Output:
```json
{
  "context": "main",
  "basePort": 4100,
  "ports": {
    "api": 4102,
    "web": 4101,
    "worker": 4103
  }
}
```

#### Use Cases

##### Quick Reference

```bash
# See all ports at a glance
dual ports
```

##### Documentation Generation

```bash
# Generate port documentation
dual ports --json | jq -r '.ports | to_entries[] | "- \(.key): \(.value)"' >> PORTS.md
```

---

## Utility Commands

### dual open

Open a service URL in the default web browser.

#### Syntax

```bash
dual open [service]
```

#### Arguments

- `[service]` - Optional. Service name. If omitted, auto-detects from current directory

#### Examples

##### Auto-Detect Service

```bash
# From within service directory
cd ~/Code/myproject/apps/web
dual open
```

Output:
```
[dual] Opening http://localhost:4101
```

##### Specific Service

```bash
# Open any service
dual open api
```

Output:
```
[dual] Opening http://localhost:4102
```

#### Platform Support

- macOS: Uses `open`
- Linux: Uses `xdg-open` or `sensible-browser`
- Windows: Uses `rundll32`

---

### dual sync

Write PORT values to service env files.

#### Syntax

```bash
dual sync
```

#### Description

The `sync` command is a fallback mechanism for environments where the command wrapper cannot be used. It reads the current context, calculates ports for all services, and writes `PORT=<value>` to each service's configured env file.

#### Examples

##### Basic Sync

```bash
dual sync
```

Output:
```
[dual] Updated web → PORT=4101 in .vercel/.env.development.local
[dual] Updated api → PORT=4102 in .env
[dual] Skipped worker (no envFile configured)

[dual] Sync complete: 2 updated, 1 skipped
```

##### Before CI/CD

```bash
# Sync ports before running tests in CI
dual sync
npm test  # Will read PORT from env files
```

#### Behavior

- Reads existing env file if present
- Updates `PORT=` line if it exists
- Adds `PORT=` line if it doesn't exist
- Preserves all other environment variables
- Creates env file and directories if they don't exist
- Atomic writes (uses temporary file + rename)
- Skips services without `envFile` configured

#### Use Cases

1. **CI/CD Environments**: When dual isn't installed
2. **Editor Integration**: When tools need PORT in env files
3. **Teammate Setup**: Quickly sync ports without wrapper
4. **Vercel Compatibility**: Update `.vercel/.env.development.local` for local dev

#### Notes

- This is NOT the primary way to use dual
- Prefer the command wrapper (`dual <command>`) when possible
- Synced values can become stale if context changes
- Vercel's `vercel pull` may overwrite synced values

---

## Debug & Verbose Options

All `dual` commands support verbose and debug output flags for troubleshooting and development.

### Flags

#### --verbose or -v

Enable verbose output showing detailed operation steps.

```bash
dual --verbose pnpm dev
dual -v port api
```

Output example:
```
Loading configuration...
Detecting context...
Detecting service...
Calculating port...
[dual] Context: main | Service: web | Port: 4101
Executing command: pnpm dev
```

#### --debug or -d

Enable debug output showing maximum detail including internal state. Implies `--verbose`.

```bash
dual --debug pnpm dev
```

Output example:
```
Loading configuration...
Config: /Users/dev/Code/myproject
Services: 3 ([api web worker])
Detecting context...
Loading registry...
Calculating port...
Environment: 15 variables total
[dual] Context: main | Service: web | Port: 4101
[dual] Env: base=12 overrides=3 total=15
Executing command: pnpm dev
```

### Environment Variable

You can also enable debug mode via environment variable:

```bash
export DUAL_DEBUG=1
dual pnpm dev
```

This is useful for:
- Shell scripts that call dual
- CI/CD pipelines
- Persistent debugging sessions

### Examples

#### Debug Port Calculation

```bash
dual --debug port api
```

#### Verbose Service List

```bash
dual --verbose service list
```

#### Debug Context Creation

```bash
dual --debug context create feature-test
```

#### Debug Environment Loading

```bash
dual --debug env show
```

### Use Cases

- **Troubleshooting**: Diagnose configuration or detection issues
- **Development**: Debug dual itself during development
- **CI/CD**: Enable verbose output in pipelines for better logs
- **Learning**: Understand how dual makes decisions
- **Performance**: Identify slow operations

### Tips

- Use `--verbose` for general troubleshooting
- Use `--debug` when reporting bugs or developing dual
- Combine with other flags: `dual --debug --service api pnpm dev`
- Debug output goes to stderr, so commands still work: `PORT=$(dual --debug port api)`

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
dual service add web --path apps/web --env-file .vercel/.env.development.local
dual service add api --path apps/api --env-file .env

# 4. Create main context
dual context create main --base-port 4100

# 5. Run your app
cd apps/web
dual pnpm dev
```

### Creating a Feature Branch

```bash
# Using worktrees (recommended)
git worktree add ~/Code/myproject-wt/feature-x -b feature-x
cd ~/Code/myproject-wt/feature-x
dual context create feature-x

# Run both main and feature simultaneously
# Terminal 1: main branch
cd ~/Code/myproject/apps/web && dual pnpm dev

# Terminal 2: feature branch
cd ~/Code/myproject-wt/feature-x/apps/web && dual pnpm dev
```

### Querying Ports

```bash
# Check current service port
dual port

# Check specific service
dual port api

# See all ports
dual ports

# Open in browser
dual open web
```

### Switching Contexts

```bash
# Context is auto-detected from git branch
git checkout main
dual port  # Uses main's base port

git checkout feature-x
dual port  # Uses feature-x's base port
```

### Manual Context Override

```bash
# Create .dual-context file to override git detection
echo "staging" > .dual-context
dual context  # Shows "staging" context
```

### Cleaning Up

```bash
# Remove context (manual registry edit)
# Edit ~/.dual/registry.json and remove the context entry

# Or delete entire project from registry
# Remove the project key from ~/.dual/registry.json
```

---

## Environment Variables

The `dual` command wrapper injects the following environment variable:

- `PORT` - The calculated port for the current service and context

All other environment variables from your shell are preserved and passed through to the wrapped command.

---

## Exit Codes

The `dual` command wrapper preserves exit codes from wrapped commands:

```bash
# If wrapped command exits with code 1
dual pnpm build
echo $?  # Outputs: 1

# If wrapped command succeeds
dual pnpm test
echo $?  # Outputs: 0
```

---

## Error Messages

### Configuration Errors

```
Error: failed to load config: no dual.config.yml found in current directory or any parent directory
Hint: Run 'dual init' to create a configuration file
```

Solution: Run `dual init` in your project root.

### Context Errors

```
Error: context "feature-x" not found in registry
Hint: Run 'dual context create' to create this context
```

Solution: Run `dual context create feature-x`.

### Service Errors

```
Error: could not auto-detect service from current directory
Available services: [web api worker]
Hint: Run this command from within a service directory or use --service flag
```

Solution: Either `cd` into a service directory or use `dual --service <name> <command>`.

---

## Tips and Tricks

### Shell Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias d="dual"
alias dp="dual pnpm"
alias dn="dual npm"

# Usage
dp dev
dn start
```

### Directory Bookmarks

```bash
# Jump to service and run
alias web="cd ~/Code/myproject/apps/web"
alias api="cd ~/Code/myproject/apps/api"

# Usage
web && dual pnpm dev
```

### tmux/Screen Sessions

```bash
# Start multiple services in tmux panes
tmux new-session -s dev \; \
  send-keys 'cd ~/Code/myproject/apps/web && dual pnpm dev' C-m \; \
  split-window -h \; \
  send-keys 'cd ~/Code/myproject/apps/api && dual pnpm dev' C-m
```

### Port Range Planning

```bash
# Plan your port ranges
main:       4100-4199
feature-1:  4200-4299
feature-2:  4300-4399
staging:    5100-5199
```

### Quick Port Lookup

```bash
# Add function to shell
dport() {
  dual port "$@" 2>/dev/null
}

# Usage
curl http://localhost:$(dport api)/health
```

---

## Next Steps

- See [EXAMPLES.md](EXAMPLES.md) for real-world usage scenarios
- See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- See [README.md](README.md) for project overview
