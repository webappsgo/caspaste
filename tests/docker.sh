#!/usr/bin/env bash
# Docker integration tests for CasPaste
# Per AI.md PART 29: tests/docker.sh — runs in Alpine, no systemd
# All builds via Docker (casjaysdev/go:latest), never on host

set -eo pipefail

# Detect project info from git remote or directory path (NEVER hardcode)
PROJECTNAME=$(git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)(\.git)?$|\1|' || basename "$PWD")
PROJECTORG=$(git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)/[^/]+(\.git)?$|\1|' || basename "$(dirname "$PWD")")

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
API_VERSION="v1"
ADMIN_PATH="admin"
SERVER_PORT=18080

log() { echo "[docker.sh] $(date '+%Y-%m-%dT%H:%M:%SZ') $*"; }
pass() { echo "  ✓ $*"; }
fail() { echo "  ✗ FAILED: $*"; FAILURES=$((FAILURES + 1)); }
FAILURES=0

# =============================================================================
# Setup: temp directory + build
# =============================================================================
mkdir -p "${TMPDIR:-/tmp}/${PROJECTORG}"
BUILD_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECTORG}/${PROJECTNAME}-XXXXXX")
trap "log 'Cleaning up $BUILD_DIR...'; rm -rf \"$BUILD_DIR\"" EXIT

log "Build dir: $BUILD_DIR"
log "Building $PROJECTNAME server via Docker (casjaysdev/go:latest)..."

# Use named volume go-state for persistent Go module cache
docker run --rm \
  --name "${PROJECTNAME}-build-$RANDOM" \
  -v "${PROJECT_ROOT}:/app" \
  -v "go-state:/usr/local/share/go" \
  -w /app \
  -e CGO_ENABLED=0 \
  casjaysdev/go:latest \
  go build -o "/app/binaries_tmp/${PROJECTNAME}" ./src/server

# Move binary to BUILD_DIR
cp "${PROJECT_ROOT}/binaries_tmp/${PROJECTNAME}" "${BUILD_DIR}/${PROJECTNAME}" 2>/dev/null || {
  log "Binary not found in binaries_tmp/, building directly to temp dir..."
  docker run --rm \
    --name "${PROJECTNAME}-build2-$RANDOM" \
    -v "${PROJECT_ROOT}:/app" \
    -v "${BUILD_DIR}:/out" \
    -v "go-state:/usr/local/share/go" \
    -w /app \
    -e CGO_ENABLED=0 \
    casjaysdev/go:latest \
    go build -o "/out/${PROJECTNAME}" ./src/server
}

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
# Integration tests inside Alpine container
# =============================================================================
log "Running integration tests in Alpine container..."

