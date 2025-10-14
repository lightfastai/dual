# dual Usage Guide

Complete reference for all `dual` commands and their usage.

## Table of Contents

- [Command Wrapper](#command-wrapper)
- [Initialization Commands](#initialization-commands)
  - [dual init](#dual-init)
- [Service Management](#service-management)
  - [dual service add](#dual-service-add)
- [Context Management](#context-management)
  - [dual context](#dual-context)
  - [dual context create](#dual-context-create)
- [Port Queries](#port-queries)
  - [dual port](#dual-port)
  - [dual ports](#dual-ports)
- [Utility Commands](#utility-commands)
  - [dual open](#dual-open)
  - [dual sync](#dual-sync)
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

1. Detects current context (git branch or `.dual-context` file)
2. Detects current service (from working directory or `--service` flag)
3. Calculates the port: `basePort + serviceIndex + 1`
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
- Service order matters for port calculation (first service = serviceIndex 0)
- Service names must be unique

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

## Common Workflows

### First-Time Setup

```bash
# 1. Initialize project
cd ~/Code/myproject
dual init

# 2. Add services
dual service add web --path apps/web --env-file .vercel/.env.development.local
dual service add api --path apps/api --env-file .env

# 3. Create main context
dual context create main --base-port 4100

# 4. Run your app
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
