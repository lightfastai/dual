# dual Examples

Real-world examples and workflows for using `dual` v0.3.0 with hook-based worktree lifecycle management.

## Table of Contents

- [Quick Start](#quick-start)
- [Hook System Examples](#hook-system-examples)
  - [Port Assignment Hooks](#port-assignment-hooks)
  - [Database Branch Management](#database-branch-management)
  - [Environment File Generation](#environment-file-generation)
  - [Dependency Installation](#dependency-installation)
  - [Pre-Delete Backups](#pre-delete-backups)
  - [Team Notifications](#team-notifications)
  - [Multi-Service Configurations](#multi-service-configurations)
- [Worktree Lifecycle Workflows](#worktree-lifecycle-workflows)
  - [Creating Feature Worktrees](#creating-feature-worktrees)
  - [Working on Multiple Features](#working-on-multiple-features)
  - [Deleting Worktrees with Cleanup](#deleting-worktrees-with-cleanup)
  - [Error Handling and Recovery](#error-handling-and-recovery)
- [Real-World Scenarios](#real-world-scenarios)
  - [Full-Stack Development with Hooks](#full-stack-development-with-hooks)
  - [Microservices with Custom Ports](#microservices-with-custom-ports)
  - [Database Per Feature Branch](#database-per-feature-branch)
  - [Monorepo Multi-Feature Development](#monorepo-multi-feature-development)
- [Team Collaboration](#team-collaboration)
  - [Shareable Configuration](#shareable-configuration)
  - [Onboarding New Developers](#onboarding-new-developers)
  - [CI/CD Integration](#cicd-integration)
- [Advanced Patterns](#advanced-patterns)
  - [Sequential Port Assignment](#sequential-port-assignment)
  - [Configuration-Based Ports](#configuration-based-ports)
  - [Conditional Hook Execution](#conditional-hook-execution)
  - [Hook Output Parsing](#hook-output-parsing)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

Get started with `dual` in a new project.

### First-Time Setup

```bash
# 1. Install dual
brew tap lightfastai/tap
brew install dual

# 2. Navigate to your project
cd ~/Code/myproject

# 3. Initialize dual
dual init

# 4. Add your services
dual service add web --path apps/web --env-file .env.local
dual service add api --path apps/api --env-file .env

# 5. Configure worktrees in dual.config.yml
cat >> dual.config.yml <<EOF

worktrees:
  path: ../worktrees
  naming: "{branch}"
EOF

# 6. Create hooks directory
mkdir -p .dual/hooks

# 7. Create a simple environment setup hook
cat > .dual/hooks/setup-environment.sh <<'EOF'
#!/bin/bash
set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate port from context hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Write .env file
cat > "$DUAL_CONTEXT_PATH/.env.local" <<ENVFILE
PORT=$PORT
NODE_ENV=development
CONTEXT_NAME=$DUAL_CONTEXT_NAME
ENVFILE

echo "✓ Environment configured (PORT=$PORT)"
EOF

chmod +x .dual/hooks/setup-environment.sh

# 8. Add hook to config
cat >> dual.config.yml <<EOF

hooks:
  postWorktreeCreate:
    - setup-environment.sh
EOF

# 9. Verify setup
dual doctor

# 10. Add registry to .gitignore
echo "/.dual/.local/" >> .gitignore
```

### Create Your First Worktree

```bash
# Create a worktree for a new feature
dual create feature-auth

# Output:
# [dual] Creating worktree for: feature-auth
# [dual] Worktree path: /Users/dev/Code/worktrees/feature-auth
# [dual] Creating git worktree...
# [dual] Registering context in registry...
# [dual] Executing postWorktreeCreate hooks...
#
# Running hook: setup-environment.sh
# Setting up environment for: feature-auth
# ✓ Environment configured (PORT=4237)
#
# [dual] Successfully created worktree: feature-auth
#   Path: /Users/dev/Code/worktrees/feature-auth

# Switch to the worktree
cd ../worktrees/feature-auth

# Start working - environment is already configured!
cd apps/web
npm run dev  # Uses PORT=4237
```

---

## Hook System Examples

The hook system is the core of dual's automation capabilities. These examples show common patterns for implementing custom logic.

### Port Assignment Hooks

#### Hash-Based Port Assignment

Assigns ports deterministically based on context name hash. Same context always gets the same port.

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
API_URL=http://localhost:$PORT
NODE_ENV=development
EOF

echo "✓ Assigned port: $PORT"
```

**Configuration:**

```yaml
hooks:
  postWorktreeCreate:
    - setup-environment.sh
```

**Usage:**

```bash
dual create feature-x
# Setting up environment for: feature-x
# ✓ Assigned port: 4237

dual create feature-y
# Setting up environment for: feature-y
# ✓ Assigned port: 4819
```

#### Multi-Service Port Assignment

Assign different ports to each service in a monorepo.

```bash
#!/bin/bash
# .dual/hooks/setup-multi-service-ports.sh

set -e

echo "Setting up multi-service ports for: $DUAL_CONTEXT_NAME"

# Calculate base port from context hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

# Assign sequential ports
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

echo "✓ Assigned ports:"
echo "  web:    $WEB_PORT"
echo "  api:    $API_PORT"
echo "  worker: $WORKER_PORT"
```

**Result:**

```bash
dual create feature-auth
# Setting up multi-service ports for: feature-auth
# ✓ Assigned ports:
#   web:    4241
#   api:    4242
#   worker: 4243
```

---

### Database Branch Management

#### PlanetScale Database Branches

Create and delete isolated database branches for each worktree.

```bash
#!/bin/bash
# .dual/hooks/create-database-branch.sh

set -e

echo "Creating PlanetScale database branch for: $DUAL_CONTEXT_NAME"

# Create branch from main
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main --wait

# Get connection string
CONNECTION_URL=$(pscale connect myapp "$DUAL_CONTEXT_NAME" --format url)

# Write to .env file
echo "DATABASE_URL=$CONNECTION_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "✓ Database branch created: $DUAL_CONTEXT_NAME"
```

```bash
#!/bin/bash
# .dual/hooks/cleanup-database.sh

set -e

echo "Deleting PlanetScale database branch: $DUAL_CONTEXT_NAME"

# Delete the branch
pscale branch delete myapp "$DUAL_CONTEXT_NAME" --force

echo "✓ Database branch deleted: $DUAL_CONTEXT_NAME"
```

**Configuration:**

```yaml
hooks:
  postWorktreeCreate:
    - create-database-branch.sh
  preWorktreeDelete:
    - cleanup-database.sh
```

**Usage:**

```bash
dual create feature-users
# Creating PlanetScale database branch for: feature-users
# ✓ Database branch created: feature-users

# Work on feature...

dual delete feature-users
# Deleting PlanetScale database branch: feature-users
# ✓ Database branch deleted: feature-users
```

#### Supabase Database Branches

Similar pattern for Supabase:

```bash
#!/bin/bash
# .dual/hooks/create-supabase-branch.sh

set -e

echo "Creating Supabase branch for: $DUAL_CONTEXT_NAME"

# Create Supabase branch
supabase branches create "$DUAL_CONTEXT_NAME"

# Get connection string
DB_URL=$(supabase branches show "$DUAL_CONTEXT_NAME" --format json | jq -r .database_url)

# Write to .env file
echo "DATABASE_URL=$DB_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "✓ Supabase branch created: $DUAL_CONTEXT_NAME"
```

#### Local PostgreSQL Databases

Create local databases per context:

```bash
#!/bin/bash
# .dual/hooks/create-local-database.sh

set -e

DB_NAME="myapp_${DUAL_CONTEXT_NAME//-/_}"  # Replace dashes with underscores

echo "Creating local database: $DB_NAME"

# Create database
createdb "$DB_NAME" || echo "Database may already exist"

# Run migrations
cd "$DUAL_CONTEXT_PATH"
DATABASE_URL="postgresql://localhost/$DB_NAME" npm run migrate

# Write to .env file
echo "DATABASE_URL=postgresql://localhost/$DB_NAME" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "✓ Database created: $DB_NAME"
```

```bash
#!/bin/bash
# .dual/hooks/cleanup-local-database.sh

set -e

DB_NAME="myapp_${DUAL_CONTEXT_NAME//-/_}"

echo "Backing up and deleting database: $DB_NAME"

# Backup first
mkdir -p ~/backups
pg_dump "$DB_NAME" > ~/backups/"${DB_NAME}_$(date +%Y%m%d_%H%M%S).sql"

# Drop database
dropdb "$DB_NAME" --if-exists

echo "✓ Database deleted: $DB_NAME"
```

---

### Environment File Generation

#### Template-Based Environment Files

Generate .env files from templates with context-specific values:

```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh

set -e

echo "Generating environment files for: $DUAL_CONTEXT_NAME"

# Calculate port
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Use template from project root
TEMPLATE="$DUAL_PROJECT_ROOT/.env.template"

if [ ! -f "$TEMPLATE" ]; then
  echo "Warning: .env.template not found, using defaults"
  TEMPLATE="/dev/null"
fi

# Copy template and substitute variables
cat "$TEMPLATE" > "$DUAL_CONTEXT_PATH/.env.local"

# Append context-specific overrides
cat >> "$DUAL_CONTEXT_PATH/.env.local" <<EOF

# Context-specific overrides (generated by dual)
CONTEXT_NAME=$DUAL_CONTEXT_NAME
PORT=$PORT
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
API_URL=http://localhost:$PORT
NODE_ENV=development
EOF

echo "✓ Environment configured (PORT=$PORT)"
```

#### Service-Specific Environment Files

Generate different .env files for each service:

```bash
#!/bin/bash
# .dual/hooks/setup-service-environments.sh

set -e

echo "Setting up service-specific environments for: $DUAL_CONTEXT_NAME"

# Calculate base port
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

WEB_PORT=$((CONTEXT_BASE + 1))
API_PORT=$((CONTEXT_BASE + 2))

# Web service
cat > "$DUAL_CONTEXT_PATH/apps/web/.env.local" <<EOF
# Web service environment
PORT=$WEB_PORT
NEXT_PUBLIC_API_URL=http://localhost:$API_PORT
NEXT_PUBLIC_CONTEXT=$DUAL_CONTEXT_NAME
NODE_ENV=development
EOF

# API service
cat > "$DUAL_CONTEXT_PATH/apps/api/.env" <<EOF
# API service environment
PORT=$API_PORT
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
JWT_SECRET=dev_secret_$DUAL_CONTEXT_NAME
CORS_ORIGIN=http://localhost:$WEB_PORT
NODE_ENV=development
EOF

echo "✓ Service environments configured"
echo "  web: $WEB_PORT"
echo "  api: $API_PORT"
```

---

### Dependency Installation

#### Auto-Install Dependencies

Automatically install dependencies when creating worktrees:

```bash
#!/bin/bash
# .dual/hooks/install-dependencies.sh

set -e

echo "Installing dependencies for: $DUAL_CONTEXT_NAME"

cd "$DUAL_CONTEXT_PATH"

# Install npm dependencies (if package.json exists)
if [ -f "package.json" ]; then
  echo "Installing npm dependencies..."
  if command -v pnpm &> /dev/null; then
    pnpm install
  elif command -v yarn &> /dev/null; then
    yarn install
  else
    npm install
  fi
  echo "✓ npm dependencies installed"
fi

# Install Python dependencies (if requirements.txt exists)
if [ -f "requirements.txt" ]; then
  echo "Installing Python dependencies..."
  pip install -r requirements.txt
  echo "✓ Python dependencies installed"
fi

# Install Ruby dependencies (if Gemfile exists)
if [ -f "Gemfile" ]; then
  echo "Installing Ruby dependencies..."
  bundle install
  echo "✓ Ruby dependencies installed"
fi

echo "✓ All dependencies installed"
```

#### Monorepo Dependency Installation

Install dependencies for specific workspaces in a monorepo:

```bash
#!/bin/bash
# .dual/hooks/install-monorepo-dependencies.sh

set -e

echo "Installing monorepo dependencies for: $DUAL_CONTEXT_NAME"

cd "$DUAL_CONTEXT_PATH"

# Install root dependencies
echo "Installing root dependencies..."
pnpm install --frozen-lockfile

# Build shared packages
echo "Building shared packages..."
pnpm --filter "./packages/*" build

# Install service dependencies
echo "Installing service dependencies..."
pnpm --filter "./apps/web" install
pnpm --filter "./apps/api" install

echo "✓ Monorepo dependencies installed"
```

---

### Pre-Delete Backups

#### Database Backup Before Deletion

Backup database before deleting worktree:

```bash
#!/bin/bash
# .dual/hooks/backup-database.sh

set -e

BACKUP_DIR="$HOME/backups/dual"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DB_NAME="myapp_${DUAL_CONTEXT_NAME//-/_}"
BACKUP_FILE="$BACKUP_DIR/${DUAL_CONTEXT_NAME}_${TIMESTAMP}.sql"

echo "Backing up database: $DB_NAME"

# Backup database
pg_dump "$DB_NAME" > "$BACKUP_FILE" || {
  echo "Warning: Database backup failed (database may not exist)"
  exit 0  # Don't fail the deletion
}

echo "✓ Database backed up to: $BACKUP_FILE"
```

#### File Backup Before Deletion

Backup important files before deletion:

```bash
#!/bin/bash
# .dual/hooks/backup-files.sh

set -e

BACKUP_DIR="$HOME/backups/dual"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/${DUAL_CONTEXT_NAME}_${TIMESTAMP}.tar.gz"

echo "Backing up worktree files: $DUAL_CONTEXT_NAME"

# Backup specific directories/files
cd "$DUAL_CONTEXT_PATH"
tar -czf "$BACKUP_FILE" \
  .env.local \
  uploads/ \
  data/ \
  logs/ \
  2>/dev/null || echo "Some files may not exist"

echo "✓ Files backed up to: $BACKUP_FILE"
```

---

### Team Notifications

#### Slack Notifications

Notify team when worktrees are created or deleted:

```bash
#!/bin/bash
# .dual/hooks/notify-slack.sh

set -e

SLACK_WEBHOOK_URL="${SLACK_WEBHOOK_URL:-https://hooks.slack.com/services/YOUR/WEBHOOK/URL}"
USER=$(whoami)
HOSTNAME=$(hostname)

case "$DUAL_EVENT" in
  postWorktreeCreate)
    MESSAGE="🚀 *Worktree Created*\nContext: \`$DUAL_CONTEXT_NAME\`\nBy: $USER@$HOSTNAME"
    ;;
  postWorktreeDelete)
    MESSAGE="🗑️ *Worktree Deleted*\nContext: \`$DUAL_CONTEXT_NAME\`\nBy: $USER@$HOSTNAME"
    ;;
  *)
    exit 0
    ;;
esac

# Send to Slack
curl -X POST "$SLACK_WEBHOOK_URL" \
  -H 'Content-Type: application/json' \
  -d "{\"text\":\"$MESSAGE\"}" \
  --silent \
  --output /dev/null

echo "✓ Team notified via Slack"
```

#### Discord Notifications

```bash
#!/bin/bash
# .dual/hooks/notify-discord.sh

set -e

DISCORD_WEBHOOK_URL="${DISCORD_WEBHOOK_URL:-https://discord.com/api/webhooks/YOUR/WEBHOOK}"
USER=$(whoami)

case "$DUAL_EVENT" in
  postWorktreeCreate)
    CONTENT="🚀 **Worktree Created**: \`$DUAL_CONTEXT_NAME\` by $USER"
    ;;
  postWorktreeDelete)
    CONTENT="🗑️ **Worktree Deleted**: \`$DUAL_CONTEXT_NAME\` by $USER"
    ;;
  *)
    exit 0
    ;;
esac

# Send to Discord
curl -X POST "$DISCORD_WEBHOOK_URL" \
  -H 'Content-Type: application/json' \
  -d "{\"content\":\"$CONTENT\"}" \
  --silent \
  --output /dev/null

echo "✓ Team notified via Discord"
```

---

### Multi-Service Configurations

#### Complete Multi-Service Setup

Full example for a microservices architecture:

```bash
#!/bin/bash
# .dual/hooks/setup-microservices.sh

set -e

echo "Setting up microservices environment for: $DUAL_CONTEXT_NAME"

# Calculate base port
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

# Service ports
GATEWAY_PORT=$((CONTEXT_BASE + 1))
AUTH_PORT=$((CONTEXT_BASE + 2))
USERS_PORT=$((CONTEXT_BASE + 3))
ORDERS_PORT=$((CONTEXT_BASE + 4))
PAYMENTS_PORT=$((CONTEXT_BASE + 5))

# Gateway service
cat > "$DUAL_CONTEXT_PATH/services/gateway/.env" <<EOF
PORT=$GATEWAY_PORT
AUTH_SERVICE_URL=http://localhost:$AUTH_PORT
USERS_SERVICE_URL=http://localhost:$USERS_PORT
ORDERS_SERVICE_URL=http://localhost:$ORDERS_PORT
PAYMENTS_SERVICE_URL=http://localhost:$PAYMENTS_PORT
EOF

# Auth service
cat > "$DUAL_CONTEXT_PATH/services/auth/.env" <<EOF
PORT=$AUTH_PORT
DATABASE_URL=postgresql://localhost/auth_${DUAL_CONTEXT_NAME}
JWT_SECRET=dev_secret_$DUAL_CONTEXT_NAME
EOF

# Users service
cat > "$DUAL_CONTEXT_PATH/services/users/.env" <<EOF
PORT=$USERS_PORT
DATABASE_URL=postgresql://localhost/users_${DUAL_CONTEXT_NAME}
AUTH_SERVICE_URL=http://localhost:$AUTH_PORT
EOF

# Orders service
cat > "$DUAL_CONTEXT_PATH/services/orders/.env" <<EOF
PORT=$ORDERS_PORT
DATABASE_URL=postgresql://localhost/orders_${DUAL_CONTEXT_NAME}
USERS_SERVICE_URL=http://localhost:$USERS_PORT
PAYMENTS_SERVICE_URL=http://localhost:$PAYMENTS_PORT
EOF

# Payments service
cat > "$DUAL_CONTEXT_PATH/services/payments/.env" <<EOF
PORT=$PAYMENTS_PORT
DATABASE_URL=postgresql://localhost/payments_${DUAL_CONTEXT_NAME}
STRIPE_SECRET_KEY=sk_test_dev
EOF

echo "✓ Microservices configured:"
echo "  gateway:  $GATEWAY_PORT"
echo "  auth:     $AUTH_PORT"
echo "  users:    $USERS_PORT"
echo "  orders:   $ORDERS_PORT"
echo "  payments: $PAYMENTS_PORT"
```

**Configuration:**

```yaml
version: 1

services:
  gateway:
    path: services/gateway
    envFile: .env
  auth:
    path: services/auth
    envFile: .env
  users:
    path: services/users
    envFile: .env
  orders:
    path: services/orders
    envFile: .env
  payments:
    path: services/payments
    envFile: .env

worktrees:
  path: ../worktrees
  naming: "{branch}"

hooks:
  postWorktreeCreate:
    - setup-microservices.sh
    - install-dependencies.sh
```

---

## Worktree Lifecycle Workflows

### Creating Feature Worktrees

#### Basic Feature Branch

```bash
# Create a feature worktree
dual create feature-auth

# Output:
# [dual] Creating worktree for: feature-auth
# [dual] Worktree path: /Users/dev/Code/worktrees/feature-auth
# [dual] Creating git worktree...
# [dual] Registering context in registry...
# [dual] Executing postWorktreeCreate hooks...
#
# Running hook: setup-environment.sh
# Setting up environment for: feature-auth
# ✓ Assigned port: 4237
#
# Running hook: create-database-branch.sh
# Creating PlanetScale database branch for: feature-auth
# ✓ Database branch created: feature-auth
#
# Running hook: install-dependencies.sh
# Installing dependencies for: feature-auth
# ✓ Dependencies installed
#
# [dual] Successfully created worktree: feature-auth
#   Path: /Users/dev/Code/worktrees/feature-auth

# Switch to worktree
cd ../worktrees/feature-auth

# Everything is ready to go!
npm run dev
```

#### Feature from Specific Base Branch

```bash
# Create feature from develop instead of current branch
dual create feature-api --from develop

# Creates worktree branching from develop
```

#### Verify Worktree Creation

```bash
# List all contexts
dual context list

# Output:
# Contexts for /Users/dev/Code/myproject:
# NAME          PATH                                        CREATED
# main          (main repository)                           2025-10-01T10:00:00Z
# feature-auth  /Users/dev/Code/worktrees/feature-auth     2025-10-15T09:30:00Z
```

---

### Working on Multiple Features

#### Simultaneous Development

```bash
# Terminal 1: Main branch
cd ~/Code/myproject/apps/web
npm run dev
# Running on port 4001 (main context)

# Terminal 2: Feature 1
cd ~/Code/worktrees/feature-auth/apps/web
npm run dev
# Running on port 4237 (feature-auth context)

# Terminal 3: Feature 2
cd ~/Code/worktrees/feature-payments/apps/web
npm run dev
# Running on port 4582 (feature-payments context)

# All three run simultaneously with isolated:
# - Ports (no conflicts)
# - Databases (separate branches)
# - Dependencies (separate node_modules)
# - Configuration (.env files)
```

#### Switching Between Features

```bash
# Work on feature-auth
cd ~/Code/worktrees/feature-auth
git status
# On branch feature-auth

# Work on feature-payments
cd ~/Code/worktrees/feature-payments
git status
# On branch feature-payments

# Back to main
cd ~/Code/myproject
git status
# On branch main
```

---

### Deleting Worktrees with Cleanup

#### Interactive Deletion

```bash
dual delete feature-old

# Output:
# About to delete worktree: feature-old
#   Path: /Users/dev/Code/worktrees/feature-old
#   Created: 2025-10-10T14:30:00Z
#
# Are you sure you want to delete this worktree? (y/N): y
#
# [dual] Executing preWorktreeDelete hooks...
#
# Running hook: backup-database.sh
# Backing up database: myapp_feature_old
# ✓ Database backed up to: ~/backups/feature-old_20251015.sql
#
# Running hook: cleanup-database.sh
# Deleting PlanetScale database branch: feature-old
# ✓ Database branch deleted: feature-old
#
# [dual] Removing git worktree...
# [dual] Removing from registry...
# [dual] Executing postWorktreeDelete hooks...
#
# Running hook: notify-team.sh
# ✓ Team notified via Slack
#
# [dual] Successfully deleted worktree: feature-old
```

#### Force Deletion (Skip Confirmation)

```bash
dual delete feature-test --force

# Skips confirmation prompt, immediately executes deletion
```

#### Batch Deletion

```bash
# Delete multiple old worktrees
for context in feature-old-1 feature-old-2 feature-old-3; do
  dual delete "$context" --force
done
```

---

### Error Handling and Recovery

#### Hook Failure During Creation

```bash
dual create feature-test

# Output:
# [dual] Creating worktree for: feature-test
# [dual] Creating git worktree...
# [dual] Registering context in registry...
# [dual] Executing postWorktreeCreate hooks...
#
# Running hook: create-database-branch.sh
# Error: Failed to create database branch (PlanetScale API error)
#
# Error: hook script failed: create-database-branch.sh (exit code 1)

# Worktree was created but hooks failed
# You can:
# 1. Fix the issue and manually run the hook
# 2. Delete and recreate the worktree
# 3. Continue without the hook's functionality

# Option 2: Delete and recreate
dual delete feature-test --force
# Fix the issue (e.g., check PlanetScale credentials)
dual create feature-test
```

#### Hook Failure During Deletion

```bash
dual delete feature-old

# Output:
# [dual] Executing preWorktreeDelete hooks...
#
# Running hook: backup-database.sh
# Error: Database connection failed
#
# Error: hook script failed: backup-database.sh (exit code 1)
# Worktree deletion halted

# Worktree and registry entry remain
# You can:
# 1. Fix the issue and retry deletion
# 2. Skip the problematic hook (edit config temporarily)
# 3. Force delete without hooks (manually edit git worktree)

# Option 1: Fix and retry
# Fix database credentials
dual delete feature-old
```

#### Manual Recovery

```bash
# If worktree is deleted outside of dual
git worktree list
# Worktree path doesn't exist

# But dual still has it registered
dual context list
# Shows the deleted worktree

# Clean up manually
git worktree prune
# Then manually edit .dual/.local/registry.json to remove the context
```

---

## Real-World Scenarios

### Full-Stack Development with Hooks

Complete setup for a Next.js + Express + PostgreSQL stack:

#### Configuration

```yaml
# dual.config.yml
version: 1

services:
  web:
    path: apps/web
    envFile: .env.local
  api:
    path: apps/api
    envFile: .env

worktrees:
  path: ../worktrees
  naming: "{branch}"

hooks:
  postWorktreeCreate:
    - create-databases.sh
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - backup-database.sh
    - cleanup-databases.sh
  postWorktreeDelete:
    - notify-team.sh
```

#### Hook: Create Databases

```bash
#!/bin/bash
# .dual/hooks/create-databases.sh

set -e

DB_NAME="myapp_${DUAL_CONTEXT_NAME//-/_}"

echo "Creating databases for: $DUAL_CONTEXT_NAME"

# Create PostgreSQL database
createdb "$DB_NAME"

# Run migrations
cd "$DUAL_CONTEXT_PATH/apps/api"
DATABASE_URL="postgresql://localhost/$DB_NAME" npm run migrate

# Seed with test data
DATABASE_URL="postgresql://localhost/$DB_NAME" npm run seed

echo "✓ Databases created and seeded: $DB_NAME"
```

#### Hook: Setup Environment

```bash
#!/bin/bash
# .dual/hooks/setup-environment.sh

set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate ports
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

WEB_PORT=$((CONTEXT_BASE + 1))
API_PORT=$((CONTEXT_BASE + 2))

DB_NAME="myapp_${DUAL_CONTEXT_NAME//-/_}"

# Web service environment
cat > "$DUAL_CONTEXT_PATH/apps/web/.env.local" <<EOF
PORT=$WEB_PORT
NEXT_PUBLIC_API_URL=http://localhost:$API_PORT
NODE_ENV=development
EOF

# API service environment
cat > "$DUAL_CONTEXT_PATH/apps/api/.env" <<EOF
PORT=$API_PORT
DATABASE_URL=postgresql://localhost/$DB_NAME
JWT_SECRET=dev_secret_$DUAL_CONTEXT_NAME
CORS_ORIGIN=http://localhost:$WEB_PORT
NODE_ENV=development
EOF

echo "✓ Environment configured"
echo "  web: http://localhost:$WEB_PORT"
echo "  api: http://localhost:$API_PORT"
```

#### Hook: Install Dependencies

```bash
#!/bin/bash
# .dual/hooks/install-dependencies.sh

set -e

echo "Installing dependencies for: $DUAL_CONTEXT_NAME"

cd "$DUAL_CONTEXT_PATH"

# Install all dependencies
pnpm install

# Build shared packages
pnpm --filter "./packages/*" build

echo "✓ Dependencies installed"
```

#### Usage

```bash
# Create feature worktree
dual create feature-auth

# Output shows all hooks executing:
# Creating databases for: feature-auth
# ✓ Databases created and seeded: myapp_feature_auth
# Setting up environment for: feature-auth
# ✓ Environment configured
#   web: http://localhost:4241
#   api: http://localhost:4242
# Installing dependencies for: feature-auth
# ✓ Dependencies installed

# Switch to worktree and start development
cd ../worktrees/feature-auth

# Terminal 1: Start API
cd apps/api
npm run dev  # Uses PORT=4242, connects to myapp_feature_auth

# Terminal 2: Start web
cd apps/web
npm run dev  # Uses PORT=4241, connects to API on 4242

# Everything is configured and ready!
```

---

### Microservices with Custom Ports

Manage a microservices architecture with deterministic port assignment:

#### Hook: Microservices Setup

```bash
#!/bin/bash
# .dual/hooks/setup-microservices.sh

set -e

echo "Configuring microservices for: $DUAL_CONTEXT_NAME"

# Define services and their port offsets
declare -A SERVICES=(
  ["gateway"]=1
  ["auth"]=2
  ["users"]=3
  ["products"]=4
  ["orders"]=5
  ["payments"]=6
  ["notifications"]=7
)

# Calculate base port
BASE_PORT=5000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

echo "Base port: $CONTEXT_BASE"

# Configure each service
for service in "${!SERVICES[@]}"; do
  offset=${SERVICES[$service]}
  port=$((CONTEXT_BASE + offset))

  echo "Configuring $service on port $port"

  cat > "$DUAL_CONTEXT_PATH/services/$service/.env" <<EOF
PORT=$port
SERVICE_NAME=$service
CONTEXT=$DUAL_CONTEXT_NAME
NODE_ENV=development

# Service URLs (for inter-service communication)
GATEWAY_URL=http://localhost:$((CONTEXT_BASE + 1))
AUTH_URL=http://localhost:$((CONTEXT_BASE + 2))
USERS_URL=http://localhost:$((CONTEXT_BASE + 3))
PRODUCTS_URL=http://localhost:$((CONTEXT_BASE + 4))
ORDERS_URL=http://localhost:$((CONTEXT_BASE + 5))
PAYMENTS_URL=http://localhost:$((CONTEXT_BASE + 6))
NOTIFICATIONS_URL=http://localhost:$((CONTEXT_BASE + 7))
EOF
done

echo "✓ Microservices configured on ports $CONTEXT_BASE-$((CONTEXT_BASE + 7))"
```

#### Run Script

Create a helper script to start all services:

```bash
#!/bin/bash
# run-all-services.sh

set -e

# Read base port from gateway .env
eval $(grep PORT= services/gateway/.env)
BASE_PORT=$((PORT - 1))

echo "Starting all microservices..."

# Start each service in background
for service in gateway auth users products orders payments notifications; do
  echo "Starting $service..."
  cd services/$service
  npm run dev &
  cd ../..
done

echo "All services started!"
echo "Gateway: http://localhost:$((BASE_PORT + 1))"
```

#### Usage

```bash
# Create worktree
dual create feature-checkout

# Output:
# Configuring microservices for: feature-checkout
# Base port: 5420
# Configuring gateway on port 5421
# Configuring auth on port 5422
# ...
# ✓ Microservices configured on ports 5420-5427

# Start all services
cd ../worktrees/feature-checkout
./run-all-services.sh

# All 7 services run on ports 5421-5427
```

---

### Database Per Feature Branch

Implement isolated database environments using PlanetScale:

#### Complete Hook Set

```bash
#!/bin/bash
# .dual/hooks/planetscale-create.sh

set -e

echo "Creating PlanetScale environment for: $DUAL_CONTEXT_NAME"

# Create branch
pscale branch create myapp "$DUAL_CONTEXT_NAME" --from main --wait

# Wait for branch to be ready
echo "Waiting for branch to be ready..."
sleep 5

# Get connection credentials
CREDENTIALS=$(pscale password create myapp "$DUAL_CONTEXT_NAME" "dual-$DUAL_CONTEXT_NAME" --format json)

# Extract connection details
HOST=$(echo "$CREDENTIALS" | jq -r .access_host_url)
USERNAME=$(echo "$CREDENTIALS" | jq -r .username)
PASSWORD=$(echo "$CREDENTIALS" | jq -r .plain_text)

# Build connection string
DATABASE_URL="mysql://${USERNAME}:${PASSWORD}@${HOST}/myapp?ssl={\"rejectUnauthorized\":true}"

# Write to .env
echo "DATABASE_URL=$DATABASE_URL" >> "$DUAL_CONTEXT_PATH/.env.local"

echo "✓ PlanetScale branch created: $DUAL_CONTEXT_NAME"
```

```bash
#!/bin/bash
# .dual/hooks/planetscale-backup.sh

set -e

BACKUP_DIR="$HOME/backups/planetscale"
mkdir -p "$BACKUP_DIR"

TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/${DUAL_CONTEXT_NAME}_${TIMESTAMP}.sql"

echo "Backing up PlanetScale branch: $DUAL_CONTEXT_NAME"

# Dump database
pscale database dump myapp "$DUAL_CONTEXT_NAME" --output "$BACKUP_FILE"

echo "✓ Backup saved: $BACKUP_FILE"
```

```bash
#!/bin/bash
# .dual/hooks/planetscale-cleanup.sh

set -e

echo "Cleaning up PlanetScale branch: $DUAL_CONTEXT_NAME"

# Delete passwords
pscale password list myapp "$DUAL_CONTEXT_NAME" --format json | \
  jq -r '.[].id' | \
  while read -r password_id; do
    pscale password delete myapp "$DUAL_CONTEXT_NAME" "$password_id" --force
  done

# Delete branch
pscale branch delete myapp "$DUAL_CONTEXT_NAME" --force

echo "✓ PlanetScale branch deleted: $DUAL_CONTEXT_NAME"
```

#### Configuration

```yaml
hooks:
  postWorktreeCreate:
    - planetscale-create.sh
    - setup-environment.sh
  preWorktreeDelete:
    - planetscale-backup.sh
    - planetscale-cleanup.sh
```

#### Usage

```bash
# Create feature with database branch
dual create feature-users

# Output:
# Creating PlanetScale environment for: feature-users
# Creating branch...
# Waiting for branch to be ready...
# ✓ PlanetScale branch created: feature-users

# Work with isolated database
cd ../worktrees/feature-users
npm run migrate
npm run seed

# Delete when done
dual delete feature-users

# Output:
# Backing up PlanetScale branch: feature-users
# ✓ Backup saved: ~/backups/planetscale/feature-users_20251015.sql
# Cleaning up PlanetScale branch: feature-users
# ✓ PlanetScale branch deleted: feature-users
```

---

### Monorepo Multi-Feature Development

Manage a complex monorepo with multiple services and shared packages:

#### Project Structure

```
myproject/
├── dual.config.yml
├── .dual/
│   └── hooks/
│       ├── setup-monorepo.sh
│       └── install-monorepo-deps.sh
├── packages/
│   ├── shared/
│   ├── ui/
│   └── utils/
└── apps/
    ├── web/
    ├── mobile/
    ├── api/
    └── admin/
```

#### Hook: Monorepo Setup

```bash
#!/bin/bash
# .dual/hooks/setup-monorepo.sh

set -e

echo "Setting up monorepo environment for: $DUAL_CONTEXT_NAME"

# Calculate base port
BASE_PORT=3000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
CONTEXT_BASE=$((BASE_PORT + 0x$CONTEXT_HASH % 100 * 10))

# App ports
WEB_PORT=$((CONTEXT_BASE + 1))
MOBILE_PORT=$((CONTEXT_BASE + 2))
API_PORT=$((CONTEXT_BASE + 3))
ADMIN_PORT=$((CONTEXT_BASE + 4))

# Database name
DB_NAME="myapp_${DUAL_CONTEXT_NAME//-/_}"

# Create databases
createdb "$DB_NAME"

# Web app
cat > "$DUAL_CONTEXT_PATH/apps/web/.env.local" <<EOF
PORT=$WEB_PORT
NEXT_PUBLIC_API_URL=http://localhost:$API_PORT
EOF

# Mobile app (Expo)
cat > "$DUAL_CONTEXT_PATH/apps/mobile/.env" <<EOF
EXPO_PUBLIC_API_URL=http://localhost:$API_PORT
EOF

# API
cat > "$DUAL_CONTEXT_PATH/apps/api/.env" <<EOF
PORT=$API_PORT
DATABASE_URL=postgresql://localhost/$DB_NAME
CORS_ORIGINS=http://localhost:$WEB_PORT,http://localhost:$ADMIN_PORT
EOF

# Admin
cat > "$DUAL_CONTEXT_PATH/apps/admin/.env.local" <<EOF
PORT=$ADMIN_PORT
NEXT_PUBLIC_API_URL=http://localhost:$API_PORT
EOF

echo "✓ Monorepo environment configured:"
echo "  web:    http://localhost:$WEB_PORT"
echo "  mobile: (Expo) → API: http://localhost:$API_PORT"
echo "  api:    http://localhost:$API_PORT"
echo "  admin:  http://localhost:$ADMIN_PORT"
```

#### Hook: Install Dependencies

```bash
#!/bin/bash
# .dual/hooks/install-monorepo-deps.sh

set -e

echo "Installing monorepo dependencies for: $DUAL_CONTEXT_NAME"

cd "$DUAL_CONTEXT_PATH"

# Install all dependencies
echo "Installing dependencies..."
pnpm install --frozen-lockfile

# Build shared packages first
echo "Building shared packages..."
pnpm --filter "./packages/*" build

# Build apps that depend on shared packages
echo "Building dependent apps..."
pnpm --filter "./apps/api" build

echo "✓ Monorepo dependencies installed and built"
```

#### Usage

```bash
# Create feature worktree
dual create feature-checkout

# Output:
# Setting up monorepo environment for: feature-checkout
# ✓ Monorepo environment configured:
#   web:    http://localhost:3751
#   mobile: (Expo) → API: http://localhost:3753
#   api:    http://localhost:3753
#   admin:  http://localhost:3754
# Installing monorepo dependencies for: feature-checkout
# Installing dependencies...
# Building shared packages...
# Building dependent apps...
# ✓ Monorepo dependencies installed and built

# Start development
cd ../worktrees/feature-checkout

# Terminal 1: API
cd apps/api && npm run dev

# Terminal 2: Web
cd apps/web && npm run dev

# Terminal 3: Admin
cd apps/admin && npm run dev

# Terminal 4: Mobile
cd apps/mobile && npm run start
```

---

## Team Collaboration

### Shareable Configuration

#### Commit Configuration to Repository

```bash
# dual.config.yml (commit this)
version: 1

services:
  web:
    path: apps/web
    envFile: .env.local
  api:
    path: apps/api
    envFile: .env

worktrees:
  path: ../worktrees
  naming: "{branch}"

hooks:
  postWorktreeCreate:
    - setup-environment.sh
    - create-database-branch.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - backup-database.sh
    - cleanup-database.sh
  postWorktreeDelete:
    - notify-team.sh
```

#### Add to .gitignore

```bash
# .gitignore
/.dual/.local/
```

#### Commit Hook Scripts

```bash
# Hooks should be committed so team can use them
git add .dual/hooks/*.sh
git add dual.config.yml
git commit -m "feat: add dual worktree lifecycle hooks"
```

#### Document Hook Requirements

```markdown
# README.md

## Development Setup

### Prerequisites

This project uses [dual](https://github.com/lightfastai/dual) for worktree lifecycle management.

Install dual:
```bash
brew tap lightfastai/tap
brew install dual
```

### Required Tools

Our hooks require the following tools:
- `pscale` - PlanetScale CLI (for database branch management)
- `jq` - JSON processor
- PostgreSQL client tools (for local database creation)

Install on macOS:
```bash
brew install planetscale/tap/pscale jq postgresql
```

### Creating a Feature Branch

```bash
# Create a new feature worktree
dual create feature-my-feature

# This will automatically:
# - Create a git worktree
# - Create a PlanetScale database branch
# - Configure environment variables
# - Install dependencies

# Switch to the worktree
cd ../worktrees/feature-my-feature

# Start development
npm run dev
```

### Cleaning Up

```bash
# Delete worktree when done
dual delete feature-my-feature

# This will automatically:
# - Backup the database
# - Delete the PlanetScale branch
# - Remove the worktree
# - Notify the team
```
```

---

### Onboarding New Developers

#### Onboarding Script

```bash
#!/bin/bash
# scripts/setup-dual.sh

set -e

echo "Setting up dual for development..."

# Check if dual is installed
if ! command -v dual &> /dev/null; then
  echo "Installing dual..."
  brew tap lightfastai/tap
  brew install dual
fi

# Check if required tools are installed
echo "Checking required tools..."

if ! command -v pscale &> /dev/null; then
  echo "Installing PlanetScale CLI..."
  brew install planetscale/tap/pscale
fi

if ! command -v jq &> /dev/null; then
  echo "Installing jq..."
  brew install jq
fi

# Verify dual configuration
echo "Verifying dual configuration..."
dual doctor

# Make sure hooks are executable
echo "Setting hook permissions..."
chmod +x .dual/hooks/*.sh

# Create main context if it doesn't exist
if ! dual context list | grep -q "main"; then
  echo "Creating main context..."
  git checkout main
  # Main context doesn't need a worktree path, just register it
fi

echo "✓ Dual setup complete!"
echo ""
echo "To create a feature worktree:"
echo "  dual create feature-my-feature"
```

#### Quick Start Guide

```markdown
# Quick Start for New Developers

## 1. Clone Repository

```bash
git clone git@github.com:company/myproject.git
cd myproject
```

## 2. Run Setup Script

```bash
./scripts/setup-dual.sh
```

This will:
- Install dual
- Install required tools (pscale, jq)
- Verify configuration
- Set hook permissions

## 3. Create Your First Feature

```bash
# Create a feature worktree
dual create feature-onboarding-test

# Switch to the worktree
cd ../worktrees/feature-onboarding-test

# Start development
npm run dev
```

## 4. Clean Up

```bash
# When done with the feature
dual delete feature-onboarding-test
```

## Troubleshooting

If you encounter issues:

```bash
# Check dual configuration
dual doctor

# Enable debug mode
dual --debug create feature-test
```

## Getting Help

- See `README.md` for full documentation
- Ask in #dev-help Slack channel
- Check [dual documentation](https://github.com/lightfastai/dual)
```

---

### CI/CD Integration

#### GitHub Actions

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:

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

      - name: Verify dual configuration
        run: dual doctor

      - name: List services
        run: dual service list

      - name: Install dependencies
        run: pnpm install

      - name: Run tests
        run: pnpm test

      - name: Build
        run: pnpm build
```

#### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - setup
  - test
  - build

variables:
  DUAL_VERSION: "latest"

.install_dual:
  before_script:
    - apt-get update && apt-get install -y curl
    - curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_Linux_x86_64.tar.gz" | tar -xzf - -C /usr/local/bin dual

verify_config:
  stage: setup
  extends: .install_dual
  script:
    - dual doctor
    - dual service list

test:
  stage: test
  script:
    - pnpm install
    - pnpm test

build:
  stage: build
  script:
    - pnpm install
    - pnpm build
```

#### Jenkins

```groovy
// Jenkinsfile
pipeline {
    agent any

    environment {
        DUAL_VERSION = 'latest'
    }

    stages {
        stage('Setup') {
            steps {
                sh '''
                    # Install dual
                    curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_Linux_x86_64.tar.gz" | \
                        tar -xzf - -C /usr/local/bin dual

                    # Verify configuration
                    dual doctor
                '''
            }
        }

        stage('Test') {
            steps {
                sh 'pnpm install'
                sh 'pnpm test'
            }
        }

        stage('Build') {
            steps {
                sh 'pnpm build'
            }
        }
    }
}
```

---

## Advanced Patterns

### Sequential Port Assignment

Assign ports sequentially based on existing contexts:

```bash
#!/bin/bash
# .dual/hooks/sequential-ports.sh

set -e

REGISTRY_FILE="$DUAL_PROJECT_ROOT/.dual/.local/registry.json"
PORT_FILE="$DUAL_PROJECT_ROOT/.dual/next-port.txt"

# Initialize if doesn't exist
if [ ! -f "$PORT_FILE" ]; then
  echo "4000" > "$PORT_FILE"
fi

# Read next available port
NEXT_PORT=$(cat "$PORT_FILE")

# Write to .env
cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
PORT=$NEXT_PORT
EOF

# Increment for next context
echo $((NEXT_PORT + 10)) > "$PORT_FILE"

echo "✓ Assigned port: $NEXT_PORT"
```

---

### Configuration-Based Ports

Define ports in a configuration file:

```bash
#!/bin/bash
# .dual/hooks/config-based-ports.sh

set -e

PORT_CONFIG="$DUAL_PROJECT_ROOT/.dual/port-config.yml"

# Read port from config (using yq)
if [ -f "$PORT_CONFIG" ]; then
  PORT=$(yq eval ".contexts.\"$DUAL_CONTEXT_NAME\".port // 4000" "$PORT_CONFIG")
else
  PORT=4000
fi

# Write to .env
echo "PORT=$PORT" > "$DUAL_CONTEXT_PATH/.env.local"

echo "✓ Assigned port: $PORT (from config)"
```

Port configuration file:

```yaml
# .dual/port-config.yml
contexts:
  main:
    port: 4000
  staging:
    port: 4100
  feature-auth:
    port: 4200
  feature-payments:
    port: 4300
```

---

### Conditional Hook Execution

Execute hooks conditionally based on context:

```bash
#!/bin/bash
# .dual/hooks/conditional-setup.sh

set -e

echo "Conditional setup for: $DUAL_CONTEXT_NAME"

# Production-like setup for main and staging
if [[ "$DUAL_CONTEXT_NAME" == "main" || "$DUAL_CONTEXT_NAME" == "staging" ]]; then
  echo "Setting up production-like environment..."

  cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
NODE_ENV=production
LOG_LEVEL=info
DEBUG=false
EOF

# Development setup for feature branches
elif [[ "$DUAL_CONTEXT_NAME" == feature-* ]]; then
  echo "Setting up development environment..."

  cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
NODE_ENV=development
LOG_LEVEL=debug
DEBUG=true
EOF

# Test setup for test contexts
elif [[ "$DUAL_CONTEXT_NAME" == test-* ]]; then
  echo "Setting up test environment..."

  cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
NODE_ENV=test
LOG_LEVEL=error
DEBUG=false
EOF

else
  echo "Unknown context type, using defaults..."
fi

echo "✓ Environment configured"
```

---

### Hook Output Parsing

Parse hook output for environment overrides:

```bash
#!/bin/bash
# .dual/hooks/parse-output.sh

set -e

echo "Setting up with output parsing..."

# Run a command that outputs key=value pairs
# For example, a script that calculates ports and database URLs

OUTPUT=$(cat <<'SCRIPT'
#!/bin/bash
# Calculate values
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$1" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))
DB_URL="postgresql://localhost/myapp_$1"

# Output as key=value pairs
echo "PORT=$PORT"
echo "DATABASE_URL=$DB_URL"
echo "API_URL=http://localhost:$PORT"
SCRIPT
)

# Parse output and write to .env
echo "$OUTPUT" | bash -s "$DUAL_CONTEXT_NAME" > "$DUAL_CONTEXT_PATH/.env.local"

echo "✓ Environment configured from parsed output"
```

---

## Troubleshooting

### Hook Execution Fails

#### Problem: Hook script not found

```bash
dual create feature-test

# Error: hook script not found: setup-environment.sh
```

**Solution:**

```bash
# Check if script exists
ls .dual/hooks/setup-environment.sh

# If not, create it
# If it exists, check configuration
grep -A 5 "hooks:" dual.config.yml
```

#### Problem: Hook script not executable

```bash
dual create feature-test

# Error: hook script not executable: setup-environment.sh
```

**Solution:**

```bash
# Make script executable
chmod +x .dual/hooks/setup-environment.sh

# Verify
ls -la .dual/hooks/setup-environment.sh
# Should show: -rwxr-xr-x
```

#### Problem: Hook fails with error

```bash
dual create feature-test

# Running hook: create-database-branch.sh
# Error: pscale: command not found
# Error: hook script failed: create-database-branch.sh (exit code 127)
```

**Solution:**

```bash
# Install missing tool
brew install planetscale/tap/pscale

# Verify installation
pscale version

# Retry
dual create feature-test
```

---

### Port Conflicts

#### Problem: Port already in use

```bash
# App starts on port from hook
npm run dev

# Error: Port 4237 is already in use
```

**Solution:**

```bash
# Find what's using the port
lsof -i :4237

# Kill the process
kill -9 <PID>

# Or choose a different port in your hook logic
```

---

### Database Issues

#### Problem: Database already exists

```bash
dual create feature-test

# Running hook: create-database.sh
# Error: database "myapp_feature_test" already exists
```

**Solution:**

```bash
# Drop existing database
dropdb myapp_feature_test

# Or update hook to check if database exists first
```

---

### Registry Issues

#### Problem: Registry corrupted

```bash
dual context list

# Error: failed to parse registry: invalid JSON
```

**Solution:**

```bash
# Backup corrupted registry
cp .dual/.local/registry.json .dual/.local/registry.json.backup

# dual will auto-create a new empty registry
# You'll need to recreate contexts

dual create main
dual create feature-x
```

---

### Worktree Issues

#### Problem: Worktree path doesn't exist

```bash
dual create feature-test

# Error: worktree path does not exist: ../worktrees
```

**Solution:**

```bash
# Create the directory
mkdir -p ../worktrees

# Retry
dual create feature-test
```

#### Problem: Branch already exists

```bash
dual create feature-auth

# Error: branch 'feature-auth' already exists
```

**Solution:**

```bash
# Use a different branch name
dual create feature-auth-v2

# Or delete the existing branch
git branch -D feature-auth
git worktree prune
dual create feature-auth
```

---

### Debug Mode

Enable debug mode for detailed troubleshooting:

```bash
# Debug single command
dual --debug create feature-test

# Or set environment variable
export DUAL_DEBUG=1
dual create feature-test

# Verbose mode (less detail than debug)
dual --verbose create feature-test
```

---

## Next Steps

- See [USAGE.md](USAGE.md) for complete command reference
- See [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- See [README.md](README.md) for project overview
- Check `.dual/hooks/README.md` for more hook examples

---

## Migration from v0.2.x

If you're upgrading from v0.2.x, note these key changes:

### Removed Features

- **Command wrapper mode**: `dual <command>` no longer injects PORT
- **Port commands**: `dual port` and `dual ports` removed
- **`dual context create`**: Use `dual create <branch>` instead
- **Global registry**: Now project-local at `.dual/.local/registry.json`

### New Features

- **Worktree lifecycle**: `dual create` and `dual delete` commands
- **Hook system**: Lifecycle hooks for custom automation
- **Project-local registry**: Isolated per-project state

### Migration Steps

1. Update configuration with `worktrees` and `hooks` sections
2. Implement port assignment in hooks (see examples above)
3. Recreate contexts with `dual create`
4. Update workflows to use new commands
5. Add registry to `.gitignore`

See [MIGRATION.md](MIGRATION.md) for detailed instructions.
