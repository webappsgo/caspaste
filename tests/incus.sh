#!/usr/bin/env bash
# Incus integration tests for CasPb (Debian + systemd)
# Per AI.md PART 29: tests/incus.sh — full OS testing preferred
# All builds via Docker (casjaysdev/go:latest), never on host

set -eo pipefail

# Detect project info from git remote or directory path (NEVER hardcode)
PROJECTNAME=$(git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)(\.git)?$|\1|' || basename "$PWD")
PROJECTORG=$(git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)/[^/]+(\.git)?$|\1|' || basename "$(dirname "$PWD")")

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
INCUS_CONTAINER="test-${PROJECTNAME}"
API_VERSION="v1"
ADMIN_PATH="admin"

log() { echo "[incus.sh] $(date '+%Y-%m-%dT%H:%M:%SZ') $*"; }
pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ FAILED: $*"; FAILURES=$((FAILURES + 1)); }
FAILURES=0

# =============================================================================
# Preflight: require incus
# =============================================================================
if ! command -v incus &>/dev/null; then
    log "ERROR: incus not found. Install incus or use tests/docker.sh instead."
    exit 1
fi

# =============================================================================
# Setup: temp directory + build
# =============================================================================
mkdir -p "${TMPDIR:-/tmp}/${PROJECTORG}"
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX")

cleanup() {
    log "Cleaning up..."
    incus delete "$INCUS_CONTAINER" --force 2>/dev/null || true
    rm -rf "$BUILD_DIR"
}
trap cleanup EXIT

log "Build dir: $BUILD_DIR"
log "Building $PROJECTNAME server via Docker (casjaysdev/go:latest)..."

# Build using casjaysdev/go:latest with named volume cache
docker run --rm \
  --name "${PROJECTNAME}-build-$RANDOM" \
  -v "${PROJECT_ROOT}:/app" \
  -v "${BUILD_DIR}:/out" \
  -v "go-state:/usr/local/share/go" \
  -w /app \
  -e CGO_ENABLED=0 \
  casjaysdev/go:latest \
  go build -o "/out/${PROJECTNAME}" ./src/server

# Build CLI if exists
if [ -d "${PROJECT_ROOT}/src/client" ]; then
  log "Building $PROJECTNAME-cli..."
  docker run --rm \
    --name "${PROJECTNAME}-build-cli-$RANDOM" \
    -v "${PROJECT_ROOT}:/app" \
    -v "${BUILD_DIR}:/out" \
    -v "go-state:/usr/local/share/go" \
    -w /app \
    -e CGO_ENABLED=0 \
    casjaysdev/go:latest \
    go build -o "/out/${PROJECTNAME}-cli" ./src/client
fi

# =============================================================================
# Launch Incus container (Debian + systemd)
# =============================================================================
log "Launching Incus container: $INCUS_CONTAINER (debian/trixie)"
incus launch images:debian/trixie "$INCUS_CONTAINER"
sleep 5   # Allow systemd to initialize

# Install dependencies
incus exec "$INCUS_CONTAINER" -- apt-get update -q
incus exec "$INCUS_CONTAINER" -- apt-get install -y -q curl jq file

# =============================================================================
# Push and test binary
# =============================================================================
log "Pushing binary to container..."
incus file push "${BUILD_DIR}/${PROJECTNAME}" "${INCUS_CONTAINER}/usr/local/bin/${PROJECTNAME}"
incus exec "$INCUS_CONTAINER" -- chmod +x "/usr/local/bin/${PROJECTNAME}"

if [ -f "${BUILD_DIR}/${PROJECTNAME}-cli" ]; then
  incus file push "${BUILD_DIR}/${PROJECTNAME}-cli" "${INCUS_CONTAINER}/usr/local/bin/${PROJECTNAME}-cli"
  incus exec "$INCUS_CONTAINER" -- chmod +x "/usr/local/bin/${PROJECTNAME}-cli"
fi

log "=== Version Check ==="
incus exec "$INCUS_CONTAINER" -- "${PROJECTNAME}" --version && \
    pass 'server --version' || fail 'server --version'

