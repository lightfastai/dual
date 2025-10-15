# Dual Hooks

This directory contains lifecycle hook scripts that run automatically during worktree management operations.

## Hook Events

- **postWorktreeCreate**: Runs after creating a new worktree
- **preWorktreeDelete**: Runs before deleting a worktree
- **postWorktreeDelete**: Runs after deleting a worktree

## Environment Variables

All hook scripts receive these environment variables:

- `DUAL_EVENT`: The hook event name (e.g., "postWorktreeCreate")
- `DUAL_CONTEXT_NAME`: Context name (usually the branch name)
- `DUAL_CONTEXT_PATH`: Absolute path to the worktree directory
- `DUAL_PROJECT_ROOT`: Absolute path to the main repository

## Configuration

Configure hooks in `dual.config.yml`:

```yaml
hooks:
  postWorktreeCreate:
    - create-database-branch.sh
    - setup-environment.sh
  preWorktreeDelete:
    - cleanup-database.sh
```

## Example Hooks

- `create-database-branch.sh`: Creates a PlanetScale database branch
- `setup-environment.sh`: Sets up environment files for the worktree
- `cleanup-database.sh`: Deletes database branches before worktree removal

## Writing Custom Hooks

1. Create a shell script in this directory
2. Make it executable: `chmod +x .dual/hooks/your-hook.sh`
3. Add it to `dual.config.yml` under the appropriate event
4. Use the environment variables to access context information

Example:

```bash
#!/bin/bash
set -e

echo "Context: $DUAL_CONTEXT_NAME"
echo "Path: $DUAL_CONTEXT_PATH"
echo "Project Root: $DUAL_PROJECT_ROOT"

# Your custom logic here
# For example, you could implement custom port assignment:
# - Read existing contexts from registry
# - Calculate next available port
# - Write to .env file
```

## Common Use Cases

### Custom Port Assignment

Dual no longer manages ports automatically. If you need port management, you can implement it in a hook:

```bash
#!/bin/bash
set -e

# Calculate port based on context name hash or sequential assignment
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Write to .env file
echo "PORT=$PORT" > "$DUAL_CONTEXT_PATH/.env.local"
echo "Assigned port: $PORT"
```

### Database Branch Creation

Create isolated database branches per worktree (PlanetScale, Neon, etc.):

```bash
#!/bin/bash
set -e

# Create branch in your database service
pscale branch create mydb "$DUAL_CONTEXT_NAME" --from main
```

### Environment File Setup

Copy and customize environment files for each worktree:

```bash
#!/bin/bash
set -e

# Copy base .env to worktree
cp "$DUAL_PROJECT_ROOT/.env.example" "$DUAL_CONTEXT_PATH/.env.local"

# Customize for this context
sed -i '' "s/CONTEXT_NAME=.*/CONTEXT_NAME=$DUAL_CONTEXT_NAME/" "$DUAL_CONTEXT_PATH/.env.local"
```

## Testing Hooks

Test hooks by running:

```bash
# Create a worktree (triggers postWorktreeCreate)
dual create feature-test

# Delete a worktree (triggers preWorktreeDelete and postWorktreeDelete)
dual delete feature-test
```

Check hook output in the terminal during execution.
