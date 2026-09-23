#!/usr/bin/env bash
# Sets code=true when a change can affect build/lint/acceptance results.
# Fails open: anything unexpected runs the full suite.
set -uo pipefail

emit() {
  echo "code=$1" >> "$GITHUB_OUTPUT"
  echo "code=$1"
  exit 0
}

# Manual runs always test.
[ "${EVENT:-}" = "workflow_dispatch" ] && emit true

# No usable base (new branch, force push, first commit) — do not guess.
case "${BASE:-}" in
  "" | 0000000000000000000000000000000000000000) emit true ;;
esac

changed="$(git diff --name-only "$BASE" "$HEAD" 2>/dev/null)" || emit true
[ -z "$changed" ] && emit false

echo "Changed files:"
echo "$changed"

if echo "$changed" | grep -qE '^(internal/|tools/|main\.go|go\.(mod|sum)|\.golangci\.yml|docker/test/|scripts/testacc\.sh|\.github/(workflows/|changed-code\.sh))'; then
  emit true
fi

emit false
