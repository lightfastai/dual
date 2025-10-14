<div align="center">

# dual

**General-purpose port management for multi-context development**

[![Go Version](https://img.shields.io/github/go-mod/go-version/lightfastai/dual)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/lightfastai/dual)](https://github.com/lightfastai/dual/releases)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/lightfastai/dual/release.yml)](https://github.com/lightfastai/dual/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/lightfastai/dual)](https://goreportcard.com/report/github.com/lightfastai/dual)

[![GitHub stars](https://img.shields.io/github/stars/lightfastai/dual)](https://github.com/lightfastai/dual/stargazers)
[![GitHub issues](https://img.shields.io/github/issues/lightfastai/dual)](https://github.com/lightfastai/dual/issues)

`dual` is a CLI tool that automatically manages port assignments across different development contexts (branches, worktrees, or clones), eliminating port conflicts and configuration headaches when working on multiple features simultaneously.

[Features](#key-features) • [Installation](#installation) • [Integration](#integration-with-web-projects) • [Quick Start](#quick-start) • [Documentation](#usage) • [Examples](#real-world-workflows)

</div>

---

## 🚨 The Problem

When working on multiple features using git worktrees or multiple clones:

```bash
# Main branch
cd ~/Code/myproject
pnpm dev  # → Port 3000

# Feature branch in worktree
cd ~/Code/myproject-wt/feature-auth
pnpm dev  # → Error: Port 3000 already in use!
```

**Common pain points:**
- Manual port management across contexts
- Port conflicts when running multiple dev servers
- `vercel pull` overwrites `.env.local`, destroying custom PORT assignments
- Remembering which ports are used by which context
- Updating ports across multiple services in a monorepo

## ✨ The Solution

`dual` acts as a transparent command wrapper that auto-detects your context and service, calculates the appropriate port, and injects it seamlessly:

```bash
# Main branch
cd ~/Code/myproject/apps/web
dual pnpm dev
# [dual] Context: main | Service: web | Port: 4101

# Feature branch (different worktree)
cd ~/Code/myproject-wt/feature-auth/apps/web
dual pnpm dev
# [dual] Context: feature-auth | Service: web | Port: 4201

# No conflicts! Both run simultaneously on different ports.
```

## 🎯 Key Features

- **Zero configuration**: After initial setup, just prefix commands with `dual`
- **Auto-detection**: Detects context (git branch) and service (from directory) automatically
- **Transparent**: See exactly what command runs and what port is used
- **Universal**: Works with any project structure, package manager, or framework
- **Vercel-proof**: Never writes to `.vercel/.env.development.local`
- **Fast**: Native Go binary, instant startup
- **Portable**: Config can be committed, registry is local
- **Complete CRUD**: Full lifecycle management for services and contexts
- **Environment management**: Set and manage environment variables with service-level overrides
- **Debug & verbose logging**: Built-in debugging and verbose output modes
- **Port conflict detection**: Automatic detection of duplicate base ports and in-use warnings

## 📦 Installation

### npm/pnpm/yarn (Recommended for Web Projects)

```bash
# npm
npm install --save-dev @lightfastai/dual

# pnpm
pnpm add -D @lightfastai/dual

# yarn
yarn add -D @lightfastai/dual
```

Then use in your package.json scripts:

```json
{
  "scripts": {
    "dev": "dual pnpm dev",
    "build": "dual pnpm build",
    "start": "dual npm start"
  }
}
```

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

## 🔌 Integration with Web Projects

### Using dual in package.json Scripts

The recommended approach is to install `dual` as an npm dev dependency (see [Installation](#installation)). This provides automatic installation, version locking, and works across all platforms.

Alternatively, you can use a bash wrapper script if you prefer minimal npm dependencies or need custom setup logic.

#### Creating a Bash Wrapper

Create a wrapper script in your project's `scripts/` directory:

**scripts/dev.sh:**
```bash
#!/bin/bash
# Wrapper script to run dev server with dual port management

# Check if dual is installed
if ! command -v dual &> /dev/null; then
  echo "⚠️  dual not found. Installing..."

  # Auto-install based on platform
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS - use Homebrew
    if ! command -v brew &> /dev/null; then
      echo "❌ Homebrew not found. Please install dual manually:"
      echo "   https://github.com/lightfastai/dual#installation"
      exit 1
    fi
    brew tap lightfastai/tap
    brew install dual
  else
    # Linux/other - use go install
    if ! command -v go &> /dev/null; then
      echo "❌ Go not found. Please install dual manually:"
      echo "   https://github.com/lightfastai/dual#installation"
      exit 1
    fi
    go install github.com/lightfastai/dual/cmd/dual@latest
    echo "✅ dual installed to \$GOPATH/bin"
    echo "   Make sure \$GOPATH/bin is in your PATH"
  fi
fi

# Execute the actual command with dual, passing all arguments
exec dual pnpm dev "$@"
```

Make it executable:
```bash
chmod +x scripts/dev.sh
```

#### Using the Wrapper in package.json

Update your `package.json` to use the wrapper:

```json
{
  "scripts": {
    "dev": "./scripts/dev.sh",
    "dev:verbose": "./scripts/dev.sh --verbose",
    "build": "dual pnpm build",
    "start": "dual pnpm start"
  }
}
```

Now team members can run:
```bash
pnpm dev        # Auto-installs dual if needed, then runs with port management
pnpm build      # Builds with correct PORT
pnpm start      # Starts production server with correct PORT
```

#### Creating Service-Specific Wrappers

For monorepos with multiple services, create specific wrapper scripts:

**scripts/dev-web.sh:**
```bash
#!/bin/bash
if ! command -v dual &> /dev/null; then
  echo "❌ dual not found. Install it first: brew install lightfastai/tap/dual"
  exit 1
fi

# Change to web directory and run dev server
cd "$(dirname "$0")/../apps/web" || exit 1
exec dual pnpm dev "$@"
```

**scripts/dev-api.sh:**
```bash
#!/bin/bash
if ! command -v dual &> /dev/null; then
  echo "❌ dual not found. Install it first: brew install lightfastai/tap/dual"
  exit 1
fi

# Change to api directory and run dev server
cd "$(dirname "$0")/../apps/api" || exit 1
exec dual pnpm dev "$@"
```

**package.json (monorepo root):**
```json
{
  "scripts": {
    "dev:web": "./scripts/dev-web.sh",
    "dev:api": "./scripts/dev-api.sh",
    "dev:all": "concurrently 'pnpm dev:web' 'pnpm dev:api'"
  }
}
```

#### Advantages and Trade-offs

**Advantages:**
- ✅ Works immediately, no waiting for npm package
- ✅ No additional npm dependencies
- ✅ Auto-installation provides smooth onboarding for new team members
- ✅ Full control over wrapper behavior
- ✅ Easy to customize for specific project needs

**Trade-offs:**
- ⚠️ Requires bash/shell environment (not Windows cmd.exe)
- ⚠️ Each developer needs dual installed or script must handle installation
- ⚠️ Slightly more verbose than native npm integration
- ⚠️ Auto-installation adds setup time on first run

**When to use the bash wrapper approach:**
- Your team is comfortable with bash scripts
- You prefer minimal npm dependencies
- You need custom setup logic before running commands
- You want auto-installation logic for new team members

**When to use the npm package (recommended):**
- You want the simplest possible `package.json` integration
- You prefer zero custom scripts in your repository
- You need Windows cmd.exe support
- You want automatic version management through package.json

#### Best Practices

1. **Commit the wrapper scripts** to your repository so all team members benefit
2. **Document the setup** in your project's README
3. **Provide fallback instructions** for developers on platforms without Homebrew or Go
4. **Test the installation path** - ensure `$GOPATH/bin` is in PATH when using `go install`
5. **Use `exec`** in wrapper scripts to properly pass signals and exit codes
6. **Pass all arguments** with `"$@"` to support flags like `--verbose` or `--debug`

#### Example: Full Monorepo Setup

**scripts/ensure-dual.sh** (shared helper):
```bash
#!/bin/bash
# Shared function to ensure dual is installed

ensure_dual() {
  if ! command -v dual &> /dev/null; then
    echo "⚠️  dual not found. Installing..."
    if [[ "$OSTYPE" == "darwin"* ]] && command -v brew &> /dev/null; then
      brew tap lightfastai/tap && brew install dual
    elif command -v go &> /dev/null; then
      go install github.com/lightfastai/dual/cmd/dual@latest
    else
      echo "❌ Cannot auto-install dual."
      echo "   Install via: https://github.com/lightfastai/dual#installation"
      exit 1
    fi
  fi
}

ensure_dual
```

**scripts/dev.sh:**
```bash
#!/bin/bash
source "$(dirname "$0")/ensure-dual.sh"
exec dual pnpm dev "$@"
```

**package.json:**
```json
{
  "scripts": {
    "dev": "./scripts/dev.sh",
    "dev:web": "cd apps/web && ../scripts/dev.sh",
    "dev:api": "cd apps/api && ../scripts/dev.sh"
  }
}
```

## 🚀 Quick Start

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
dual service add web --path apps/web --env-file .vercel/.env.development.local
dual service add api --path apps/api --env-file .env

# For a single-service project
dual service add app --path . --env-file .env.local

# List services to verify
dual service list
dual service list --ports  # Show port assignments
```

### 3. Create contexts

```bash
# Main branch gets default ports (4100 block)
dual context create main --base-port 4100

# Feature branch in worktree gets different ports (4200 block)
cd ~/Code/myproject-wt/feature-auth
dual context create feature-auth --base-port 4200

# List contexts to verify
dual context list
dual context list --ports  # Show port ranges
```

### 4. Run commands with dual

```bash
# Instead of: pnpm dev
dual pnpm dev

# Instead of: npm start
dual npm start

# Works with any command
dual bun run dev
dual cargo run

# Enable verbose mode to see what's happening
dual --verbose pnpm dev

# Debug mode for troubleshooting
dual --debug pnpm dev
```

## 📖 Usage

### Command Wrapper (Primary Interface)

Prefix any command with `dual` to inject the PORT environment variable:

```bash
dual <command> [args...]
```

**Examples:**

```bash
# Run dev server with auto-detected port
dual pnpm dev

# Build with correct port
dual pnpm build

# Start production server
dual npm start

# Run custom scripts
dual node scripts/migrate.js

# Override service detection
dual --service api pnpm dev
```

### Management Commands

#### Initialization
```bash
# Initialize project
dual init
```

#### Service Management
```bash
# Add service
dual service add <name> --path <path> --env-file <file>

# List services
dual service list
dual service list --ports  # Show port assignments
dual service list --json   # JSON output

# Remove service
dual service remove <name>
```

#### Context Management
```bash
# Create context
dual context create [name] --base-port <port>

# List contexts
dual context list
dual context list --ports  # Show port ranges
dual context list --json   # JSON output

# Delete context
dual context delete <name>
```

#### Environment Management
```bash
# Show environment summary
dual env
dual env show

# Set environment variables
dual env set <key> <value>                      # Global override
dual env set --service <name> <key> <value>     # Service-specific

# Remove overrides
dual env unset <key>
dual env unset --service <name> <key>

# Export merged environment
dual env export

# Validate configuration
dual env check

# Compare environments
dual env diff <context1> <context2>
```

### Query Commands

```bash
# Get port for current service
dual port

# Get port for specific service
dual port api

# List all ports in current context
dual ports

# Show current context info
dual context
```

### Debug & Logging Options

```bash
# Verbose mode - show detailed execution info
dual --verbose pnpm dev
dual -v pnpm dev

# Debug mode - show internal debugging information
dual --debug pnpm dev
dual -d pnpm dev

# Environment variable alternative
DUAL_DEBUG=1 dual pnpm dev
```

**Verbose output example:**
```
[dual] Loading config from: /Users/dev/Code/myproject/dual.config.yml
[dual] Detected context: feature-auth
[dual] Detected service: web
[dual] Calculated port: 4201
[dual] Executing: pnpm dev
[dual] Environment: PORT=4201
```

**Debug output example:**
```
[dual:debug] Config loaded: 3 services
[dual:debug] Registry path: /Users/dev/.dual/registry.json
[dual:debug] Context detection: git branch = feature-auth
[dual:debug] Service detection: matched path apps/web -> web
[dual:debug] Port calculation: basePort=4200, serviceIndex=1, port=4201
```

### Utility Commands

```bash
# Open service in browser
dual open web

# Sync ports to env files (fallback for non-dual workflows)
dual sync
```

For detailed usage information, see [USAGE.md](USAGE.md).

## ⚙️ Configuration

### Project Config (`dual.config.yml`)

Lives at your project root and defines services:

```yaml
version: 1
services:
  web:
    path: apps/web
    envFile: .vercel/.env.development.local
  api:
    path: services/api
    envFile: .env
  worker:
    path: apps/worker
    envFile: .env.local
```

**Can be committed** to share service definitions with your team.

### Global Registry (`~/.dual/registry.json`)

Stores context-to-port mappings across all projects:

```json
{
  "projects": {
    "/Users/dev/Code/myproject": {
      "contexts": {
        "main": {
          "basePort": 4100,
          "created": "2025-10-14T10:00:00Z"
        },
        "feature-auth": {
          "path": "/Users/dev/Code/myproject-wt/feature-auth",
          "basePort": 4200,
          "created": "2025-10-14T11:30:00Z"
        }
      }
    }
  }
}
```

**Local only** (not committed).

## 🔧 How It Works

### Port Calculation

```
port = basePort + serviceIndex + 1
```

**Example:**

```yaml
# Services are sorted alphabetically (not by config order!)
services:
  web: { path: apps/web }
  api: { path: apps/api }
  worker: { path: apps/worker }
```

```
Context "main" (basePort: 4100):
  api    → 4101  (4100 + 0 + 1)  # 'api' is alphabetically first
  web    → 4102  (4100 + 1 + 1)  # 'web' is alphabetically second
  worker → 4103  (4100 + 2 + 1)  # 'worker' is alphabetically third

Context "feature-auth" (basePort: 4200):
  api    → 4201  (4200 + 0 + 1)
  web    → 4202  (4200 + 1 + 1)
  worker → 4203  (4200 + 2 + 1)
```

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
# Match: "web" ✓
```

### Port Conflict Detection

`dual` automatically detects and warns about port conflicts:

**Duplicate base ports across contexts:**
```bash
dual context create feature-b --base-port 4200
# Warning: Base port 4200 is already used by context 'feature-a'
# This will cause port conflicts. Consider using a different base port.
```

**Ports currently in use:**
```bash
dual pnpm dev
# Warning: Port 4201 is currently in use by another process
# Suggestion: Use base ports 4300-4399 (available)
```

For detailed architecture information, see [ARCHITECTURE.md](ARCHITECTURE.md).

## 💼 Real-World Workflows

### Monorepo with Worktrees

```bash
# Setup once
cd ~/Code/myproject
dual init
dual service add web --path apps/web --env-file .vercel/.env.development.local
dual service add api --path apps/api --env-file .env
dual context create main --base-port 4100

# Create feature branch worktree
git worktree add ~/Code/myproject-wt/feature-auth -b feature-auth
cd ~/Code/myproject-wt/feature-auth
dual context create feature-auth --base-port 4200

# Work on both simultaneously
# Terminal 1 (main)
cd ~/Code/myproject/apps/web && dual pnpm dev  # → Port 4101

# Terminal 2 (feature)
cd ~/Code/myproject-wt/feature-auth/apps/web && dual pnpm dev  # → Port 4201

# Open in browser
dual open web  # Opens http://localhost:4201 (auto-detects context)
```

### Multiple Projects

```bash
# Project A
cd ~/Code/project-a
dual init
dual service add frontend --path . --env-file .env.local
dual context create main --base-port 5100

# Project B
cd ~/Code/project-b
dual init
dual service add app --path . --env-file .env
dual context create main --base-port 6100

# No conflicts between projects!
```

### Managing Environment Variables

```bash
# Set global environment override (applies to all services)
dual env set API_URL https://api.staging.example.com

# Set service-specific override
dual env set --service api DATABASE_URL postgres://localhost/feature_db
dual env set --service web NEXT_PUBLIC_API_URL https://localhost:4201

# View merged environment for current service
dual env show

# Export environment for use in scripts
eval $(dual env export)

# Validate configuration
dual env check

# Compare environments between contexts
dual env diff main feature-auth
```

### CI/CD Integration

For CI environments where dual isn't installed, use fallback:

```bash
# Setup
dual sync  # Writes PORT to env files

# CI can use regular commands
pnpm dev  # Reads PORT from .env.local
```

Or install dual in CI for consistency:

```bash
# .github/workflows/test.yml
- name: Install dual
  run: |
    curl -sSL https://github.com/lightfastai/dual/releases/latest/download/dual-linux-amd64 \
      -o /usr/local/bin/dual
    chmod +x /usr/local/bin/dual

- name: Run tests
  run: dual pnpm test
```

For more detailed examples, see [EXAMPLES.md](EXAMPLES.md).

## 📊 Comparison

### vs Manual Port Management

```bash
# Before dual
PORT=4201 pnpm dev  # Have to remember/manage ports manually

# With dual
dual pnpm dev  # Automatic, deterministic
```

### vs Environment Files

```bash
# Before dual
echo "PORT=4201" >> .env.local  # Gets overwritten by vercel pull
vercel pull  # Lost your PORT!

# With dual
dual pnpm dev  # PORT never written to files
vercel pull  # No conflicts
```

### vs Other Tools

| Feature | dual | dotenv-cli | cross-env |
|---------|------|------------|-----------|
| Auto-detects context | ✅ | ❌ | ❌ |
| Manages port registry | ✅ | ❌ | ❌ |
| Works across worktrees | ✅ | ❌ | ❌ |
| Zero config after setup | ✅ | ❌ | ❌ |
| Universal (any command) | ✅ | ✅ | ✅ |

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
git clone https://github.com/lightfastai/dual.git
cd dual
go mod download
go build -o dual ./cmd/dual
./dual --version
```

### Running Tests

```bash
go test ./...
```

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🗺️ Roadmap

### Completed (Wave 1 & 2)
- [x] Core port management
- [x] Command wrapper with auto-detection
- [x] Debug & verbose logging modes (#43)
- [x] Complete service management (list, remove) (#37)
- [x] Complete context management (list, delete) (#36)
- [x] Environment variable management with service-level overrides (#32, #40)
- [x] Port conflict detection (#42)

### In Progress
- [ ] Shell completions (bash/zsh/fish)
- [ ] `dual doctor` - health check and cleanup

### Planned
- [ ] Visual dashboard (`dual ui`)
- [ ] Windows support
- [ ] Integration with tmux/terminal multiplexers
- [ ] Environment variable templates

## 🙏 Credits

Built by [Lightfast](https://github.com/lightfastai) to solve our own multi-context development workflow. Open-sourced to help other developers facing the same challenges.

## 💬 Support

- 🐛 [Report a bug](https://github.com/lightfastai/dual/issues/new?template=bug_report.md)
- 💡 [Request a feature](https://github.com/lightfastai/dual/issues/new?template=feature_request.md)
- 💬 [Join discussions](https://github.com/lightfastai/dual/discussions)
- 📖 [Read the docs](https://github.com/lightfastai/dual/wiki)

---

**Made with ❤️ by developers who got tired of port conflicts.**