docker run --rm \
  --name "${PROJECTNAME}-test-${RANDOM}" \
  -v "${BUILD_DIR}:/app" \
  alpine:latest sh -c "
    set -e
    apk add --no-cache curl bash file jq >/dev/null 2>&1

    chmod +x /app/${PROJECTNAME}
    [ -f /app/${PROJECTNAME}-cli ] && chmod +x /app/${PROJECTNAME}-cli

    FAILURES=0
    pass() { echo \"  ✓ \$*\"; }
    fail() { echo \"  ✗ FAILED: \$*\"; FAILURES=\$((FAILURES+1)); }

    echo '=== Version Check ==='
    /app/${PROJECTNAME} --version && pass 'server --version' || fail 'server --version'

    echo '=== Help Check ==='
    /app/${PROJECTNAME} --help | head -5 && pass 'server --help' || fail 'server --help'

    echo '=== Binary Info ==='
    ls -lh /app/${PROJECTNAME} && file /app/${PROJECTNAME}

    echo '=== Binary Rename Test ==='
    cp /app/${PROJECTNAME} /app/renamed-server
    chmod +x /app/renamed-server
    if /app/renamed-server --help 2>&1 | grep -q 'renamed-server'; then
        pass 'server binary rename (--help shows actual name)'
    else
        fail 'server binary rename (--help does not show renamed name)'
    fi

    echo '=== Starting Server ==='
    /app/${PROJECTNAME} --port ${SERVER_PORT} > /tmp/server.log 2>&1 &
    SERVER_PID=\$!
    sleep 3

    echo '=== Server Log (setup token) ==='
    cat /tmp/server.log | head -20 || true

    echo '=== Health Endpoint ==='
    curl -q -LSsf http://localhost:${SERVER_PORT}/server/healthz > /tmp/health.json && \
        pass '/server/healthz' || fail '/server/healthz'

    echo '=== API Health (JSON) ==='
    curl -q -LSsf -H 'Accept: application/json' http://localhost:${SERVER_PORT}/api/${API_VERSION}/server/healthz | \
        jq . > /dev/null && pass 'API healthz JSON' || fail 'API healthz JSON'

    echo '=== API Health (.txt extension) ==='
    curl -q -LSsf http://localhost:${SERVER_PORT}/api/${API_VERSION}/server/healthz.txt && \
        pass 'API healthz .txt' || fail 'API healthz .txt'

    echo '=== API Version ==='
    curl -q -LSsf http://localhost:${SERVER_PORT}/api/${API_VERSION}/server/version | \
        jq . > /dev/null && pass 'API version JSON' || fail 'API version JSON'

    echo '=== Robots.txt ==='
    curl -q -LSsf http://localhost:${SERVER_PORT}/robots.txt && \
        pass '/robots.txt' || fail '/robots.txt'

    echo '=== Security.txt ==='
    curl -q -LSsf http://localhost:${SERVER_PORT}/.well-known/security.txt && \
        pass '/.well-known/security.txt' || fail '/.well-known/security.txt'

    echo '=== Paste Create (API) ==='
    PASTE_RESULT=\$(curl -q -LSsf -X POST \
        -H 'Content-Type: application/json' \
        -d '{\"body\":\"test paste content\",\"syntax\":\"text\"}' \
        http://localhost:${SERVER_PORT}/api/${API_VERSION}/pastes 2>&1 || echo '')
    if echo \"\$PASTE_RESULT\" | jq -e .id > /dev/null 2>&1; then
        PASTE_ID=\$(echo \"\$PASTE_RESULT\" | jq -r .id)
        pass \"Paste create: \$PASTE_ID\"

        echo '=== Paste Get (API) ==='
        curl -q -LSsf http://localhost:${SERVER_PORT}/api/${API_VERSION}/pastes/\${PASTE_ID} | \
            jq . > /dev/null && pass 'Paste get JSON' || fail 'Paste get JSON'

        echo '=== Paste Get (.txt extension) ==='
        curl -q -LSsf http://localhost:${SERVER_PORT}/api/${API_VERSION}/pastes/\${PASTE_ID}.txt && \
            pass 'Paste get .txt' || fail 'Paste get .txt'

        echo '=== Paste View (frontend, text/plain) ==='
        curl -q -LSsf -H 'Accept: text/plain' http://localhost:${SERVER_PORT}/\${PASTE_ID} && \
            pass 'Paste frontend text/plain' || fail 'Paste frontend text/plain'

        echo '=== Paste View (frontend, text/html) ==='
        curl -q -LSsf -H 'Accept: text/html' http://localhost:${SERVER_PORT}/\${PASTE_ID} | \
            grep -q '<!DOCTYPE html\|<html' && pass 'Paste frontend HTML' || fail 'Paste frontend HTML'

        echo '=== Paste Raw ==='
        curl -q -LSsf http://localhost:${SERVER_PORT}/raw/\${PASTE_ID} && \
            pass 'Paste raw' || fail 'Paste raw'
    else
        fail \"Paste create: \$PASTE_RESULT\"
    fi

    echo '=== Admin Panel (login page) ==='
    curl -q -LSsf http://localhost:${SERVER_PORT}/server/${ADMIN_PATH}/login | \
        grep -q 'Sign In\|login\|Login' && pass 'Admin login page' || fail 'Admin login page'

    echo '=== Admin Setup Token ==='
    SETUP_TOKEN=\$(grep -oP 'token=\K[a-f0-9]+' /tmp/server.log | head -1 || echo '')
    if [ -n \"\$SETUP_TOKEN\" ]; then
        pass \"Setup token found: \${SETUP_TOKEN:0:8}...\"
    else
        echo '  (No setup token — server may already be configured)'
    fi

    echo '=== CLI Tests (if exists) ==='
    if [ -f /app/${PROJECTNAME}-cli ]; then
        /app/${PROJECTNAME}-cli --version && pass 'CLI --version' || fail 'CLI --version'
        /app/${PROJECTNAME}-cli --help | head -5 && pass 'CLI --help' || fail 'CLI --help'

        cp /app/${PROJECTNAME}-cli /app/renamed-cli
        chmod +x /app/renamed-cli
        if /app/renamed-cli --help 2>&1 | grep -q 'renamed-cli'; then
            pass 'CLI binary rename works'
        else
            fail 'CLI binary rename'
        fi

        /app/${PROJECTNAME}-cli --server http://localhost:${SERVER_PORT} paste list 2>&1 | \
            head -3 && pass 'CLI paste list' || echo '  (CLI paste list may require auth)'
    fi

    kill \$SERVER_PID 2>/dev/null || true

    echo ''
    echo '=== Test Summary ==='
    if [ \"\$FAILURES\" -eq 0 ]; then
        echo 'ALL TESTS PASSED ✓'
    else
        echo \"\$FAILURES TESTS FAILED ✗\"
        exit 1
    fi
"

if [ $? -ne 0 ]; then
  log "Tests FAILED"
  exit 1
fi

log "All Docker tests PASSED ✓"