log "=== Help Check ==="
incus exec "$INCUS_CONTAINER" -- "${PROJECTNAME}" --help | head -5 && \
    pass 'server --help' || fail 'server --help'

log "=== Binary Info ==="
incus exec "$INCUS_CONTAINER" -- ls -lh "/usr/local/bin/${PROJECTNAME}"
incus exec "$INCUS_CONTAINER" -- file "/usr/local/bin/${PROJECTNAME}"

log "=== Binary Rename Test ==="
incus exec "$INCUS_CONTAINER" -- cp "/usr/local/bin/${PROJECTNAME}" /usr/local/bin/renamed-server
if incus exec "$INCUS_CONTAINER" -- /usr/local/bin/renamed-server --help 2>&1 | grep -q 'renamed-server'; then
    pass 'server binary rename works'
else
    fail 'server binary rename'
fi

log "=== Service Install (--service install) ==="
incus exec "$INCUS_CONTAINER" -- "${PROJECTNAME}" --service install && \
    pass '--service install' || fail '--service install'

log "=== Service Enable/Start ==="
incus exec "$INCUS_CONTAINER" -- systemctl enable "${PROJECTNAME}" 2>/dev/null && \
    pass "systemctl enable $PROJECTNAME" || fail "systemctl enable $PROJECTNAME"
incus exec "$INCUS_CONTAINER" -- systemctl start "${PROJECTNAME}" && \
    pass "systemctl start $PROJECTNAME" || fail "systemctl start $PROJECTNAME"
sleep 3

log "=== Service Status ==="
incus exec "$INCUS_CONTAINER" -- systemctl status "${PROJECTNAME}" && \
    pass "service running" || fail "service not running"

log "=== Health Check via Binary ==="
incus exec "$INCUS_CONTAINER" -- "${PROJECTNAME}" --status && \
    pass '--status' || fail '--status'

log "=== API Health (JSON) ==="
incus exec "$INCUS_CONTAINER" -- curl -q -LSsf \
    -H 'Accept: application/json' \
    "http://localhost:80/api/${API_VERSION}/server/healthz" | \
    jq . > /dev/null && pass 'API healthz JSON' || fail 'API healthz JSON'

log "=== Paste Create ==="
PASTE_RESULT=$(incus exec "$INCUS_CONTAINER" -- curl -q -LSsf -X POST \
    -H 'Content-Type: application/json' \
    -d '{"body":"incus test paste","syntax":"text"}' \
    "http://localhost:80/api/${API_VERSION}/pastes" 2>&1 || echo '')
if echo "$PASTE_RESULT" | jq -e .id > /dev/null 2>&1; then
    PASTE_ID=$(echo "$PASTE_RESULT" | jq -r .id)
    pass "Paste create: $PASTE_ID"

    incus exec "$INCUS_CONTAINER" -- curl -q -LSsf \
        "http://localhost:80/api/${API_VERSION}/pastes/${PASTE_ID}" | \
        jq . > /dev/null && pass 'Paste get JSON' || fail 'Paste get JSON'

    incus exec "$INCUS_CONTAINER" -- curl -q -LSsf \
        "http://localhost:80/api/${API_VERSION}/pastes/${PASTE_ID}.txt" && \
        pass 'Paste get .txt' || fail 'Paste get .txt'
else
    fail "Paste create: $PASTE_RESULT"
fi

log "=== Service Stop ==="
incus exec "$INCUS_CONTAINER" -- systemctl stop "${PROJECTNAME}" && \
    pass "systemctl stop" || fail "systemctl stop"

log "=== Service Uninstall (--service remove) ==="
incus exec "$INCUS_CONTAINER" -- "${PROJECTNAME}" --service remove && \
    pass '--service remove' || fail '--service remove'

log ""
log "=== Test Summary ==="
if [ "$FAILURES" -eq 0 ]; then
    log "ALL TESTS PASSED ✓"
else
    log "$FAILURES TESTS FAILED ✗"
    exit 1
fi
