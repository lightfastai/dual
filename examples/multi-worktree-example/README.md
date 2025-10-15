# Multi-Worktree Example with Custom Port Assignment

This example demonstrates how to use `dual` as a worktree lifecycle manager with custom port assignment implemented via hooks.

## Overview

With dual's refactored architecture, port management is no longer built into the core tool. Instead, dual provides lifecycle hooks that allow you to implement custom port assignment logic. This example shows:

- Creating multiple worktrees with dual
- Automatic port assignment via `postWorktreeCreate` hooks
- Hash-based deterministic port allocation
- Environment file generation for services
- Multi-service monorepo setup

## What This Example Demonstrates

1. **Worktree Lifecycle Management**: Use dual to create and manage git worktrees
2. **Custom Port Assignment**: Implement port allocation in hooks, not in dual core
3. **Deterministic Ports**: Hash-based port assignment ensures same branch always gets same ports
4. **Service-Specific Ports**: Each service (web, api, worker) gets its own calculated port
5. **Environment File Generation**: Automatically create `.env.local` files with correct ports
6. **No Port Conflicts**: Each worktree gets a unique port range

## Architecture

### Port Assignment Strategy

This example uses **hash-based port assignment** for deterministic, conflict-free port allocation:

```
basePort = 4000 + (hash(contextName) % 100) * 100
```

This ensures:
- Same context always gets the same ports (deterministic)
- Wide spacing (100 ports between contexts) prevents conflicts
- Port range: 4000-14000 (100 possible contexts)

Each service then gets an offset from the base port:
- Web: basePort + 1 (e.g., 4001)
- API: basePort + 2 (e.g., 4002)
- Worker: basePort + 3 (e.g., 4003)

### Hook Flow

```
dual create feature-1
  ↓
Git worktree created
  ↓
Context registered in .dual/registry.json
  ↓
postWorktreeCreate hooks executed:
  1. assign-ports.sh
     - Calculate hash-based port
     - Write to .dual-port file
     - Echo assigned ports
  2. setup-environment.sh
     - Read base port from .dual-port
     - Calculate service ports
     - Generate .env.local file
```

## Prerequisites

- **Go**: For building and installing dual
- **Node.js**: For running the example services
- **Git**: For worktree operations
- **dual installed**: Run `go install ./cmd/dual` from the dual project root

## Directory Structure

```
multi-worktree-example/
├── README.md                    # This file
├── dual.config.yml             # Dual configuration with hooks
├── .dual/
│   └── hooks/
│       ├── assign-ports.sh      # Port assignment logic
│       └── setup-environment.sh # Environment file creation
├── apps/
│   ├── web/
│   │   ├── package.json
│   │   └── server.js           # Express web server
│   ├── api/
│   │   ├── package.json
│   │   └── server.js           # Express API server
│   └── worker/
│       ├── package.json
│       └── worker.js            # Background worker
├── .env.example                 # Example environment file
└── demo.sh                      # Automated demonstration script

After running demo.sh:
../worktrees/
├── dev/                         # Development worktree
│   ├── .dual-port              # Base port file
│   ├── .env.local              # Generated environment
│   └── apps/...
├── feature-1/                   # Feature 1 worktree
│   ├── .dual-port
│   ├── .env.local
│   └── apps/...
└── feature-2/                   # Feature 2 worktree
    ├── .dual-port
    ├── .env.local
    └── apps/...
```

## Step-by-Step Tutorial

### 1. Setup

First, navigate to this example directory and make scripts executable:

```bash
cd examples/multi-worktree-example
chmod +x .dual/hooks/*.sh demo.sh
```

### 2. Initialize Git Repository

```bash
# Initialize a git repo (if not already in one)
git init
git add .
git commit -m "Initial commit"
```

### 3. Initialize Dual

```bash
# Initialize dual configuration (already provided)
dual init

# Add services (already configured in dual.config.yml)
# dual service add web apps/web
# dual service add api apps/api
# dual service add worker apps/worker
```

### 4. Create Worktrees

Create multiple worktrees to demonstrate port assignment:

```bash
# Create dev worktree
dual create dev

# Create feature worktrees
dual create feature-1
dual create feature-2
```

### 5. Inspect Generated Files

Check the assigned ports in each worktree:

```bash
# Dev worktree ports
cat ../worktrees/dev/.dual-port
cat ../worktrees/dev/.env.local

# Feature-1 worktree ports
cat ../worktrees/feature-1/.dual-port
cat ../worktrees/feature-1/.env.local

# Feature-2 worktree ports
cat ../worktrees/feature-2/.dual-port
cat ../worktrees/feature-2/.env.local
```

### 6. Run Services

Navigate to any worktree and run a service:

```bash
# In the dev worktree
cd ../worktrees/dev/apps/web
npm install
npm start
# Server will start on the port from .env.local (e.g., PORT_WEB=4001)

# In another terminal, run feature-1
cd ../worktrees/feature-1/apps/web
npm install
npm start
# Server will start on a different port (e.g., PORT_WEB=7301)
```

### 7. Cleanup

Delete worktrees when done:

```bash
dual delete dev
dual delete feature-1
dual delete feature-2
```

## Automated Demo

Run the included demo script to see everything in action:

```bash
./demo.sh
```

This will:
1. Initialize a git repository
2. Initialize dual configuration
3. Create three worktrees (dev, feature-1, feature-2)
4. Show the registry contents
5. Display port assignments for each worktree
6. Show the generated .env.local files
7. Demonstrate that each worktree has unique ports
8. Clean up all worktrees

## Expected Output

### Port Assignments

