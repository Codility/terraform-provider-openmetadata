#!/usr/bin/env bash
# scripts/update-test-compose.sh
#
# Downloads the official OpenMetadata docker-compose.yml for the version
# pinned in docker/test/.env and saves it to docker/test/docker-compose.yml.
#
# Run this whenever you bump OPENMETADATA_VERSION in docker/test/.env, then
# commit the updated docker-compose.yml alongside the .env change.
#
# Requirements: gh (GitHub CLI), bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${REPO_ROOT}/docker/test/.env"
OUT_DIR="${REPO_ROOT}/docker/test"

command -v gh >/dev/null 2>&1 || { echo "ERROR: gh (GitHub CLI) is required"; exit 1; }

# Read OPENMETADATA_VERSION from .env
VERSION="$(grep -E '^OPENMETADATA_VERSION=' "${ENV_FILE}" | cut -d= -f2 | tr -d ' ')"
if [ -z "${VERSION}" ]; then
  echo "ERROR: OPENMETADATA_VERSION not found in ${ENV_FILE}"
  exit 1
fi

TAG="${VERSION}-release"
echo "Downloading official docker-compose.yml for OpenMetadata ${VERSION} (tag: ${TAG})..."

gh release download "${TAG}" \
  --repo open-metadata/OpenMetadata \
  --pattern "docker-compose.yml" \
  --dir "${OUT_DIR}" \
  --clobber

echo "Saved to ${OUT_DIR}/docker-compose.yml"
echo ""
echo "Next steps:"
echo "  1. Review the diff: git diff docker/test/docker-compose.yml"
echo "  2. Commit: git add docker/test/docker-compose.yml docker/test/.env && git commit"
