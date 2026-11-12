#!/usr/bin/env bash
# scripts/update-test-compose.sh
#
# Downloads the official OpenMetadata docker-compose.yml for a given version,
# replaces the hardcoded image tags with ${OPENMETADATA_VERSION}, and saves the
# result to docker/test/docker-compose.yml.
#
# Usage:
#   ./scripts/update-test-compose.sh <version>
#   ./scripts/update-test-compose.sh           # uses OPENMETADATA_VERSION from .env
#
# Examples:
#   ./scripts/update-test-compose.sh 2.0.3
#   ./scripts/update-test-compose.sh 1.12.4
#
# Run this when upgrading the compose template to pick up structural changes
# in the official file (new services, changed defaults, etc.).
# After running, review the diff and commit docker/test/docker-compose.yml.
#
# Requirements: gh (GitHub CLI), sed, bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${REPO_ROOT}/docker/test/.env"
OUT_FILE="${REPO_ROOT}/docker/test/docker-compose.yml"

command -v gh >/dev/null 2>&1 || { echo "ERROR: gh (GitHub CLI) is required"; exit 1; }

# Resolve version: CLI arg > env var > .env file
if [ -n "${1:-}" ]; then
  VERSION="${1}"
elif [ -n "${OPENMETADATA_VERSION:-}" ]; then
  VERSION="${OPENMETADATA_VERSION}"
elif [ -n "${OM_VERSION:-}" ]; then
  VERSION="${OM_VERSION}"
else
  VERSION="$(grep -E '^OPENMETADATA_VERSION=' "${ENV_FILE}" | cut -d= -f2 | tr -d ' ')"
fi

if [ -z "${VERSION}" ]; then
  echo "ERROR: version not specified. Pass it as an argument or set OPENMETADATA_VERSION."
  echo "Usage: $0 <version>   e.g. $0 2.0.3"
  exit 1
fi

TAG="${VERSION}-release"
TMPFILE="$(mktemp)"

echo "Downloading official docker-compose.yml for OpenMetadata ${VERSION} (tag: ${TAG})..."

gh release download "${TAG}" \
  --repo open-metadata/OpenMetadata \
  --pattern "docker-compose.yml" \
  --output "${TMPFILE}" \
  --clobber

echo "Parameterizing image tags (replacing ${VERSION} → \${OPENMETADATA_VERSION})..."

# Replace the hardcoded version in OM image tags with the variable so a single
# compose file works for all versions.
sed \
  -e "s|openmetadata/db:${VERSION}|openmetadata/db:\${OPENMETADATA_VERSION}|g" \
  -e "s|openmetadata/server:${VERSION}|openmetadata/server:\${OPENMETADATA_VERSION}|g" \
  -e "s|openmetadata/ingestion:${VERSION}|openmetadata/ingestion:\${OPENMETADATA_VERSION}|g" \
  "${TMPFILE}" > "${OUT_FILE}"

rm -f "${TMPFILE}"

echo "Saved to ${OUT_FILE}"
echo ""
echo "Next steps:"
echo "  1. Review:  git diff docker/test/docker-compose.yml"
echo "  2. Commit:  git add docker/test/docker-compose.yml && git commit -m 'chore: refresh compose template from ${VERSION}'"