Based on hash-based calculation, you might see:

```
Dev worktree (hash: 17):
  Base Port: 5700
  PORT_WEB=5701
  PORT_API=5702
  PORT_WORKER=5703

Feature-1 worktree (hash: 13):
  Base Port: 5300
  PORT_WEB=5301
  PORT_API=5302
  PORT_WORKER=5303

Feature-2 worktree (hash: 38):
  Base Port: 7800
  PORT_WEB=7801
  PORT_API=7802
  PORT_WORKER=7803
```

Note: Actual ports will vary based on the hash of the context name.

### Registry Structure

The `.dual/registry.json` will look like:

```json
{
  "projects": {
    "/path/to/project": {
      "contexts": {
        "dev": {
          "created": "2024-01-15T10:30:00Z",
          "path": "/path/to/worktrees/dev"
        },
        "feature-1": {
          "created": "2024-01-15T10:31:00Z",
          "path": "/path/to/worktrees/feature-1"
        },
        "feature-2": {
          "created": "2024-01-15T10:32:00Z",
          "path": "/path/to/worktrees/feature-2"
        }
      }
    }
  }
}
```

## Customization

### Changing Port Assignment Strategy

Edit `.dual/hooks/assign-ports.sh` to implement different strategies:

**Sequential allocation:**
```bash
# Count existing contexts and multiply by 100
num_contexts=$(count_existing_contexts)
BASE_PORT=$((4000 + num_contexts * 100))
```

**Fixed ranges:**
```bash
# Assign based on branch name pattern
case "$DUAL_CONTEXT_NAME" in
  dev)       BASE_PORT=4000 ;;
  feature-*) BASE_PORT=5000 ;;
  hotfix-*)  BASE_PORT=6000 ;;
  *)         BASE_PORT=7000 ;;
esac
```

**Random allocation with conflict detection:**
```bash
# Generate random port and check for conflicts
while true; do
  BASE_PORT=$((4000 + RANDOM % 10000))
  if ! port_in_use "$BASE_PORT"; then
    break
  fi
done
```

### Adding More Services

1. Add service to `dual.config.yml`:
```yaml
services:
  database:
    path: apps/database
    envFile: apps/database/.env
```

2. Update `assign-ports.sh` to calculate database port:
```bash
DATABASE_PORT=$((BASE_PORT + 4))
```

3. Update `setup-environment.sh` to include database port:
```bash
cat >> .env.local << EOF
PORT_DATABASE=${DATABASE_PORT}
EOF
```

### Adding More Environment Variables

Edit `setup-environment.sh` to add custom variables:

```bash
# Add context-specific database URL
cat >> .env.local << EOF
DATABASE_URL=postgresql://localhost:5432/${DUAL_CONTEXT_NAME}
REDIS_URL=redis://localhost:6379/${DUAL_CONTEXT_NAME}
NODE_ENV=development
EOF
```

## Troubleshooting

### Hooks Not Executing

**Problem**: Hooks don't run when creating worktrees.

**Solutions**:
1. Ensure hooks are executable: `chmod +x .dual/hooks/*.sh`
2. Check hook configuration in `dual.config.yml`
3. Verify hook files exist in `.dual/hooks/` directory

### Port Conflicts

**Problem**: Services can't bind to assigned ports.

**Solutions**:
1. Check if another process is using the port: `lsof -i :PORT`
2. Modify hash function in `assign-ports.sh` to use different range
3. Implement conflict detection in the hook

### Environment File Not Created

**Problem**: `.env.local` not found in worktree.

**Solutions**:
1. Check `setup-environment.sh` executed successfully
2. Verify `.dual-port` file exists
3. Run the hook manually for debugging:
```bash
cd /path/to/worktree
DUAL_CONTEXT_NAME=test DUAL_CONTEXT_PATH=$(pwd) DUAL_PROJECT_ROOT=/path/to/project .dual/hooks/setup-environment.sh
```

### Registry Lock Errors

**Problem**: "timeout waiting for registry lock" error.

**Solutions**:
1. Check for stale lock file: `rm .dual/registry.json.lock`
2. Ensure no other dual commands are running
3. Increase lock timeout in dual code if needed

## Advanced Topics

### Running Services in Parallel

Use a process manager to run all services across worktrees:

```bash
# Create a Procfile
cat > Procfile << EOF
main-web: cd ../worktrees/main/apps/web && npm start
main-api: cd ../worktrees/main/apps/api && npm start
feature1-web: cd ../worktrees/feature-1/apps/web && npm start
feature1-api: cd ../worktrees/feature-1/apps/api && npm start
EOF

# Run with foreman/overmind
overmind start
```

### Integration with Docker

Modify `setup-environment.sh` to generate docker-compose port mappings:

```bash
# Generate docker-compose.override.yml
cat > docker-compose.override.yml << EOF
version: '3'
services:
  web:
    ports:
      - "${PORT_WEB}:3000"
  api:
    ports:
      - "${PORT_API}:3000"
EOF
```

### CI/CD Integration

Set up different contexts for different environments:

```bash
dual create staging-v1
dual create staging-v2
dual create production
```

Each gets isolated ports and environment variables.

## Contributing

This example is part of the dual project. To contribute improvements:

1. Test your changes: Run `./demo.sh` to verify everything works
2. Update this README if you add features
3. Submit a PR to the dual repository

## License

This example is part of the dual project and uses the same license.

## Learn More

- [Dual Documentation](https://github.com/lightfastai/dual)
- [Git Worktrees Guide](https://git-scm.com/docs/git-worktree)
- [Environment Variables Best Practices](https://12factor.net/config)
