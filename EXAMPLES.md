# dual Examples

Real-world examples and workflows for using `dual` in different scenarios.

## Table of Contents

- [Single Service Project](#single-service-project)
- [Monorepo with Multiple Services](#monorepo-with-multiple-services)
- [Git Worktrees Workflow](#git-worktrees-workflow)
- [Multiple Clones Workflow](#multiple-clones-workflow)
- [Vercel Integration](#vercel-integration)
- [CI/CD Integration](#cicd-integration)
- [Team Collaboration](#team-collaboration)
- [Advanced Scenarios](#advanced-scenarios)

---

## Single Service Project

### Scenario

You have a single Next.js application that you want to run on different ports for main and feature branches.

### Setup

```bash
# Initialize in project root
cd ~/Code/my-nextjs-app
dual init

# Add single service
dual service add app --path . --env-file .env.local

# Create context for main branch
git checkout main
dual context create main --base-port 3100
```

### Usage

```bash
# Work on main branch
git checkout main
dual pnpm dev
# Runs on port 3101

# Create and work on feature branch
git checkout -b feature-new-ui
dual context create feature-new-ui --base-port 3200
dual pnpm dev
# Runs on port 3201

# Both branches can run simultaneously!
```

### Expected Ports

```
main branch:        3101
feature-new-ui:     3201
```

---

## Monorepo with Multiple Services

### Scenario

You have a monorepo with a web frontend, API backend, and background worker. You want each service to have its own port, deterministically calculated.

### Project Structure

```
my-monorepo/
├── dual.config.yml
├── apps/
│   ├── web/          # Next.js frontend
│   ├── api/          # Express API
│   └── worker/       # Background jobs
└── packages/
    └── shared/
```

### Setup

```bash
cd ~/Code/my-monorepo
dual init

# Add services (will be sorted alphabetically for port calculation)
dual service add web --path apps/web --env-file apps/web/.env.local
dual service add api --path apps/api --env-file apps/api/.env
dual service add worker --path apps/worker --env-file apps/worker/.env

# Create main context
dual context create main --base-port 4100
```

### Configuration File

```yaml
# dual.config.yml
version: 1
services:
  web:
    path: apps/web
    envFile: apps/web/.env.local
  api:
    path: apps/api
    envFile: apps/api/.env
  worker:
    path: apps/worker
    envFile: apps/worker/.env
```

### Running Services

```bash
# Terminal 1: Web frontend
cd ~/Code/my-monorepo/apps/web
dual pnpm dev
# [dual] Context: main | Service: web | Port: 4101

# Terminal 2: API backend
cd ~/Code/my-monorepo/apps/api
dual pnpm dev
# [dual] Context: main | Service: api | Port: 4102

# Terminal 3: Worker
cd ~/Code/my-monorepo/apps/worker
dual pnpm dev
# [dual] Context: main | Service: worker | Port: 4103
```

### Port Calculation

```
Formula: port = basePort + serviceIndex + 1
Services are sorted alphabetically: api, web, worker

Context "main" (basePort: 4100):
  api    (index 0) → 4101  # alphabetically first
  web    (index 1) → 4102  # alphabetically second
  worker (index 2) → 4103  # alphabetically third
```

### Checking Ports

```bash
# View all ports
dual ports
# Output:
# Context: main (base: 4100)
# api:    4101
# web:    4102
# worker: 4103

# Get specific port
dual port api
# Output: 4101
```

---

## Git Worktrees Workflow

### Scenario

You're working on multiple features simultaneously using git worktrees. Each worktree needs its own set of ports to avoid conflicts.

### Initial Setup

```bash
# Main repository
cd ~/Code/my-app
dual init
dual service add web --path apps/web --env-file .vercel/.env.development.local
dual service add api --path apps/api --env-file .env
dual context create main --base-port 4100
```

### Creating Worktrees

```bash
# Create feature-1 worktree
git worktree add ~/Code/my-app-wt/feature-1 -b feature-1
cd ~/Code/my-app-wt/feature-1
dual context create feature-1 --base-port 4200

# Create feature-2 worktree
git worktree add ~/Code/my-app-wt/feature-2 -b feature-2
cd ~/Code/my-app-wt/feature-2
dual context create feature-2 --base-port 4300

# Create bugfix worktree
git worktree add ~/Code/my-app-wt/bugfix-auth -b bugfix-auth
cd ~/Code/my-app-wt/bugfix-auth
dual context create bugfix-auth --base-port 4400
```

### Directory Layout

```
~/Code/
├── my-app/                  # Main repo (main branch)
└── my-app-wt/              # Worktrees directory
    ├── feature-1/          # feature-1 branch
    ├── feature-2/          # feature-2 branch
    └── bugfix-auth/        # bugfix-auth branch
```

### Running Multiple Worktrees

```bash
# Terminal 1: Main branch - web
cd ~/Code/my-app/apps/web
dual pnpm dev
# Port: 4101

# Terminal 2: Main branch - api
cd ~/Code/my-app/apps/api
dual pnpm dev
# Port: 4102

# Terminal 3: Feature 1 - web
cd ~/Code/my-app-wt/feature-1/apps/web
dual pnpm dev
# Port: 4201

# Terminal 4: Feature 2 - web
cd ~/Code/my-app-wt/feature-2/apps/web
dual pnpm dev
# Port: 4301
```

### Port Mapping

```
Context      | Service | Port
-------------|---------|------
main         | web     | 4101
main         | api     | 4102
feature-1    | web     | 4201
feature-1    | api     | 4202
feature-2    | web     | 4301
feature-2    | api     | 4302
bugfix-auth  | web     | 4401
bugfix-auth  | api     | 4402
```

### Opening in Browser

```bash
# Open feature-1 web app
cd ~/Code/my-app-wt/feature-1/apps/web
dual open
# Opens http://localhost:4201

# Open main api docs
cd ~/Code/my-app/apps/api
dual open
# Opens http://localhost:4102
```

### Cleanup

```bash
# Remove worktree
cd ~/Code/my-app
git worktree remove feature-1

# Manually remove context from ~/.dual/registry.json if needed
```

---

## Multiple Clones Workflow

### Scenario

You prefer multiple clones instead of worktrees. Each clone should have its own ports.

### Setup

```bash
# Clone 1: Main development
cd ~/Code
git clone git@github.com:user/repo.git repo-main
cd repo-main
dual init
dual service add app --path . --env-file .env.local
dual context create main --base-port 5000

# Clone 2: Feature development
cd ~/Code
git clone git@github.com:user/repo.git repo-feature
cd repo-feature
git checkout -b feature-x
# dual.config.yml already exists from clone
dual context create feature-x --base-port 5100

# Clone 3: Hotfix
cd ~/Code
git clone git@github.com:user/repo.git repo-hotfix
cd repo-hotfix
git checkout -b hotfix-critical
dual context create hotfix-critical --base-port 5200
```

### Running Clones

```bash
# Terminal 1
cd ~/Code/repo-main
dual npm dev
# Port: 5001

# Terminal 2
cd ~/Code/repo-feature
dual npm dev
# Port: 5101

# Terminal 3
cd ~/Code/repo-hotfix
dual npm dev
# Port: 5201
```

### Advantages

- Isolated dependencies (separate node_modules)
- Cleaner git history per clone
- No worktree complexity

### Disadvantages

- More disk space usage
- Need to pull changes manually
- Config changes need to be synced

---

## Vercel Integration

### Scenario

You're using Vercel for deployment and development. `vercel pull` overwrites `.vercel/.env.development.local`, destroying manual PORT assignments. `dual` solves this by never writing to the file.

### Setup

```bash
cd ~/Code/my-vercel-app
dual init

# Add service with Vercel's env file path
dual service add web --path . --env-file .vercel/.env.development.local

# Create context
dual context create main --base-port 3100
```

### Traditional Workflow (Without dual)

```bash
# 1. Set PORT in .vercel/.env.development.local
echo "PORT=3100" >> .vercel/.env.development.local

# 2. Run dev
vercel dev  # Uses PORT=3100

# 3. Pull latest env vars
vercel pull  # Overwrites .vercel/.env.development.local
# PORT is now gone! 😱

# 4. Have to manually add PORT back
echo "PORT=3100" >> .vercel/.env.development.local
```

### With dual (Recommended)

```bash
# 1. Pull latest env vars (no PORT in file)
vercel pull

# 2. Run dev with dual
dual vercel dev
# [dual] Context: main | Service: web | Port: 3101
# dual injects PORT in environment, never writes to file

# 3. Pull again - no conflicts!
vercel pull  # Works perfectly, no PORT to overwrite
```

### Multiple Environments

```bash
# Main branch
git checkout main
dual context create main --base-port 3100
dual vercel dev  # Port 3101

# Staging branch
git checkout staging
dual context create staging --base-port 3200
dual vercel dev  # Port 3201

# Feature branches
git checkout feature-x
dual context create feature-x --base-port 3300
dual vercel dev  # Port 3301
```

### Development + Preview

```bash
# Terminal 1: Development server
dual vercel dev  # Port 3101

# Terminal 2: Production build preview
vercel build
dual vercel preview  # Would use same port concept
```

---

## CI/CD Integration

### Scenario

You want to use `dual` in CI/CD for consistent port management, or you need a fallback when dual isn't available.

### GitHub Actions

#### Option 1: Install dual

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'

      - name: Install dual
        run: |
          curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_Linux_x86_64.tar.gz" | \
            sudo tar -xzf - -C /usr/local/bin dual

      - name: Install dependencies
        run: npm ci

      - name: Setup dual context
        run: |
          dual init
          dual service add app --path . --env-file .env.test
          dual context create ci --base-port 9000

      - name: Run tests
        run: dual npm test
```

#### Option 2: Use sync fallback

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '18'

      - name: Install dependencies
        run: npm ci

      # dual.config.yml already committed in repo
      # Use sync to write PORT to env files
      - name: Sync ports locally
        run: |
          # Install dual temporarily
          curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_Linux_x86_64.tar.gz" | \
            tar -xzf - -C /tmp dual

          # Create CI context and sync
          /tmp/dual context create ci --base-port 9000
          /tmp/dual sync

      - name: Run tests
        run: npm test  # Reads PORT from .env.test
```

### GitLab CI

```yaml
# .gitlab-ci.yml
test:
  image: node:18
  before_script:
    - apt-get update && apt-get install -y curl
    - curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_Linux_x86_64.tar.gz" | tar -xzf - -C /usr/local/bin dual
    - npm ci
  script:
    - dual context create ci --base-port 9000
    - dual npm test
```

### Docker

```dockerfile
# Dockerfile.test
FROM node:18

# Install dual
RUN curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_Linux_x86_64.tar.gz" | \
    tar -xzf - -C /usr/local/bin dual

WORKDIR /app
COPY package*.json ./
RUN npm ci

COPY . .

# Setup dual
RUN dual context create docker --base-port 8000

CMD ["dual", "npm", "test"]
```

### Local Development Script

```bash
#!/bin/bash
# scripts/dev.sh

set -e

# Ensure dual context exists
if ! dual context 2>/dev/null; then
  echo "Creating dual context..."
  dual context create $(git branch --show-current)
fi

# Check if in CI
if [ "$CI" = "true" ]; then
  echo "Running in CI, syncing ports..."
  dual sync
  npm test
else
  echo "Running locally with dual..."
  dual npm test
fi
```

---

## Team Collaboration

### Scenario

Your team wants to use `dual` without everyone having to set up contexts manually.

### Shareable Configuration

**Commit to repository:**

```yaml
# dual.config.yml
version: 1
services:
  web:
    path: apps/web
    envFile: .vercel/.env.development.local
  api:
    path: apps/api
    envFile: .env
  worker:
    path: apps/worker
    envFile: .env
```

**Don't commit:**
- `~/.dual/registry.json` (local machine registry)
- `.dual-context` (local context override)

### Onboarding Script

```bash
#!/bin/bash
# scripts/setup-dual.sh

echo "Setting up dual contexts..."

# Main branch
if ! dual context 2>/dev/null | grep -q "main"; then
  echo "Creating main context..."
  git checkout main
  dual context create main --base-port 4100
fi

echo "✓ dual setup complete!"
echo ""
echo "Usage:"
echo "  cd apps/web && dual pnpm dev"
echo ""
echo "To create contexts for new branches:"
echo "  dual context create <branch-name>"
```

### Team Documentation

```markdown
# Development Setup

## Port Management with dual

We use `dual` to manage ports across different branches and services.

### First Time Setup

1. Install dual:
   ```bash
   brew tap lightfastai/tap
   brew install dual
   ```

2. Run setup script:
   ```bash
   ./scripts/setup-dual.sh
   ```

### Daily Usage

Run services with `dual`:
```bash
cd apps/web
dual pnpm dev
```

### Creating Feature Branches

```bash
git checkout -b feature-name
dual context create feature-name  # Auto-assigns port
```

### Port Ranges

- Main: 4100-4199
- Feature branches: Auto-assigned in 100-port increments

### Commands

- `dual ports` - See all ports
- `dual port <service>` - Get specific port
- `dual open <service>` - Open in browser
```

---

## Advanced Scenarios

### Custom Context Names

Sometimes git branch names are too long or don't match your workflow.

```bash
# Create custom context name
git checkout feature/really-long-branch-name-JIRA-1234
dual context create feat-1234 --base-port 5000

# Create .dual-context file to override branch detection
echo "feat-1234" > .dual-context

# Now dual uses "feat-1234" instead of branch name
dual context
# Output: Context: feat-1234
```

### Shared Development Machine

Multiple developers sharing a machine (e.g., shared dev server).

```bash
# Developer 1
cd /home/dev1/project
dual context create dev1-main --base-port 4100

# Developer 2
cd /home/dev2/project
dual context create dev2-main --base-port 4200

# No port conflicts!
```

### Nested Services

Services within services (monorepo with nested structure).

```yaml
# dual.config.yml
version: 1
services:
  web:
    path: apps/web
    envFile: apps/web/.env.local
  admin:
    path: apps/web/admin  # Nested under web
    envFile: apps/web/admin/.env.local
  api:
    path: services/api
    envFile: services/api/.env
```

Service detection uses longest match:

```bash
# In apps/web/admin
cd apps/web/admin
dual port
# Detects "admin" (longest match), not "web"
```

### Multiple Services, One Terminal

Use process managers like `concurrently`:

```bash
# package.json
{
  "scripts": {
    "dev:all": "concurrently \"dual --service web pnpm dev\" \"dual --service api pnpm dev\""
  }
}

# Run all services
pnpm dev:all
```

Or with `tmux`:

```bash
#!/bin/bash
# scripts/dev-all.sh

tmux new-session -d -s dev

# Web service
tmux send-keys -t dev 'cd apps/web && dual pnpm dev' C-m

# API service
tmux split-window -h -t dev
tmux send-keys -t dev 'cd apps/api && dual pnpm dev' C-m

# Worker service
tmux split-window -v -t dev
tmux send-keys -t dev 'cd apps/worker && dual pnpm dev' C-m

# Attach to session
tmux attach -t dev
```

### Port Forwarding

Forward ports from remote machine:

```bash
# SSH tunnel
ssh -L 4101:localhost:4101 dev-server

# On dev-server
cd ~/project/apps/web
dual pnpm dev  # Port 4101

# On local machine, visit http://localhost:4101
```

### Service-Specific URLs

Different services might need different URL paths:

```bash
# Get service port
WEB_PORT=$(dual port web)
API_PORT=$(dual port api)

# Generate URLs
echo "WEB_URL=http://localhost:$WEB_PORT" > .env.local
echo "API_URL=http://localhost:$API_PORT/api/v1" >> .env.local
```

### Environment-Specific Base Ports

```bash
# Development contexts (4000 range)
dual context create main --base-port 4100
dual context create feature-1 --base-port 4200

# Staging contexts (5000 range)
dual context create staging --base-port 5100

# QA contexts (6000 range)
dual context create qa --base-port 6100

# Keep ranges separate for easy identification
```

---

## Troubleshooting Examples

### Port Already in Use

```bash
# Check what's using the port
lsof -i :4101

# Option 1: Kill the process
kill -9 <PID>

# Option 2: Use different base port
dual context create main --base-port 4500

# Option 3: Create new context for this branch
dual context create feature-new --base-port 4200
```

### Wrong Port Detected

```bash
# Check current context
dual context

# Check current service
cd apps/web
dual port --verbose
# [dual] Context: main | Service: web | Port: 4101

# Override service detection
dual --service api pnpm dev
```

### Lost Context

```bash
# Context deleted from registry?
dual context
# Error: context "main" not found in registry

# Recreate it
dual context create main --base-port 4100
```

---

## Next Steps

- See [USAGE.md](USAGE.md) for detailed command reference
- See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- See [README.md](README.md) for project overview
