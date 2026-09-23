#!/usr/bin/env bash
# scripts/testacc.sh — Local acceptance test runner.
#
# Starts the OpenMetadata test stack (official docker-compose + env overrides),
# acquires a JWT token, runs all TF_ACC acceptance tests, then tears down.
#
# Usage:
#   ./scripts/testacc.sh [go test flags]
#
# Examples:
#   ./scripts/testacc.sh                           # run all acceptance tests
#   ./scripts/testacc.sh -run TestAccClassification    # run one test
#   ./scripts/testacc.sh -v -timeout 30m           # verbose with custom timeout
#
# Prerequisites: docker (compose v2), go, python3, curl
#
# Environment overrides:
#   OPENMETADATA_VERSION  Image tag (read from docker/test/.env by default)
#   OM_ADMIN_EMAIL        Admin email   (default: admin@open-metadata.org)
#   OM_ADMIN_PASSWORD     Admin password (default: admin)
#   OM_HOST               OM API base URL (default: http://localhost:8585)
#   SKIP_DOCKER           Set to 1 to skip compose lifecycle (use existing OM)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
COMPOSE_FILE="${REPO_ROOT}/docker/test/docker-compose.yml"
ENV_FILE="${REPO_ROOT}/docker/test/.env"

OM_HOST="${OM_HOST:-http://localhost:8585}"
# OM exposes its health check on port 8586 (/healthcheck), not 8585.
OM_HEALTH_URL="${OM_HEALTH_URL:-http://localhost:8586/healthcheck}"
OM_ADMIN_EMAIL="${OM_ADMIN_EMAIL:-admin@open-metadata.org}"
OM_ADMIN_PASSWORD="${OM_ADMIN_PASSWORD:-admin}"
SKIP_DOCKER="${SKIP_DOCKER:-0}"

# Services to start — we intentionally skip the heavy `ingestion` (Airflow)
# service because PIPELINE_SERVICE_CLIENT_ENABLED=false makes it unnecessary.
OM_SERVICES="mysql elasticsearch execute-migrate-all openmetadata-server"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[testacc]${NC} $*"; }
warn() { echo -e "${YELLOW}[testacc]${NC} $*"; }
die()  { echo -e "${RED}[testacc] ERROR:${NC} $*" >&2; exit 1; }

# ── prerequisites ─────────────────────────────────────────────────────────────

command -v docker   >/dev/null 2>&1 || die "docker is not installed"
command -v python3  >/dev/null 2>&1 || die "python3 is not installed"
command -v curl     >/dev/null 2>&1 || die "curl is not installed"
command -v go       >/dev/null 2>&1 || die "go is not installed"
docker compose version >/dev/null 2>&1 || die "docker compose (v2) is not available"

[ -f "${COMPOSE_FILE}" ] || die "docker-compose.yml not found at ${COMPOSE_FILE}. Run: scripts/update-test-compose.sh"
[ -f "${ENV_FILE}" ]     || die ".env not found at ${ENV_FILE}"

# ── docker compose lifecycle ───────────────────────────────────────────────────

cleanup() {
  if [ "${SKIP_DOCKER}" = "0" ]; then
    log "Tearing down test stack..."
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" \
      down -v --remove-orphans 2>/dev/null || true
  fi
}
trap cleanup EXIT

if [ "${SKIP_DOCKER}" = "0" ]; then
  log "Pulling images (this may take a while on first run)..."
  docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" pull --quiet ${OM_SERVICES}

  log "Starting OpenMetadata test stack..."
  log "  (DB migration runs first — expect 3-7 min on first boot)"
  docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" \
    up -d ${OM_SERVICES}
fi

# ── wait for OpenMetadata ─────────────────────────────────────────────────────

log "Waiting for OpenMetadata health check at ${OM_HEALTH_URL} ..."
MAX_WAIT=480
INTERVAL=10
ELAPSED=0

until curl -sf "${OM_HEALTH_URL}" >/dev/null 2>&1; do
  if [ "${ELAPSED}" -ge "${MAX_WAIT}" ]; then
    echo ""
    echo "OpenMetadata did not become healthy within ${MAX_WAIT}s."
    echo "Server logs (last 50 lines):"
    docker compose -f "${COMPOSE_FILE}" --env-file "${ENV_FILE}" \
      logs openmetadata-server --tail=50
    exit 1
  fi
  echo "  ... still waiting (${ELAPSED}s / ${MAX_WAIT}s)"
  sleep "${INTERVAL}"
  ELAPSED=$((ELAPSED + INTERVAL))
done

log "OpenMetadata is healthy."

# ── acquire JWT token ─────────────────────────────────────────────────────────

log "Acquiring JWT token for ${OM_ADMIN_EMAIL} ..."

OM_ADMIN_PASSWORD_B64="$(echo -n "${OM_ADMIN_PASSWORD}" | base64)"

LOGIN_PAYLOAD_FILE="$(mktemp)"
printf '{"email":"%s","password":"%s"}' "${OM_ADMIN_EMAIL}" "${OM_ADMIN_PASSWORD_B64}" > "${LOGIN_PAYLOAD_FILE}"

LOGIN_RESPONSE="$(curl -sf -X POST \
  "${OM_HOST}/api/v1/users/login" \
  -H "Content-Type: application/json" \
  --data-binary "@${LOGIN_PAYLOAD_FILE}" 2>&1)" || {
    rm -f "${LOGIN_PAYLOAD_FILE}"
    die "Login request failed. Is the admin password correct? (OM_ADMIN_PASSWORD=${OM_ADMIN_PASSWORD})"
  }
rm -f "${LOGIN_PAYLOAD_FILE}"

OM_TOKEN="$(echo "${LOGIN_RESPONSE}" | python3 -c "
import sys, json
try:
    d = json.load(sys.stdin)
    tok = d.get('accessToken') or d.get('token') or ''
    if not tok:
        sys.stderr.write('No token in login response: ' + repr(d)[:400] + '\n')
    print(tok, end='')
except Exception as e:
    sys.stderr.write('Failed to parse login response: ' + str(e) + '\n')
    print('', end='')
")"

if [ -z "${OM_TOKEN}" ]; then
  die "Could not extract JWT token. Raw response: ${LOGIN_RESPONSE:0:400}"
fi

log "JWT token acquired (length: ${#OM_TOKEN})."

# ── run acceptance tests ───────────────────────────────────────────────────────

log "Running acceptance tests..."
log "  OPENMETADATA_HOST  = ${OM_HOST}"
log "  TF_ACC             = 1"

cd "${REPO_ROOT}"
TF_ACC=1 \
  OPENMETADATA_HOST="${OM_HOST}" \
  OPENMETADATA_TOKEN="${OM_TOKEN}" \
  go test \
    -v \
    -count=1 \
    -timeout 30m \
    ./internal/provider/... \
    "$@"
