# @lightfastai/dual

> General-purpose port management for multi-context development

[![npm version](https://img.shields.io/npm/v/@lightfastai/dual)](https://www.npmjs.com/package/@lightfastai/dual)
[![npm downloads](https://img.shields.io/npm/dm/@lightfastai/dual)](https://www.npmjs.com/package/@lightfastai/dual)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/lightfastai/dual/blob/main/LICENSE)

`dual` is a CLI tool that automatically manages port assignments across different development contexts (git branches, worktrees, or clones), eliminating port conflicts when working on multiple features simultaneously.

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
npx dual service add web --path apps/web --env-file .vercel/.env.development.local
npx dual service add api --path apps/api --env-file .env

# For a single-service project
npx dual service add app --path . --env-file .env.local
```

### 3. Create contexts

```bash
# Main branch gets default ports
npx dual context create main --base-port 4100

# Feature branch in worktree gets different ports
cd ~/Code/myproject-wt/feature-auth
npx dual context create feature-auth --base-port 4200
```

### 4. Use in package.json scripts

```json
{
  "scripts": {
    "dev": "dual pnpm dev",
    "build": "dual pnpm build",
    "start": "dual npm start"
  }
}
```

Now run your scripts normally:

```bash
pnpm dev    # Runs with auto-detected port
pnpm build  # Builds with correct PORT
pnpm start  # Starts with correct PORT
```

## Using npx vs Local Installation

### With local installation (recommended)

When you install `@lightfastai/dual` as a dev dependency, you can use it directly in your package.json scripts:

```json
{
  "scripts": {
    "dev": "dual pnpm dev"
  }
}
```

npm/pnpm/yarn will automatically find the locally installed binary in `node_modules/.bin/`.

### With npx (no installation)

You can also use `dual` without installing it:

```bash
npx @lightfastai/dual pnpm dev
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

## Example: Monorepo Setup

**package.json (root):**
```json
{
  "name": "my-monorepo",
  "scripts": {
    "dev:web": "dual pnpm --filter web dev",
    "dev:api": "dual pnpm --filter api dev",
    "dev:all": "concurrently \"pnpm dev:web\" \"pnpm dev:api\"",
    "build": "dual pnpm -r build"
  },
  "devDependencies": {
    "@lightfastai/dual": "^0.1.0",
    "concurrently": "^8.0.0"
  }
}
```

**dual.config.yml:**
```yaml
version: 1
services:
  web:
    path: apps/web
    envFile: .vercel/.env.development.local
  api:
    path: apps/api
    envFile: .env
```

Now team members just run:
```bash
pnpm install       # Installs dual automatically
pnpm dev:web       # Runs web with auto-detected port
pnpm dev:api       # Runs api with auto-detected port
pnpm dev:all       # Runs both simultaneously, no conflicts!
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
