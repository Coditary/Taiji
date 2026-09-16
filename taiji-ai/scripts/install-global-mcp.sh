#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGET="${XDG_CONFIG_HOME:-$HOME/.taiji}"

mkdir -p "$TARGET"
cp "$ROOT/global/mcp.json" "$TARGET/mcp.json"
cp "$ROOT/global/config.yaml" "$TARGET/config.yaml"

echo "Installed global Taiji MCP config to $TARGET"
