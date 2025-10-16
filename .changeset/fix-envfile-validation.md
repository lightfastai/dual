---
"@lightfastai/dual": patch
---

Fix config validation blocking dual run when env file directories don't exist

Removes overly strict validation that checked if env file directories exist during config loading. This was causing `dual run` to fail in worktrees where gitignored directories (like `.vercel/`) don't exist yet.

**What's Fixed:**
- `dual run` no longer fails when configured env file directories are missing
- Config validation now only checks that env file paths are relative (not their existence)
- Environment loading gracefully handles missing files by returning empty maps

**Impact:**
- Worktrees with missing env file directories now work correctly
- Fresh checkouts don't require creating all env file directories upfront
- Consistent with the design principle of optional env files
