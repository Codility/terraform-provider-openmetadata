#!/usr/bin/env bash
# scripts/testacc-all.sh — Run acceptance tests against every supported OM version.
#
# Reads the version list from docker/test/versions (one version per line,
# lines starting with # are comments) and calls scripts/testacc.sh for each.
# All extra arguments are forwarded to go test.
#
# Usage:
#   ./scripts/testacc-all.sh                          # all versions
#   ./scripts/testacc-all.sh -run TestAccTeam         # filter tests, all versions
#
# Each version runs sequentially; the script reports a summary and exits 1 if
# any version fails.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERSIONS_FILE="${REPO_ROOT}/docker/test/versions"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

[ -f "${VERSIONS_FILE}" ] || { echo "ERROR: ${VERSIONS_FILE} not found" >&2; exit 1; }

VERSIONS=()
while IFS= read -r line; do
  [[ "${line}" =~ ^[[:space:]]*# || -z "${line// }" ]] && continue
  VERSIONS+=("${line}")
done < "${VERSIONS_FILE}"

if [ ${#VERSIONS[@]} -eq 0 ]; then
  echo "ERROR: no versions found in ${VERSIONS_FILE}" >&2
  exit 1
fi

echo -e "${BOLD}Testing ${#VERSIONS[@]} OpenMetadata version(s): ${VERSIONS[*]}${NC}"

PASSED=()
FAILED=()

for v in "${VERSIONS[@]}"; do
  echo ""
  echo -e "${BOLD}══════════════════════════════════════════════════${NC}"
  echo -e "${BOLD}  OpenMetadata ${v}${NC}"
  echo -e "${BOLD}══════════════════════════════════════════════════${NC}"

  if OM_VERSION="${v}" bash "${SCRIPT_DIR}/testacc.sh" "$@"; then
    PASSED+=("${v}")
    echo -e "${GREEN}[PASS] ${v}${NC}"
  else
    FAILED+=("${v}")
    echo -e "${RED}[FAIL] ${v}${NC}"
  fi
done

echo ""
echo -e "${BOLD}══════════════════════════════════════════════════${NC}"
echo -e "${BOLD}  Summary${NC}"
echo -e "${BOLD}══════════════════════════════════════════════════${NC}"

for v in "${PASSED[@]+"${PASSED[@]}"}"; do
  echo -e "  ${GREEN}✓ ${v}${NC}"
done
for v in "${FAILED[@]+"${FAILED[@]}"}"; do
  echo -e "  ${RED}✗ ${v}${NC}"
done

if [ ${#FAILED[@]} -gt 0 ]; then
  echo ""
  echo -e "${RED}FAILED: ${FAILED[*]}${NC}"
  exit 1
fi

echo -e "${GREEN}All versions passed.${NC}"
