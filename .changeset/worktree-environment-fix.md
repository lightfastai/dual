---
"@lightfastai/dual": minor
---

Fix worktree environment access with two-root architecture

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
