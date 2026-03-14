#!/usr/bin/env sh
# Sync all go.mod to the version in .go-version.
# Usage: ./scripts/sync-go-version.sh
# After editing .go-version (e.g. to 1.26), run this so you don't forget to update go.mod.

set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION=$(cat "$ROOT/.go-version" | tr -d '[:space:]')
if [ -z "$VERSION" ]; then
  echo "error: .go-version is empty"
  exit 1
fi

echo "Using Go version: $VERSION"

for dir in api gateway mcp matching-engine indexer settlement notification eventlog; do
  if [ -f "$ROOT/$dir/go.mod" ]; then
    # Portable: write updated first line + rest of file
    tmp=$(mktemp)
    sed "s/^go [0-9][0-9.]*$/go $VERSION/" "$ROOT/$dir/go.mod" > "$tmp"
    mv "$tmp" "$ROOT/$dir/go.mod"
    echo "  $dir/go.mod -> go $VERSION"
  fi
done

echo "Done. Commit .go-version and the updated go.mod files."
