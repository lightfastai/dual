# dual Examples

Real-world examples and workflows for using `dual` in different scenarios.

## Table of Contents

- [Single Service Project](#single-service-project)
- [Monorepo with Multiple Services](#monorepo-with-multiple-services)
- [Git Worktrees Workflow](#git-worktrees-workflow)
- [Multiple Clones Workflow](#multiple-clones-workflow)
- [Vercel Integration](#vercel-integration)
- [Environment Variable Management](#environment-variable-management)
- [CI/CD Integration](#cicd-integration)
- [Debug & Troubleshooting](#debug--troubleshooting)
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

## Environment Variable Management

### Scenario

You need to manage different environment configurations across multiple development contexts. Each branch/context might need different database URLs, API keys, debug flags, or service-specific configurations.

### Basic Setup

```bash
cd ~/Code/my-app
dual init
dual service add web --path apps/web --env-file .env.local
dual service add api --path apps/api --env-file .env
dual service add worker --path apps/worker --env-file .env

# Create contexts
dual context create main --base-port 4100
dual context create feature-auth --base-port 4200
```

### Workflow 1: Global Environment Overrides

Set environment variables that apply to all services in a context.

```bash
# Switch to main branch
git checkout main

# Set production-like database for main branch
dual env set DATABASE_URL "mysql://localhost/myapp_main"
dual env set LOG_LEVEL "info"
dual env set DEBUG "false"

# Check what's set
dual env show
# Output:
# Base:      .env (12 vars)
# Overrides: 3 vars
# Effective: 15 vars total
#
# Overrides for context 'main':
#   DATABASE_URL=mysql://localhost/myapp_main...
#   LOG_LEVEL=info
#   DEBUG=false
```

Now switch to feature branch with different configuration:

```bash
# Switch to feature branch
git checkout feature-auth

# Set development database for feature branch
dual env set DATABASE_URL "mysql://localhost/myapp_auth"
dual env set LOG_LEVEL "debug"
dual env set DEBUG "true"
dual env set AUTH_DEBUG "true"

# Run services - they automatically get the feature-auth overrides
cd apps/api
dual pnpm dev
# API now uses mysql://localhost/myapp_auth with debug enabled
```

### Workflow 2: Service-Specific Overrides

Different services in the same context can have different configurations.

```bash
# Main context: Production-like setup
git checkout main

# API uses MySQL
dual env set --service api DATABASE_URL "mysql://localhost/api_main"
dual env set --service api API_TIMEOUT "30s"
dual env set --service api MAX_CONNECTIONS "100"

# Web uses PostgreSQL for user data
dual env set --service web DATABASE_URL "postgres://localhost/web_main"
dual env set --service web SESSION_TIMEOUT "3600"

# Worker uses Redis for job queue
dual env set --service worker REDIS_URL "redis://localhost:6379/0"
dual env set --service worker WORKER_CONCURRENCY "5"

# View service-specific overrides
dual env show --service api
# Output:
# Base:      .env (12 vars)
# Overrides: 5 vars (including 3 service-specific for 'api')
# Effective: 17 vars total
#
# Overrides for context 'main':
#   LOG_LEVEL=info (global)
#   DEBUG=false (global)
#   DATABASE_URL=mysql://localhost/api_main (api)
#   API_TIMEOUT=30s (api)
#   MAX_CONNECTIONS=100 (api)
```

### Workflow 3: PlanetScale Branch Workflow

Common pattern: Use PlanetScale database branches that match your git branches.

```bash
# Main branch uses main database branch
git checkout main
dual env set DATABASE_URL "mysql://user:pass@aws.connect.psdb.cloud/myapp?sslaccept=strict&sslcert=/etc/ssl/cert.pem"

# Create feature branch
git checkout -b feature-payment
dual context create feature-payment --base-port 4300

# Create PlanetScale branch (outside dual)
pscale branch create myapp feature-payment

# Get connection string for PlanetScale branch
PSDB_URL=$(pscale connect myapp feature-payment --format json | jq -r .url)

# Set it in dual context
dual env set DATABASE_URL "$PSDB_URL"

# Now your feature branch uses the feature database branch
cd apps/api
dual pnpm dev
# API connects to feature-payment database branch
```

### Workflow 4: Environment Diff Before Merge

Before merging a feature branch, check what environment differences exist.

```bash
# You're on feature-auth branch, ready to merge to main
git checkout feature-auth

# Check environment differences
dual env diff main feature-auth
# Output:
# Comparing environments: main → feature-auth
#
# Changed:
#   DATABASE_URL: mysql://localhost/myapp_main → mysql://localhost/myapp_auth
#   LOG_LEVEL: info → debug
#   DEBUG: false → true
#
# Added:
#   AUTH_DEBUG=true
#   JWT_SECRET=test_secret_123
#
# Removed:
#   LEGACY_MODE=true

# Review the differences and decide:
# 1. Should JWT_SECRET be added to main?
# 2. Should LEGACY_MODE be kept in main?
# 3. Update main with necessary changes before merge
```

### Workflow 5: Environment Export for CI/CD

Export merged environment for deployment or CI/CD.

```bash
# Export as dotenv format
dual env export > .env.production
cat .env.production
# NODE_ENV=production
# DATABASE_URL=mysql://localhost/myapp_main
# LOG_LEVEL=info
# DEBUG=false
# PORT=0
# ...

# Export as JSON for processing
dual env export --format=json > env.json
cat env.json | jq '.DATABASE_URL'
# "mysql://localhost/myapp_main"

# Export as shell format for sourcing
dual env export --format=shell > env.sh
source env.sh
echo $DATABASE_URL
# mysql://localhost/myapp_main

# Export service-specific environment
dual env export --service api --format=dotenv > .env.api
```

### Workflow 6: Environment Validation

Check environment configuration before deployment.

```bash
# Validate current context
dual env check
# Output:
# ✓ Base environment file exists: .env (12 vars)
# ✓ Context detected: main
# ✓ Context has 3 environment override(s) (3 global, 0 service-specific)
#
# ✓ Environment configuration is valid

# In CI/CD pipeline
dual env check || exit 1
dual env export --format=dotenv > .env.deploy
```

### Workflow 7: Multi-Service Different Databases

Each service has its own database for development isolation.

```bash
# Main context
git checkout main

# API service: Main MySQL database
dual env set --service api DATABASE_URL "mysql://localhost/api_main"
dual env set --service api DB_POOL_SIZE "20"

# Web service: User PostgreSQL database
dual env set --service web DATABASE_URL "postgres://localhost/users_main"
dual env set --service web DB_SSL "true"

# Worker service: Separate MySQL for job state
dual env set --service worker DATABASE_URL "mysql://localhost/jobs_main"
dual env set --service worker DB_TIMEOUT "60s"

# Analytics service: MongoDB
dual env set --service analytics DATABASE_URL "mongodb://localhost/analytics_main"

# Each service now connects to its own database
cd apps/api && dual pnpm dev        # Uses MySQL
cd apps/web && dual pnpm dev        # Uses PostgreSQL
cd apps/worker && dual pnpm dev     # Uses MySQL (different DB)
cd apps/analytics && dual pnpm dev  # Uses MongoDB
```

### Workflow 8: Removing Environment Overrides

Clean up overrides when no longer needed.

```bash
# Remove global override
dual env unset DATABASE_URL
# Output:
# Removed override for DATABASE_URL in context 'main'
# Fallback to base value: DATABASE_URL=mysql://localhost/defaultdb

# Remove service-specific override
dual env unset --service api API_TIMEOUT
# Output:
# Removed override for API_TIMEOUT in service 'api' for context 'main'

# Remove multiple overrides
dual env unset DEBUG
dual env unset LOG_LEVEL
dual env unset CACHE_TTL
```

### Workflow 9: Viewing Environment Layers

Understand how environment variables are layered.

```bash
# Show all variables (base + overrides)
dual env show --values
# Base:      .env (12 vars)
# Overrides: 5 vars
# Effective: 17 vars total
#
# Base variables:
#   NODE_ENV=development
#   API_KEY=abc123
#   REDIS_URL=redis://localhost:6379
#   ...
#
# Overrides for context 'main':
#   DATABASE_URL=mysql://localhost/myapp_main
#   DEBUG=true
#   LOG_LEVEL=debug
#   API_TIMEOUT=30s (api)
#   MAX_CONNECTIONS=100 (api)

# Show only base variables
dual env show --base-only
# Base environment (.env):
# NODE_ENV
# API_KEY
# REDIS_URL
# DATABASE_HOST
# DATABASE_PORT
# ...

# Show only overrides
dual env show --overrides-only --values
# Overrides for context 'main':
# DATABASE_URL=mysql://localhost/myapp_main
# DEBUG=true
# LOG_LEVEL=debug
```

### Environment Priority

Environment variables are resolved in this priority order (highest to lowest):

1. **Runtime injection** (highest) - PORT set by dual wrapper
2. **Service-specific overrides** - Set via `dual env set --service`
3. **Context overrides** - Set via `dual env set`
4. **Base environment file** (lowest) - Loaded from dual.config.yml

Example:

```bash
# Base .env file
echo "DATABASE_URL=mysql://localhost/default" > .env
echo "LOG_LEVEL=info" >> .env

# Set context override
dual env set DATABASE_URL "mysql://localhost/main_db"

# Set service-specific override
dual env set --service api DATABASE_URL "mysql://localhost/api_db"

# Priority resolution for API service:
# 1. Service override wins: mysql://localhost/api_db
dual --service api env show --values
# DATABASE_URL=mysql://localhost/api_db (from api service override)

# Priority resolution for web service:
# 1. No service override
# 2. Context override wins: mysql://localhost/main_db
dual --service web env show --values
# DATABASE_URL=mysql://localhost/main_db (from context override)
```

### Cross-Reference

For complete syntax and options, see [USAGE.md Environment Management](USAGE.md#environment-management).

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

## Debug & Troubleshooting

### Scenario

You need to diagnose issues with port detection, service detection, context resolution, or environment variable loading. `dual` provides verbose and debug modes to help troubleshoot problems.

### Workflow 1: Using Verbose Mode

Verbose mode shows detailed operation steps.

```bash
# Check port with verbose output
dual --verbose port api
# Output:
# Loading configuration from /Users/dev/Code/myproject/dual.config.yml
# Detected 3 services: api, web, worker
# Detecting context...
# Context detected: main (from git branch)
# Loading registry from ~/.dual/registry.json
# Found context 'main' with base port 4100
# Calculating port for service 'api'...
# Service 'api' is at index 0 (alphabetically sorted)
# Port calculation: 4100 + 0 + 1 = 4101
# [dual] Context: main | Service: api | Port: 4101

# Run command with verbose output
dual --verbose pnpm dev
# Output:
# Loading configuration...
# Detecting context...
# Detecting service from /Users/dev/Code/myproject/apps/web
# Service detected: web
# Calculating port...
# [dual] Context: main | Service: web | Port: 4102
# Executing command: pnpm dev
# [command output follows...]
```

### Workflow 2: Using Debug Mode

Debug mode shows maximum detail including internal state.

```bash
# Debug port calculation
dual --debug port api
# Output:
# [DEBUG] Config file: /Users/dev/Code/myproject/dual.config.yml
# [DEBUG] Project root: /Users/dev/Code/myproject
# [DEBUG] Services loaded: 3
# [DEBUG]   - api: path=apps/api envFile=.env
# [DEBUG]   - web: path=apps/web envFile=.vercel/.env.development.local
# [DEBUG]   - worker: path=apps/worker envFile=.env
# [DEBUG] Detecting context...
# [DEBUG] Checking git branch...
# [DEBUG] Git command: git branch --show-current
# [DEBUG] Git output: main
# [DEBUG] Context detected: main (method: git)
# [DEBUG] Loading registry from ~/.dual/registry.json
# [DEBUG] Registry contains 2 projects
# [DEBUG] Found project: /Users/dev/Code/myproject
# [DEBUG] Found context 'main' with base port 4100
# [DEBUG] Calculating port for service 'api'
# [DEBUG] Services alphabetically: [api web worker]
# [DEBUG] Service 'api' index: 0
# [DEBUG] Port: 4100 + 0 + 1 = 4101
# [dual] Context: main | Service: api | Port: 4101

# Debug environment loading
dual --debug env show
# Output:
# [DEBUG] Loading base environment from .env
# [DEBUG] Base environment contains 12 variables
# [DEBUG] Loading context overrides for 'main'
# [DEBUG] Found 3 global overrides
# [DEBUG] Found 2 service-specific overrides for 'api'
# [DEBUG] Merging environment layers...
# [DEBUG] Final environment: 17 variables
# Base:      .env (12 vars)
# Overrides: 5 vars (including 2 service-specific for 'api')
# Effective: 17 vars total
```

### Workflow 3: Troubleshooting Port Conflicts

Diagnose and resolve port conflicts.

```bash
# Scenario: Port already in use
dual pnpm dev
# Output:
# Error: Port 4101 is already in use

# Step 1: Find what's using the port
lsof -i :4101
# Output:
# COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME
# node    12345 dev   21u  IPv4 0x1234      0t0  TCP *:4101 (LISTEN)

# Step 2: Debug to see which context/service is assigned this port
dual --debug ports
# Output:
# [DEBUG] Context: main
# [DEBUG] Base port: 4100
# [DEBUG] Services: [api web worker]
# Context: main (base: 4100)
# api:    4101
# web:    4102
# worker: 4103

# Solution 1: Kill the conflicting process
kill -9 12345

# Solution 2: Check if another context is using this port
dual context list --ports
# Output:
# NAME          BASE PORT  PORTS
# main          4100       api:4101, web:4102, worker:4103
# feature-old   4100       api:4101, web:4102, worker:4103  ← Duplicate!

# Fix: Delete old context or change its base port
dual context delete feature-old --force

# Solution 3: Create new context with different base port
dual context create main-v2 --base-port 4500
```

### Workflow 4: Troubleshooting Service Detection

Diagnose service detection issues.

```bash
# Scenario: Service not detected
cd ~/Code/myproject/apps/web
dual port
# Output:
# Error: could not auto-detect service from current directory
# Available services: [api web worker]
# Hint: Run this command from within a service directory or use --service flag

# Step 1: Debug service detection
dual --debug port
# Output:
# [DEBUG] Current directory: /Users/dev/Code/myproject/apps/web
# [DEBUG] Resolving symlinks...
# [DEBUG] Resolved to: /Users/dev/Code/myproject-wt/main/apps/web
# [DEBUG] Project root: /Users/dev/Code/myproject
# [DEBUG] Trying to match against services:
# [DEBUG]   - api: /Users/dev/Code/myproject/apps/api (no match)
# [DEBUG]   - web: /Users/dev/Code/myproject/apps/web (no match)
# [DEBUG]   - worker: /Users/dev/Code/myproject/apps/worker (no match)
# [DEBUG] Current path is not under project root!
# Error: could not auto-detect service

# Problem: You're in a worktree with different root path
# The worktree is at /Users/dev/Code/myproject-wt/main
# But the project root is /Users/dev/Code/myproject

# Solution 1: Use --service flag
dual --service web port
# Output: 4102

# Solution 2: cd to the actual service directory
cd /Users/dev/Code/myproject/apps/web
dual port
# Output: 4102

# Step 2: Verify service paths are correct
dual service list --paths
# Output:
# Services in dual.config.yml:
#   api     /Users/dev/Code/myproject/apps/api      .env
#   web     /Users/dev/Code/myproject/apps/web      .vercel/.env.development.local
#   worker  /Users/dev/Code/myproject/apps/worker   .env.local
```

### Workflow 5: Troubleshooting Context Detection

Diagnose context detection issues.

```bash
# Scenario: Wrong context detected
dual --debug context
# Output:
# [DEBUG] Detecting context...
# [DEBUG] Method 1: Checking git branch...
# [DEBUG] Git command: git branch --show-current
# [DEBUG] Git error: not a git repository
# [DEBUG] Method 2: Checking .dual-context file...
# [DEBUG] Reading .dual-context from current directory
# [DEBUG] File not found
# [DEBUG] Method 3: Using fallback 'default'
# Context: default
# Base Port: 4000

# Problem: Not in a git repo and no .dual-context file

# Solution 1: Initialize git
git init
git checkout -b main

# Solution 2: Create .dual-context file
echo "main" > .dual-context
dual context
# Output: Context: main

# Scenario: Context exists but not in registry
dual pnpm dev
# Output:
# Error: context "feature-x" not found in registry
# Hint: Run 'dual context create feature-x' to create this context

# Debug to confirm
dual --debug context
# Output:
# [DEBUG] Context detected: feature-x (from git branch)
# [DEBUG] Loading registry...
# [DEBUG] Registry projects: [/Users/dev/Code/myproject]
# [DEBUG] Contexts for this project: [main staging]
# Error: context "feature-x" not found

# Solution: Create the context
dual context create feature-x --base-port 4200
```

### Workflow 6: Troubleshooting Environment Issues

Diagnose environment variable problems.

```bash
# Scenario: Environment variable not applied
cd apps/api
dual pnpm dev
# App connects to wrong database

# Step 1: Check environment configuration
dual --debug env show --service api
# Output:
# [DEBUG] Loading base environment from apps/api/.env
# [DEBUG] Base file not found: apps/api/.env
# [DEBUG] Skipping base environment
# [DEBUG] Loading overrides for context 'main', service 'api'
# [DEBUG] Found 3 overrides for service 'api':
# [DEBUG]   DATABASE_URL=mysql://localhost/api_main
# [DEBUG]   API_TIMEOUT=30s
# [DEBUG]   MAX_CONNECTIONS=100
# Base:      apps/api/.env (file not found!)
# Overrides: 3 vars

# Problem: Base env file doesn't exist!

# Step 2: Check configuration
dual service list
# Output:
# api     apps/api      .env

# Solution: Create the base env file or fix the path
echo "NODE_ENV=development" > apps/api/.env

# Step 3: Validate environment
dual env check
# Output:
# ✓ Base environment file exists: apps/api/.env (1 var)
# ✓ Context detected: main
# ✓ Context has 3 environment override(s) for service 'api'
# ✓ Environment configuration is valid

# Scenario: Override not taking effect
dual env set DATABASE_URL "mysql://localhost/new_db"
dual env show --values
# DATABASE_URL doesn't show up?

# Debug it
dual --debug env show
# Output:
# [DEBUG] Context: main
# [DEBUG] Loading overrides for context 'main'
# [DEBUG] Registry contains 0 overrides for context 'main'
# [DEBUG] Previous set command may have failed

# Problem: Override wasn't saved
# Solution: Check registry file permissions
ls -la ~/.dual/registry.json
# -r--r--r--  1 dev  staff  1234 Jan 15 10:00 registry.json
# Read-only! Can't write

# Fix permissions
chmod 644 ~/.dual/registry.json

# Try again
dual env set DATABASE_URL "mysql://localhost/new_db"
# Output: Set DATABASE_URL=mysql://localhost/new_db for context 'main' (global)
```

### Workflow 7: Common Error Messages

Quick reference for common errors and solutions.

#### "failed to load config: no dual.config.yml found"

```bash
# Error
dual port
# Error: failed to load config: no dual.config.yml found in current directory or any parent directory
# Hint: Run 'dual init' to create a configuration file

# Solution
dual init
dual service add web --path . --env-file .env.local
```

#### "context not found in registry"

```bash
# Error
dual pnpm dev
# Error: context "feature-x" not found in registry
# Hint: Run 'dual context create' to create this context

# Solution
dual context create feature-x --base-port 4200
```

#### "could not auto-detect service"

```bash
# Error
dual port
# Error: could not auto-detect service from current directory
# Available services: [api web worker]
# Hint: Run this command from within a service directory or use --service flag

# Solution 1: Use --service flag
dual --service api port

# Solution 2: cd to service directory
cd apps/api
dual port
```

#### "base port already in use"

```bash
# Error
dual context create new-feature --base-port 4100
# Error: base port 4100 is already assigned to context 'main'
# Hint: Use a different base port or run 'dual context list' to see existing assignments

# Solution: Use different port or delete old context
dual context create new-feature --base-port 4200
```

### Workflow 8: Debug Environment Variable

Use debug mode to trace environment variable resolution.

```bash
# Set DUAL_DEBUG environment variable for persistent debugging
export DUAL_DEBUG=1

# Now all dual commands show debug output
dual port api
# [DEBUG] output appears automatically

# Unset to disable
unset DUAL_DEBUG
```

### Workflow 9: Finding Processes Using Ports

Find and manage processes using specific ports.

```bash
# macOS/Linux: Find process using port
lsof -i :4101

# Alternative: Use netstat
netstat -an | grep 4101

# Kill process by PID
kill -9 <PID>

# Kill all node processes (use with caution!)
pkill -9 node

# Find all dual-managed ports in use
for port in $(dual ports --json | jq -r '.ports[]'); do
  echo "Port $port:"
  lsof -i :$port
done
```

### Cross-Reference

For complete command syntax and options, see:
- [USAGE.md Debug Options](USAGE.md#debug--verbose-options)
- [USAGE.md Error Messages](USAGE.md#error-messages)

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

### Context & Service Management

#### Listing and Cleaning Up Contexts

```bash
# View all contexts for current project
dual context list
# Output:
# Contexts for /Users/dev/Code/myproject:
# NAME          BASE PORT  CREATED     CURRENT
# main          4100       2024-01-15  (current)
# feature-auth  4200       2024-01-16
# feature-old   4300       2024-01-10
# bugfix-123    4400       2024-01-12
#
# Total: 4 contexts

# View with port assignments
dual context list --ports
# Output:
# NAME          BASE PORT  CREATED     PORTS                              CURRENT
# main          4100       2024-01-15  api:4101, web:4102, worker:4103   (current)
# feature-auth  4200       2024-01-16  api:4201, web:4202, worker:4203
# feature-old   4300       2024-01-10  api:4301, web:4302, worker:4303
# bugfix-123    4400       2024-01-12  api:4401, web:4402, worker:4403

# Clean up old contexts
dual context delete feature-old
# Output:
# About to delete context: feature-old
#   Project: /Users/dev/Code/myproject
#   Base Port: 4300
#   Environment Overrides: 0
#
# Are you sure you want to delete this context? (y/N): y
# [dual] Deleted context "feature-old"

# Delete multiple contexts
dual context delete bugfix-123 --force
dual context delete temp-test --force
```

#### Managing Services

```bash
# List all configured services
dual service list
# Output:
# Services in dual.config.yml:
#   api     apps/api      .env
#   web     apps/web      .vercel/.env.development.local
#   worker  apps/worker   .env.local
#
# Total: 3 services

# View with current port assignments
dual service list --ports
# Output:
# Services (context: main, base: 4100):
#   api     apps/api      Port: 4101
#   web     apps/web      Port: 4102
#   worker  apps/worker   Port: 4103

# Remove a service (with confirmation)
dual service remove worker
# Output:
# Warning: Removing "worker" will change port assignments:
#   api: 4101 → 4101 (stays at index 0)
#   web: 4102 → 4102 (stays at index 1)
#
# Continue? (y/N): y
# [dual] Service "worker" removed from config

# Add it back
dual service add worker --path apps/worker --env-file .env.local
# Output:
# [dual] Added service "worker"
#   Path: apps/worker
#   Env File: .env.local
```

#### Port Assignment Changes

Understanding how service removal affects ports:

```bash
# Initial setup: 4 services
dual service list --ports
# Output:
# Services (context: main, base: 4100):
#   api       apps/api       Port: 4101  (index 0)
#   frontend  apps/frontend  Port: 4102  (index 1)
#   web       apps/web       Port: 4103  (index 2)
#   worker    apps/worker    Port: 4104  (index 3)

# Remove 'frontend' service
dual service remove frontend --force

# Check new port assignments
dual service list --ports
# Output:
# Services (context: main, base: 4100):
#   api     apps/api     Port: 4101  (index 0) - unchanged
#   web     apps/web     Port: 4102  (index 1) - CHANGED from 4103!
#   worker  apps/worker  Port: 4103  (index 2) - CHANGED from 4104!

# Important: Services are always sorted alphabetically
# Removing a service shifts all services that come after it alphabetically
```

### Port Conflict Detection & Resolution

#### Detecting Duplicate Base Ports

```bash
# List all contexts with ports to find duplicates
dual context list --ports --all
# Output:
# Project: /Users/dev/Code/myproject
# NAME          BASE PORT  PORTS
# main          4100       api:4101, web:4102, worker:4103
# feature-old   4100       api:4101, web:4102, worker:4103  ← Duplicate!
# feature-new   4200       api:4201, web:4202, worker:4203
#
# Project: /Users/dev/Code/otherproject
# NAME          BASE PORT  PORTS
# main          4100       app:4101  ← Also using 4100!

# Solution 1: Delete duplicate context
dual context delete feature-old --force

# Solution 2: Create context with auto-assigned port
dual context create feature-newer
# [dual] Auto-assigned base port: 4300

# Solution 3: Check all projects for conflicts
dual context list --all --json | jq -r '.[] | .contexts[] | "\(.name): \(.basePort)"' | sort -k2
```

#### Smart Port Assignment Strategy

```bash
# Use port ranges by project/environment
# Development (4000-4999)
dual context create main --base-port 4100
dual context create feature-1 --base-port 4200
dual context create feature-2 --base-port 4300

# Staging (5000-5999)
dual context create staging --base-port 5100

# QA (6000-6999)
dual context create qa --base-port 6100

# Document your port allocation
cat > PORTS.md << 'EOF'
# Port Allocation

## Development (4000-4999)
- main: 4100-4199
- feature-1: 4200-4299
- feature-2: 4300-4399

## Staging (5000-5999)
- staging: 5100-5199

## QA (6000-6999)
- qa: 6100-6199

## Services per context
With 3 services (api, web, worker), each context uses:
- Base port + 1 (api)
- Base port + 2 (web)
- Base port + 3 (worker)

Total: 3 ports per context
EOF
```

#### Finding Port Conflicts Across System

```bash
# Check if any dual ports are in use
for context in $(dual context list --json | jq -r '.contexts[].name'); do
  echo "Context: $context"
  dual --context $context ports --json | jq -r '.ports | to_entries[] | "\(.key): \(.value)"' | while read line; do
    service=$(echo $line | cut -d: -f1)
    port=$(echo $line | cut -d: -f2 | xargs)
    if lsof -i :$port >/dev/null 2>&1; then
      echo "  ⚠️  $service (port $port) is IN USE"
      lsof -i :$port | tail -n +2
    else
      echo "  ✓  $service (port $port) is available"
    fi
  done
  echo
done
```

#### Resolving Port Conflicts in Worktrees

```bash
# Scenario: Two worktrees accidentally sharing the same context
cd ~/Code/myproject-wt/main
dual context
# Context: main (base: 4100)

cd ~/Code/myproject-wt/feature-x
dual context
# Context: main (base: 4100)  ← Wrong! Should be feature-x

# The worktree is on feature-x branch but using main context
git branch --show-current
# feature-x

# Problem: Context not created for feature-x branch
# Solution: Create proper context
dual context create feature-x --base-port 4200

# Now it works correctly
dual context
# Context: feature-x (base: 4200)

# Alternative: Use .dual-context file for explicit control
echo "feature-x-worktree" > .dual-context
dual context create feature-x-worktree --base-port 4200
dual context
# Context: feature-x-worktree (base: 4200)
```

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

## Quick Troubleshooting Reference

For detailed troubleshooting workflows and debug techniques, see the [Debug & Troubleshooting](#debug--troubleshooting) section above.

### Port Already in Use

```bash
# Check what's using the port
lsof -i :4101

# Kill the process
kill -9 <PID>

# Or use different base port
dual context create main --base-port 4500
```

### Wrong Port Detected

```bash
# Check current context and port
dual --verbose port

# Override service detection
dual --service api pnpm dev
```

### Lost Context

```bash
# Recreate context
dual context create main --base-port 4100
```

---

## Next Steps

- See [USAGE.md](USAGE.md) for detailed command reference
- See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- See [README.md](README.md) for project overview
