# Release Process

This document describes the automated release process for `dual` using GoReleaser and GitHub Actions.

## Overview

The release process is fully automated using:
- **GoReleaser**: Builds binaries, creates archives, generates checksums, and manages releases
- **GitHub Actions**: Triggers on version tags to run GoReleaser
- **Homebrew Tap**: Automatically updates the formula in `lightfastai/homebrew-tap`

## Automated Release Workflow

### 1. Create and Push a Version Tag

```bash
# Ensure all changes are committed
git add .
git commit -m "chore: prepare for release v1.0.0"

# Create a version tag (use semantic versioning)
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to trigger the release
git push origin v1.0.0
```

### 2. GitHub Actions Automatically:

1. Checks out the code
2. Sets up Go 1.23
3. Runs tests via `go test ./...`
4. Runs GoReleaser which:
   - Builds binaries for:
     - darwin/amd64
     - darwin/arm64
     - linux/amd64
     - linux/arm64
   - Creates tar.gz archives with README and LICENSE
   - Generates SHA256 checksums
   - Creates GitHub release with changelog
   - Updates Homebrew tap at `lightfastai/homebrew-tap`

### 3. Release is Published

- GitHub Release is created at: `https://github.com/lightfastai/dual/releases`
- Homebrew formula is updated automatically
- Users can install via: `brew tap lightfastai/tap && brew install dual`

## Version Information

Binaries are built with version information embedded via ldflags:
- `version`: Git tag (e.g., v1.0.0)
- `commit`: Git commit SHA
- `date`: Build timestamp

Check version with:
```bash
dual --version
```

## Supported Platforms

### macOS
- Apple Silicon (arm64)
- Intel (amd64)

### Linux
- ARM64
- x86_64 (amd64)

## Installation Methods

### Homebrew (Recommended for macOS/Linux)

```bash
# Add the tap (only needed once)
brew tap lightfastai/tap

# Install dual
brew install dual

# Or in one command
brew tap lightfastai/tap && brew install dual
```

### Manual Installation

1. Go to [Releases](https://github.com/lightfastai/dual/releases)
2. Download the appropriate archive for your platform
3. Extract and move `dual` to a directory in your PATH:
   ```bash
   tar -xzf dual_Darwin_arm64.tar.gz
   sudo mv dual /usr/local/bin/
   ```

### Verification

Verify the downloaded binary with checksums:
```bash
# Download checksums.txt from the release
sha256sum -c checksums.txt
```

## Release Types

### Stable Release
Tag format: `v1.0.0`, `v1.2.3`
- Full semantic version
- Creates a stable release
- Updates Homebrew formula

### Pre-release
Tag format: `v1.0.0-rc1`, `v1.0.0-beta1`, `v1.0.0-alpha1`
- Marked as pre-release automatically
- Does not update Homebrew formula (stable channel)

## Changelog Generation

GoReleaser automatically generates changelogs from commit messages using conventional commit format:

- `feat:` → Features section
- `fix:` → Bug fixes section
- `perf:` → Performance improvements
- `refactor:` → Refactors section

Example commit messages:
```bash
feat: add automatic context detection from git branch
fix: resolve port conflict in multi-service setup
perf: optimize registry file access with caching
refactor: simplify service detection logic
```

## Testing Releases Locally

Install GoReleaser:
```bash
brew install goreleaser
```

Test the release process without publishing:
```bash
# Snapshot build (no tag required)
goreleaser release --snapshot --clean

# Check what would be released
goreleaser check
```

Build artifacts will be in the `dist/` directory.

## Homebrew Tap Setup

The Homebrew tap repository `lightfastai/homebrew-tap` is automatically managed by GoReleaser.

### First-Time Setup

1. Create the tap repository: `lightfastai/homebrew-tap`
2. GoReleaser will automatically:
   - Create `Formula/dual.rb`
   - Update it on each release
   - Commit and push changes

### Manual Formula Update (if needed)

If you need to manually update the formula:
```bash
# Clone the tap
git clone https://github.com/lightfastai/homebrew-tap.git

# Edit Formula/dual.rb
cd homebrew-tap
vim Formula/dual.rb

# Commit and push
git commit -am "Update dual formula"
git push
```

## Troubleshooting

### Release Fails Due to Permissions

Ensure the GitHub token has the correct permissions:
- `contents: write` - For creating releases
- `packages: write` - For pushing to other repos

If using a PAT (Personal Access Token):
1. Create a token with `repo` scope
2. Add it as `GORELEASER_TOKEN` secret
3. Update `.github/workflows/release.yml` to use it

### Homebrew Formula Not Updating

1. Check that `lightfastai/homebrew-tap` repository exists
2. Verify GitHub token has push access to the tap repo
3. Check GoReleaser logs in GitHub Actions

### Build Fails for Specific Platform

1. Check Go compatibility with the target platform
2. Review CGO requirements (currently disabled)
3. Check GitHub Actions logs for detailed error messages

## CI/CD Integration

The release workflow is defined in `.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    tags:
      - 'v*'
```

This workflow:
- Triggers on any tag starting with `v`
- Runs on `ubuntu-latest`
- Uses `goreleaser/goreleaser-action@v6`
- Requires Go 1.23

## Configuration Files

### .goreleaser.yml
Main configuration for:
- Build targets and flags
- Archive formats
- Homebrew tap settings
- Changelog generation
- Release notes

### .github/workflows/release.yml
GitHub Actions workflow that:
- Checks out code with full history
- Sets up Go environment
- Runs GoReleaser with GITHUB_TOKEN

## Best Practices

1. **Always test locally first**: Use `goreleaser --snapshot` before tagging
2. **Use semantic versioning**: Follow `vMAJOR.MINOR.PATCH` format
3. **Write good commit messages**: Use conventional commits for better changelogs
4. **Create annotated tags**: Use `git tag -a` with descriptive messages
5. **Review the release**: Check GitHub release page after automation completes

## Security

- Binaries are built with `-s -w` ldflags to strip debug info
- CGO is disabled for static binaries
- Checksums are provided for all artifacts
- All builds happen in GitHub's secure runners

## Support

For issues with:
- **Releases**: Check GitHub Actions logs
- **Installation**: See installation instructions in README
- **Homebrew**: Check `lightfastai/homebrew-tap` repository
