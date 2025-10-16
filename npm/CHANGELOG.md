# @lightfastai/dual

## 1.2.1

### Patch Changes

- a91c07a: Fix config validation blocking dual run when env file directories don't exist

  Removes overly strict validation that checked if env file directories exist during config loading. This was causing `dual run` to fail in worktrees where gitignored directories (like `.vercel/`) don't exist yet.

  **What's Fixed:**

  - `dual run` no longer fails when configured env file directories are missing
  - Config validation now only checks that env file paths are relative (not their existence)
  - Environment loading gracefully handles missing files by returning empty maps

  **Impact:**

  - Worktrees with missing env file directories now work correctly
  - Fresh checkouts don't require creating all env file directories upfront
  - Consistent with the design principle of optional env files

## 1.2.0

### Minor Changes

- eb29975: Fix worktree environment access with two-root architecture

  Implements the correct two-root architecture for worktree support where registry and environment overrides are stored in the parent repository and shared across all worktrees.

  **Breaking Change:** Environment override files now stored in parent repo's `.dual/.local/service/<service>/.env` instead of per-worktree. Existing worktrees with isolated override files will need to migrate (see worktree troubleshooting in docs).

  **What's Fixed:**

  - Worktrees can now properly load config/env files from parent repository
  - Environment overrides are shared across all worktrees (as intended)
  - Registry is correctly accessed from parent repo in all commands
  - All integration tests updated and passing

  **What's New:**

  - Comprehensive worktree architecture documentation in CLAUDE.md
  - Worktree troubleshooting section with practical debugging guidance
  - Updated examples/env-remapping with ARCHITECTURE.md guide
  - Fixed README.md hook examples to use `dual env set`

  **Migration:**
  If you have existing worktrees with environment overrides, they may be in the wrong location. Run `dual doctor` to diagnose and see the troubleshooting section in CLAUDE.md for migration steps.

## 1.1.0

### Minor Changes

- 01dc6dc: feat: major improvements to environment handling, error UX, and dotenv compatibility

  This release includes significant enhancements from multiple PRs:

  ## PR #77: Full Dotenv Compatibility

  Replaces the custom .env file parser with the industry-standard godotenv library, adding full compatibility with Node.js dotenv features:

  - **Multiline values**: Support for certificates, keys, and formatted text using quotes
  - **Variable expansion**: `${VAR}` and `$VAR` syntax for DRY configuration
  - **Escape sequences**: Process `\n`, `\\`, `\"` within double-quoted strings
  - **Inline comments**: Support `KEY=value # comment` syntax
  - **Complex quoting**: Handle nested and mixed quotes properly

  ## PR #82: Unified Environment Loading

  Fixes critical bugs and unifies environment loading implementation:

  - **Fixed**: Environment variables now properly load from service .env files
  - **Fixed**: Base environment configuration now correctly recognized
  - **Unified**: All environment loading now goes through consistent `LoadLayeredEnv()` function
  - **Improved**: Consistent behavior across all commands (`env show`, `run`, etc.)

  ## PR #73: Enhanced Error Handling & UX

  Improves error messages with actionable user guidance:

  - **Better error messages**: Clear, actionable hints for common issues
  - **Improved diagnostics**: More detailed information when things go wrong
  - **User-friendly output**: Helpful suggestions for fixing configuration problems
  - **Better validation**: Early detection of configuration issues

  Breaking changes:

  - Variable expansion is now enabled by default (previously literal values)
  - Escape sequences like `\n` are now processed in double quotes (previously literal)
  - Inline comments after values are now stripped (previously included in value)

  Migration guide:

  - Use single quotes for literal `${VAR}` values: `'${BASE_URL}/api'`
  - Use single quotes or escape backslash for literal `\n`: `'Hello\nWorld'` or `"Hello\\nWorld"`
  - Remove inline comments or quote values containing `#`: `"value#notacomment"`

## 1.0.0

### Major Changes

- 246bd0f: Release v0.3.0: Worktree lifecycle management with hooks

  BREAKING CHANGES:

  - Removed automatic port management (implement in hooks if needed)
  - Removed command wrapper mode (`dual <command>`)
  - Removed `dual port` and `dual ports` commands
  - Removed `dual context` command (replaced with `dual list`)
  - Moved registry from global `~/.dual/` to project-local `.dual/.local/`

  NEW FEATURES:

  - Worktree lifecycle management (`dual create`/`dual delete`)
  - Hook system with 3 lifecycle events
  - Environment remapping via hook outputs
  - Project-local registry at `.dual/.local/registry.json`
  - Simplified command structure

  See release notes and migration guide for full details.

## 0.2.2

### Patch Changes

- 35ba33e: Test automated release workflow after consolidation refactoring. This patch validates that the new path-based trigger system correctly automates tag creation and multi-channel publishing without manual intervention.

## 0.2.1

### Patch Changes

- ecb9c90: Add npm package badges to README

  Display npm version and download count badges in the README to make npm installation more prominent and show package popularity.

## 0.2.0

### Minor Changes

- ec2d79a: Enable npm package distribution with automated multi-channel releases

  The dual CLI is now available via npm (`npm install -g @lightfastai/dual`) in addition to GitHub Releases and Homebrew. The npm wrapper automatically downloads the correct native binary for your platform during installation.

  This release also introduces fully automated release management using Changesets:

  - Automatic version bumping based on semantic versioning
  - Auto-generated CHANGELOGs from changeset descriptions
  - "Version Packages" PRs that preview all changes before release
  - One-click releases across GitHub, Homebrew, and npm
  - Post-release verification of all distribution channels

  Contributors can now document changes by running `npm run changeset` when making PRs. See CONTRIBUTING.md for the complete workflow.
