# Dual Hooks

This directory contains lifecycle hook scripts that run automatically during worktree management operations.

## Hook Events

- **postWorktreeCreate**: Runs after creating a new worktree
- **postPortAssign**: Runs after assigning ports to a context
- **preWorktreeDelete**: Runs before deleting a worktree
- **postWorktreeDelete**: Runs after deleting a worktree

## Environment Variables

All hook scripts receive these environment variables:

- `DUAL_EVENT`: The hook event name (e.g., "postWorktreeCreate")
- `DUAL_CONTEXT_NAME`: Context name (usually the branch name)
- `DUAL_CONTEXT_PATH`: Absolute path to the worktree directory
- `DUAL_PROJECT_ROOT`: Absolute path to the main repository
- `DUAL_BASE_PORT`: Base port assigned to this context
- `DUAL_PORT_<SERVICE>`: Port for each service (e.g., `DUAL_PORT_WEB=4201`)

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
- `setup-environment.sh`: Updates .env files with port numbers
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
echo "Port: $DUAL_BASE_PORT"
echo "Path: $DUAL_CONTEXT_PATH"

# Your custom logic here
```

## Testing Hooks

Test hooks by running:

```bash
# Create a worktree (triggers postWorktreeCreate and postPortAssign)
dual create feature-test

# Delete a worktree (triggers preWorktreeDelete and postWorktreeDelete)
dual delete feature-test
```

Check hook output in the terminal during execution.
