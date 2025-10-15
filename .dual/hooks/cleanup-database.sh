#!/bin/bash
set -e

# Example hook: Cleanup database branch before worktree deletion
# Triggered by: preWorktreeDelete

echo "[Hook] Cleaning up database for context: $DUAL_CONTEXT_NAME"

# Configuration
DB_NAME="${PLANETSCALE_DB_NAME:-myapp}"
BRANCH_NAME="$DUAL_CONTEXT_NAME"

# Don't delete main/master branches
if [[ "$BRANCH_NAME" == "main" ]] || [[ "$BRANCH_NAME" == "master" ]]; then
    echo "[Hook] Skipping cleanup for protected branch: $BRANCH_NAME"
    exit 0
fi

# Check if pscale CLI is installed
if ! command -v pscale &> /dev/null; then
    echo "[Hook] Warning: pscale CLI not found, skipping database cleanup"
    exit 0
fi

# Ask for confirmation (optional - can be automated)
echo "[Hook] About to delete database branch: $BRANCH_NAME"
read -p "[Hook] Continue? (y/N): " -n 1 -r
echo

if [[ $REPLY =~ ^[Yy]$ ]]; then
    if pscale branch delete "$DB_NAME" "$BRANCH_NAME" --force 2>/dev/null; then
        echo "[Hook] ✓ Database branch deleted successfully"
    else
        echo "[Hook] Warning: Database branch deletion failed (may not exist)"
    fi
else
    echo "[Hook] Database cleanup skipped"
fi
