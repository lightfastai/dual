---
"@lightfastai/dual": major
---

Release v0.3.0: Worktree lifecycle management with hooks

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
