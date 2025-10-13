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

[Features](#key-features) • [Installation](#installation) • [Quick Start](#quick-start) • [Documentation](#usage) • [Examples](#real-world-workflows)

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

## 📦 Installation

### Homebrew (macOS/Linux)

```bash
brew install lightfastai/tap/dual
```

### Direct Download

```bash
# Download latest binary
curl -sSL https://github.com/lightfastai/dual/releases/latest/download/dual-$(uname -s)-$(uname -m) \
  -o /usr/local/bin/dual
chmod +x /usr/local/bin/dual
```

### Build from Source

```bash
git clone https://github.com/lightfastai/dual.git
cd dual
go build -o dual ./cmd/dual
mv dual /usr/local/bin/
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
```

### 3. Create contexts

```bash
# Main branch gets default ports (4100 block)
dual context create main --base-port 4100

# Feature branch in worktree gets different ports (4200 block)
cd ~/Code/myproject-wt/feature-auth
dual context create feature-auth --base-port 4200
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

```bash
# Initialize project
dual init

# Add service
dual service add <name> --path <path> --env-file <file>

# Create context
dual context create [name] --base-port <port>
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

### Utility Commands

```bash
# Open service in browser
dual open web

# Sync ports to env files (fallback for non-dual workflows)
dual sync
```

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
# Config defines service order
services:
  web: { path: apps/web }
  api: { path: apps/api }
  worker: { path: apps/worker }
```

```
Context "main" (basePort: 4100):
  web    → 4101  (4100 + 0 + 1)
  api    → 4102  (4100 + 1 + 1)
  worker → 4103  (4100 + 2 + 1)

Context "feature-auth" (basePort: 4200):
  web    → 4201  (4200 + 0 + 1)
  api    → 4202  (4200 + 1 + 1)
  worker → 4203  (4200 + 2 + 1)
```

### Context Detection

Priority order:

1. **Git branch name** (primary)
   ```bash
   git branch --show-current  # → "feature-auth"
   ```

2. **`.dual-context` file** (override)
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

- [x] Core port management
- [x] Command wrapper with auto-detection
- [ ] Shell completions (bash/zsh/fish)
- [ ] `dual doctor` - health check and cleanup
- [ ] Visual dashboard (`dual ui`)
- [ ] Windows support
- [ ] Integration with tmux/terminal multiplexers

## 🙏 Credits

Built by [Lightfast](https://github.com/lightfastai) to solve our own multi-context development workflow. Open-sourced to help other developers facing the same challenges.

## 💬 Support

- 🐛 [Report a bug](https://github.com/lightfastai/dual/issues/new?template=bug_report.md)
- 💡 [Request a feature](https://github.com/lightfastai/dual/issues/new?template=feature_request.md)
- 💬 [Join discussions](https://github.com/lightfastai/dual/discussions)
- 📖 [Read the docs](https://github.com/lightfastai/dual/wiki)

---

**Made with ❤️ by developers who got tired of port conflicts.**
