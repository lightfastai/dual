# Release Process

This document describes the automated release process for `dual` using GoReleaser and GitHub Actions.

## Overview

The release process is fully automated across **three distribution channels**:
- **GitHub Releases**: Go binaries for darwin/linux (amd64/arm64)
- **Homebrew Tap**: Formula in `lightfastai/homebrew-tap`
- **npm Registry**: Node.js wrapper package `@lightfastai/dual`

Powered by:
- **GoReleaser**: Builds binaries, creates archives, generates checksums
- **GitHub Actions**: Multi-job workflow triggered by version tags
- **npm with Provenance**: Supply chain security for npm packages

## Automated Release Workflow

### Recommended: Changesets Workflow (Fully Automated)

The dual project uses [Changesets](https://github.com/changesets/changesets) for automated version management and CHANGELOG generation.

**Developer Workflow:**

```bash
# 1. Create your feature branch
git checkout -b feature/add-context-switching

# 2. Make your changes
vim internal/context/detector.go

# 3. Add a changeset describing your changes
npm run changeset

# You'll be prompted:
# ? What kind of change is this? → minor
# ? Please enter a summary → Add automatic context switching between branches

# This creates a file like: .changeset/funny-clouds-dance.md

# 4. Commit everything together
git add .
git commit -m "feat: add automatic context switching"

# 5. Create PR and merge when ready
gh pr create --fill
```

**What Happens Automatically:**

1. When your PR with a changeset is merged to `main`, the Changesets GitHub Action triggers
2. A "Version Packages" PR is automatically created/updated
3. The Version Packages PR contains:
   - Updated `npm/package.json` version (following semver)
   - Updated `CHANGELOG.md` with all changes since last release
   - All consumed changesets removed
4. When a maintainer merges the "Version Packages" PR:
   - The version commit is pushed to `main`
   - The release workflow automatically triggers (detects version change)
   - A git tag is created (e.g., `v0.2.0`)
   - GoReleaser publishes to GitHub and Homebrew
   - npm package is published with provenance
   - All three channels are verified automatically

**Visual Flow:**

```
PR with changeset merged
↓
Changesets bot creates "Version Packages" PR
↓
Maintainer reviews changelog and version
↓
Merge "Version Packages" PR
↓
Release workflow auto-triggers (path-based on npm/package.json)
↓
├─ Create git tag (v0.2.0)
├─ Build binaries with GoReleaser
├─ Publish to npm with provenance
└─ Verify all channels
↓
✅ Release complete on GitHub, Homebrew, and npm
```

### Alternative: Manual Release Process (Emergency/Hotfix)

If you need to bypass the changeset workflow (e.g., emergency hotfix):

**Option 1: Release Preparation Script**

```bash
# Prepare a release (dry-run first to preview)
./scripts/prepare-release.sh --dry-run 1.2.3

# Prepare the actual release
./scripts/prepare-release.sh 1.2.3

# Push to trigger the release
git push origin main
git push origin v1.2.3
```

**Option 2: Fully Manual**

```bash
# Update npm package version
cd npm
npm version 1.2.3 --no-git-tag-version
cd ..

# Update CHANGELOG.md manually

# Commit the version bump
git add npm/package.json CHANGELOG.md
git commit -m "chore: bump version to v1.2.3"

# Create and push tag
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin main
git push origin v1.2.3
```

### 2. GitHub Actions Automatically Runs Three Jobs:

#### Job 1: goreleaser
1. Checks out the code with full history
2. Sets up Go 1.23
3. Runs GoReleaser which:
   - Builds binaries for darwin/amd64, darwin/arm64, linux/amd64, linux/arm64
   - Creates tar.gz archives with README and LICENSE
   - Generates SHA256 checksums
   - Creates GitHub release with changelog
   - Updates Homebrew tap at `lightfastai/homebrew-tap`

#### Job 2: npm-publish (depends on goreleaser)
1. Extracts version from git tag (v1.2.3 → 1.2.3)
2. Updates npm/package.json with the version
3. Publishes to npm with provenance attestations
4. Requires `NPM_TOKEN` secret

#### Job 3: verify-release (depends on both)
1. Waits 60 seconds for npm propagation
2. Verifies GitHub Release exists
3. Verifies npm package is published
4. Tests binary download and execution
5. Checks Homebrew formula was updated
6. Prints summary of all release channels

### 3. Release is Published Across All Channels

- **GitHub Release**: `https://github.com/lightfastai/dual/releases`
- **Homebrew**: `brew tap lightfastai/tap && brew install dual`
- **npm**: `npm install -g @lightfastai/dual`

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

### npm (Cross-platform, easiest)

```bash
# Install globally
npm install -g @lightfastai/dual

# Or use npx (no install needed)
npx @lightfastai/dual --version
```

The npm package automatically downloads the correct binary for your platform during installation.

### Homebrew (Recommended for macOS/Linux)

```bash
# Add the tap (only needed once)
brew tap lightfastai/tap

# Install dual
brew install dual

# Or in one command
brew tap lightfastai/tap && brew install dual
```

### Manual Installation from GitHub Releases

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

For npm packages, provenance attestations provide cryptographic proof of build origin.

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

## Required GitHub Secrets

The automated release process requires the following secrets to be configured in the repository:

### LIGHTFAST_RELEASE_BOT_GITHUB_TOKEN (Required for Version Packages PRs)

A GitHub Personal Access Token (PAT) with permissions to create and update pull requests.

**How to create:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Set description: "lightfast-release-bot - dual releases"
4. Select scopes:
   - `repo` (full control of repositories)
   - `workflow` (update GitHub Actions workflows)
5. Generate and copy the token
6. Add to repository secrets as `LIGHTFAST_RELEASE_BOT_GITHUB_TOKEN`

**Used for:**
- Creating "Version Packages" pull requests
- Committing version updates

### LIGHTFAST_RELEASE_BOT_HOMEBREW_TAP_TOKEN (Required for Homebrew updates)

A GitHub Personal Access Token (PAT) with permissions to push to `lightfastai/homebrew-tap`.

**How to create:**
1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Set description: "lightfast-release-bot - homebrew tap updates"
4. Select scope: `repo` (full control of repositories)
5. Generate and copy the token
6. Add to repository secrets as `LIGHTFAST_RELEASE_BOT_HOMEBREW_TAP_TOKEN`

**Alternatively**, use a fine-grained PAT:
- Repository access: `lightfastai/homebrew-tap`
- Permissions: Contents (Read and write)

**Fallback:** If not set, the workflow will fall back to `GITHUB_TOKEN`, but this may not have permission to push to the tap repository.

### LIGHTFAST_RELEASE_BOT_NPM_TOKEN (Required for npm publishing)

An npm automation token for publishing `@lightfastai/dual`.

**How to create:**
1. Log in to npmjs.com
2. Go to Access Tokens → Generate New Token
3. Select "Automation" token type (recommended for CI/CD)
4. Copy the token
5. Add to repository secrets as `LIGHTFAST_RELEASE_BOT_NPM_TOKEN`

**Permissions needed:**
- Publish access to `@lightfastai` organization (or public if not scoped)

### Summary of Required Secrets

All three secrets should use the `LIGHTFAST_RELEASE_BOT_` prefix for consistency:

```bash
# Set all secrets at once
gh secret set LIGHTFAST_RELEASE_BOT_GITHUB_TOKEN
gh secret set LIGHTFAST_RELEASE_BOT_HOMEBREW_TAP_TOKEN
gh secret set LIGHTFAST_RELEASE_BOT_NPM_TOKEN
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
2. Verify `LIGHTFAST_RELEASE_BOT_HOMEBREW_TAP_TOKEN` has push access to the tap repo
3. Check GoReleaser logs in GitHub Actions

### npm Publishing Fails

**Error: "Unable to authenticate"**
- Verify `LIGHTFAST_RELEASE_BOT_NPM_TOKEN` secret is set correctly
- Check token hasn't expired (automation tokens don't expire by default)
- Ensure token has publish permissions

**Error: "Version already exists"**
- Check if version was already published: `npm view @lightfastai/dual`
- Use a different version number
- Cannot re-publish the same version (npm policy)

**Error: "Package name too similar to existing package"**
- First time publishing: May need to verify ownership of `@lightfastai` scope
- Contact npm support if scope ownership issues persist

### Build Fails for Specific Platform

1. Check Go compatibility with the target platform
2. Review CGO requirements (currently disabled)
3. Check GitHub Actions logs for detailed error messages

### Release Workflow Doesn't Trigger After Version Packages PR

**Symptom:**
- "Version Packages" PR was merged
- Release workflow never started

**Solution:**
Check the workflow run logs in the Actions tab. The release workflow triggers automatically on `npm/package.json` changes. If it didn't run:

1. Verify the commit message contains "version packages"
2. Check that the workflow file exists at `.github/workflows/release.yml`
3. Look for workflow errors in the Actions tab

```bash
# Check recent workflow runs
gh run list --workflow=release.yml --limit 5

# View details of a specific run
gh run view <run-id> --log
```

### Verification Job Fails

**npm package not found after 60 seconds:**
- npm propagation can sometimes take longer
- Check manually: `npm view @lightfastai/dual@VERSION`
- Re-run the workflow if it was a timing issue

**Binary download fails:**
- Ensure GoReleaser job succeeded
- Check GitHub Release has all expected assets
- Verify asset naming matches the expected pattern

## CI/CD Integration

The release workflow is defined in `.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    branches: [main]
    paths:
      - 'npm/package.json'
      - '.changeset/**'
  workflow_dispatch:

concurrency:
  group: release-${{ github.workflow }}
  cancel-in-progress: false
```

This workflow:
- Triggers automatically when `npm/package.json` is modified on `main` branch
- Detects "version packages" commits from Changesets
- Runs five jobs in sequence: `check-release` → `create-tag` → `goreleaser` + `npm-publish` → `verify-release`
- Uses concurrency control to prevent simultaneous releases
- Requires Go 1.23 and Node.js 20
- Requires `LIGHTFAST_RELEASE_BOT_HOMEBREW_TAP_TOKEN` and `LIGHTFAST_RELEASE_BOT_NPM_TOKEN` secrets

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

- **Binaries**: Built with `-s -w` ldflags to strip debug info
- **Static builds**: CGO is disabled for portable binaries
- **Checksums**: SHA256 provided for all GitHub Release artifacts
- **npm Provenance**: Cryptographic attestations link packages to source
- **Secure runners**: All builds happen in GitHub's secure CI environment
- **Token scoping**: Secrets are scoped to minimum required permissions

## Post-Release Verification

After a release is published, verify all channels:

```bash
# Check GitHub Release
gh release view v1.2.3

# Check npm package
npm view @lightfastai/dual@1.2.3

# Check Homebrew formula
curl -fsSL https://raw.githubusercontent.com/lightfastai/homebrew-tap/main/Formula/dual.rb | grep "1.2.3"

# Test installation
npm install -g @lightfastai/dual
dual --version
```

The `verify-release` job in the workflow automatically performs these checks.

## Workflow Refactoring Notes

### October 2024 - Consolidated Release Workflow

The release workflow was refactored to eliminate the manual tag re-push requirement. Key changes:

- **Path-based triggering**: Workflow now triggers on `npm/package.json` changes instead of tag pushes
- **Automatic tag creation**: Git tags are created within the workflow after version detection
- **Single workflow file**: All release steps consolidated into one workflow for better maintainability
- **No manual intervention**: Complete automation from Version Packages PR merge to multi-channel publication

This refactoring improves reliability and reduces the potential for human error in the release process.

## Support

For issues with:
- **Releases**: Check GitHub Actions logs at https://github.com/lightfastai/dual/actions
- **Installation**: See installation instructions in README
- **Homebrew**: Check `lightfastai/homebrew-tap` repository
- **npm**: Check package page at https://www.npmjs.com/package/@lightfastai/dual
