#!/bin/bash
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Usage: $0 <tag>"
  echo "Example: $0 v0.1.0"
  exit 1
fi

TAG=$1

# Validate tag format
if ! echo "$TAG" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: tag must match v<major>.<minor>.<patch> (e.g. v0.1.0)"
  exit 1
fi

# Check working tree is clean
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is not clean. Commit or stash changes first."
  exit 1
fi

# Create and push tag
git tag -a "$TAG" -m "Release $TAG"
git push origin "$TAG"

echo "Tag $TAG pushed. GitHub Actions will build the release."
