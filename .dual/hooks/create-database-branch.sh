#!/bin/bash
set -e

# Example hook: Create PlanetScale database branch
# Triggered by: postWorktreeCreate

echo "[Hook] Creating database branch for context: $DUAL_CONTEXT_NAME"

# Configuration
DB_NAME="${PLANETSCALE_DB_NAME:-myapp}"
BRANCH_NAME="$DUAL_CONTEXT_NAME"

# Check if pscale CLI is installed
if ! command -v pscale &> /dev/null; then
    echo "[Hook] Warning: pscale CLI not found, skipping database branch creation"
    echo "[Hook] Install from: https://github.com/planetscale/cli"
    exit 0
fi

# Create database branch
echo "[Hook] Creating PlanetScale branch: $BRANCH_NAME"
if pscale branch create "$DB_NAME" "$BRANCH_NAME" --from main 2>/dev/null; then
    echo "[Hook] ✓ Database branch created successfully"
    echo "[Hook] Connect at: https://app.planetscale.com/$DB_NAME/$BRANCH_NAME"
else
    echo "[Hook] Warning: Database branch creation failed (may already exist)"
fi

echo "[Hook] Database setup complete"
