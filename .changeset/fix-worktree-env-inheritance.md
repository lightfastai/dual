---
"@lightfastai/dual": patch
---

fix: load service .env files from both parent repo and worktree

Previously, when running in a git worktree, only the worktree's service .env file was loaded, causing the parent repository's environment variables to be completely skipped. This resulted in missing required environment variables (e.g., only 9 variables loaded instead of the expected 23).

This fix implements proper environment inheritance for worktrees:
- Service .env files are now loaded from both the parent repo (baseline) and the worktree (overrides)
- Worktree-specific values take precedence over parent repo values
- If the worktree's .env is empty or gitignored, parent repo variables are still available
- Non-worktree behavior remains unchanged (backward compatible)

Also fixes the example config to use the correct YAML key (`env.baseFile` instead of `env.base`).
