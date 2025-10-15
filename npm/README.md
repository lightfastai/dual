# @lightfastai/dual

> Git worktree lifecycle management with environment remapping via hooks

[![npm version](https://img.shields.io/npm/v/@lightfastai/dual)](https://www.npmjs.com/package/@lightfastai/dual)
[![npm downloads](https://img.shields.io/npm/dm/@lightfastai/dual)](https://www.npmjs.com/package/@lightfastai/dual)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/lightfastai/dual/blob/main/LICENSE)

`dual` is a CLI tool that manages git worktree lifecycle (creation, deletion) with a flexible hook system for custom environment configuration. It enables developers to work on multiple features simultaneously by automating worktree setup and providing hooks for custom environment setup (database branches, port assignment, dependency installation, etc.).

## Installation

```bash
# npm
npm install --save-dev @lightfastai/dual

# pnpm
pnpm add -D @lightfastai/dual

# yarn
yarn add -D @lightfastai/dual
```

## Quick Start

### 1. Initialize your project

```bash
npx dual init
```

This creates `dual.config.yml` in your project root.

### 2. Register your services

```bash
# For a monorepo
npx dual service add web --path apps/web
npx dual service add api --path apps/api

# For a single-service project
npx dual service add app --path .
```

### 3. Configure worktrees

Edit `dual.config.yml` to add worktree configuration:

```yaml
version: 1

services:
  web:
    path: ./apps/web
  api:
    path: ./apps/api

worktrees:
  path: ../worktrees
  naming: "{branch}"
```

### 4. Create worktrees with hooks

```bash
# Create a new worktree for a feature branch
npx dual create feature-auth

# This will:
# 1. Create git worktree at ../worktrees/feature-auth
# 2. Register the context
# 3. Run postWorktreeCreate hooks (if configured)
```

### 5. Delete worktrees with cleanup

```bash
# Delete a worktree and run cleanup hooks
npx dual delete feature-auth

# This will:
# 1. Run preWorktreeDelete hooks (if configured)
# 2. Remove git worktree
# 3. Unregister the context
# 4. Run postWorktreeDelete hooks (if configured)
```

## Using npx vs Local Installation

### With local installation (recommended)

When you install `@lightfastai/dual` as a dev dependency, you can use it directly in your package.json scripts:

```json
{
  "scripts": {
    "worktree:create": "dual create",
    "worktree:delete": "dual delete",
    "worktree:list": "dual context list"
  }
}
```

npm/pnpm/yarn will automatically find the locally installed binary in `node_modules/.bin/`.

### With npx (no installation)

You can also use `dual` without installing it:

```bash
npx @lightfastai/dual create feature-auth
npx @lightfastai/dual delete feature-auth
```

However, this downloads the package on every run, so local installation is recommended for regular use.

## How It Works

The npm package downloads the appropriate native Go binary for your platform during installation:

1. **Postinstall**: Downloads the correct binary from GitHub Releases based on your OS and architecture
2. **Execution**: The `dual` command runs the native binary, passing through all arguments and environment variables
3. **Streaming**: stdout/stderr are streamed in real-time, exit codes are preserved

### Supported Platforms

- **macOS**: x64, arm64 (Apple Silicon)
- **Linux**: x64, arm64
- **Windows**: x64, arm64

## Benefits of the npm Package

- **Zero manual installation**: Binary is downloaded automatically during `npm install`
- **Team-friendly**: Committed to `package.json`, ensures everyone uses the same version
- **CI-ready**: Works in CI/CD without additional setup steps
- **Version pinning**: Lock to specific versions with package.json
- **Cross-platform**: Automatically installs the correct binary for each platform

## Example: Monorepo Setup with Hooks

**package.json (root):**
```json
{
  "name": "my-monorepo",
  "scripts": {
    "worktree:new": "dual create",
    "worktree:remove": "dual delete",
    "worktree:list": "dual context list"
  },
  "devDependencies": {
    "@lightfastai/dual": "^0.3.0"
  }
}
```

**dual.config.yml:**
```yaml
version: 1

services:
  web:
    path: apps/web
  api:
    path: apps/api

worktrees:
  path: ../worktrees
  naming: "{branch}"

hooks:
  postWorktreeCreate:
    - setup-database.sh
    - setup-environment.sh
    - install-dependencies.sh
  preWorktreeDelete:
    - cleanup-database.sh
```

**.dual/hooks/setup-environment.sh:**
```bash
#!/bin/bash
set -e

echo "Setting up environment for: $DUAL_CONTEXT_NAME"

# Calculate port based on context name hash
BASE_PORT=4000
CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

# Write to .env file
cat > "$DUAL_CONTEXT_PATH/.env.local" <<EOF
PORT=$PORT
DATABASE_URL=postgresql://localhost/myapp_${DUAL_CONTEXT_NAME}
EOF

echo "Assigned port: $PORT"
```

Now team members just run:
```bash
pnpm install              # Installs dual automatically
pnpm worktree:new feat-x  # Creates worktree with hooks
pnpm worktree:list        # Lists all worktrees
pnpm worktree:remove feat-x  # Deletes worktree with cleanup
```

## Troubleshooting

### Binary not found after installation

If you see "Binary not found" errors:

1. Try reinstalling:
   ```bash
   npm install --force @lightfastai/dual
   ```

2. Check if the postinstall script ran:
   ```bash
   ls node_modules/@lightfastai/dual/bin/
   ```

3. Manually run postinstall:
   ```bash
   cd node_modules/@lightfastai/dual
   node postinstall.js
   ```

### Unsupported platform

If your platform is not supported, you can install `dual` using alternative methods:

**Homebrew (macOS/Linux):**
```bash
brew tap lightfastai/tap
brew install dual
```

**Direct download:**
```bash
curl -sSL "https://github.com/lightfastai/dual/releases/latest/download/dual_$(uname -s)_$(uname -m).tar.gz" | \
  sudo tar -xzf - -C /usr/local/bin dual
```

**Build from source:**
```bash
go install github.com/lightfastai/dual/cmd/dual@latest
```

### Network issues during installation

If the binary download fails due to network issues:

1. Check your internet connection
2. Try again with verbose npm logging:
   ```bash
   npm install --loglevel verbose @lightfastai/dual
   ```

3. Manually download the binary from [GitHub Releases](https://github.com/lightfastai/dual/releases) and place it in `node_modules/@lightfastai/dual/bin/`

## Migrating from v0.2.x to v0.3.0

Version 0.3.0 introduces breaking changes. The CLI now focuses on worktree lifecycle management with hooks, removing automatic port management.

### What Changed

**Removed Features:**
- Automatic port assignment and management
- Command wrapper mode (`dual <command>`)
- `dual port` and `dual ports` commands
- `dual context create` with `--base-port` flag
- `postPortAssign` hook event

**New Features:**
- `dual create <branch>` - Create worktrees with lifecycle hooks
- `dual delete <context>` - Delete worktrees with cleanup hooks
- Hook system for custom environment setup
- Project-local registry (moved from `~/.dual/.local/registry.json` to `$PROJECT_ROOT/.dual/.local/registry.json`)

### Migration Steps

1. **Update your package.json:**
   ```bash
   npm install @lightfastai/dual@^0.3.0
   ```

2. **Remove command wrapper usage:**
   ```diff
   - "dev": "dual pnpm dev"
   + "dev": "pnpm dev"
   ```

3. **If you need port assignment, implement it in hooks:**

   Create `.dual/hooks/setup-environment.sh`:
   ```bash
   #!/bin/bash
   set -e

   # Calculate port based on context name hash
   BASE_PORT=4000
   CONTEXT_HASH=$(echo -n "$DUAL_CONTEXT_NAME" | md5sum | cut -c1-4)
   PORT=$((BASE_PORT + 0x$CONTEXT_HASH % 1000))

   # Write to .env file
   echo "PORT=$PORT" > "$DUAL_CONTEXT_PATH/.env.local"
   ```

   Configure in `dual.config.yml`:
   ```yaml
   hooks:
     postWorktreeCreate:
       - setup-environment.sh
   ```

4. **Update worktree workflows:**
   ```diff
   - npx dual context create feature-x --base-port 4200
   + npx dual create feature-x

   - npx dual context delete feature-x
   + npx dual delete feature-x
   ```

5. **Recreate existing contexts:**

   The registry has moved from `~/.dual/.local/registry.json` to `$PROJECT_ROOT/.dual/.local/registry.json`. Existing contexts need to be recreated:
   ```bash
   # Delete old contexts (if any)
   rm -rf ~/.dual/.local/registry.json

   # Recreate worktrees with new system
   npx dual create feature-x
   ```

For more details, see the [Migration Guide](https://github.com/lightfastai/dual/blob/main/MIGRATION.md) in the main repository.

## Documentation

For complete documentation, see the [main repository](https://github.com/lightfastai/dual):

- [Full README](https://github.com/lightfastai/dual#readme)
- [Usage Guide](https://github.com/lightfastai/dual/blob/main/USAGE.md)
- [Architecture](https://github.com/lightfastai/dual/blob/main/ARCHITECTURE.md)
- [Examples](https://github.com/lightfastai/dual/blob/main/EXAMPLES.md)

## Comparison with Other Installation Methods

| Method | Team Friendly | CI Ready | Auto-installs | Version Locked |
|--------|---------------|----------|---------------|----------------|
| npm package | ✅ | ✅ | ✅ | ✅ |
| Homebrew | ❌ | ⚠️ | ❌ | ⚠️ |
| Direct download | ❌ | ⚠️ | ❌ | ❌ |
| Go install | ❌ | ⚠️ | ❌ | ⚠️ |

The npm package is the recommended installation method for web projects using npm/pnpm/yarn.

## License

Apache License 2.0 - see [LICENSE](https://github.com/lightfastai/dual/blob/main/LICENSE) for details.

## Support

- [Report a bug](https://github.com/lightfastai/dual/issues/new)
- [Request a feature](https://github.com/lightfastai/dual/issues/new)
- [View documentation](https://github.com/lightfastai/dual)

---

Made with ❤️ by [Lightfast](https://github.com/lightfastai)
