#!/usr/bin/env bash
# Finalize the dist/pcraft/ release layout from already-built pieces.
# Caller must have run, in this order:
#   - Vite assets synced into apps/backend/internal/webapp/embedded/generated
#   - go build ./cmd/{pcraft,agentctl} -o dist/pcraft/bin/...
# After this: dist/pcraft/bin is ready to install or tar.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BUNDLE="$ROOT_DIR/dist/pcraft"

if [ ! -f "$BUNDLE/bin/pcraft" ] && [ ! -f "$BUNDLE/bin/pcraft.exe" ]; then
  echo "Missing native launcher in $BUNDLE/bin; build cmd/pcraft first" >&2
  exit 1
fi

if [ ! -f "$BUNDLE/bin/agentctl" ] && [ ! -f "$BUNDLE/bin/agentctl.exe" ]; then
  echo "Missing agentctl in $BUNDLE/bin; build cmd/agentctl first" >&2
  exit 1
fi

if [ ! -f "$BUNDLE/bin/agentctl-linux-amd64" ]; then
  echo "Missing linux/amd64 agentctl helper in $BUNDLE/bin; build cmd/agentctl for linux/amd64 first" >&2
  exit 1
fi

echo "Bundle assembled at $BUNDLE"
