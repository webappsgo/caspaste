#!/bin/bash
# CasPaste Comprehensive Test Suite
# This script builds and tests all functionality using a temp directory
# Usage: ./test/run_tests.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
GODIR="${GODIR:-$HOME/.local/share/go}"
CONTAINER_NAME="caspaste-test-$$"
SERVER_PORT=18080

# Parse arguments
case "${1:-}" in
    -h|--help)
        echo "CasPaste Test Suite"
        echo ""
        echo "Usage: $0"
        echo ""
        echo "Runs comprehensive tests for CasPaste including:"
        echo "  - Go unit tests"
        echo "  - Binary builds"
        echo "  - Server startup"
        echo "  - API endpoints"
        echo "  - CLI functionality"
        echo "  - Content-type validation"
        echo "  - Security tests"
        exit 0
        ;;
esac

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Create temp directory per AI.md PART 29
mkdir -p "${TMPDIR:-/tmp}/casjay-forks"
TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/casjay-forks/caspaste-XXXXXX")
mkdir -p "$TEMP_DIR"/{data,config,backups,binaries}

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}CasPaste Comprehensive Test Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Project: $PROJECT_DIR"
echo -e "Temp Dir: $TEMP_DIR"
echo -e "Go Dir: $GODIR"
echo ""

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    docker stop "$CONTAINER_NAME" 2>/dev/null || true
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
    rm -rf "$TEMP_DIR"
    echo "Removed temp directory: $TEMP_DIR"
}
trap cleanup EXIT

# Test helper functions
log_test() {
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    echo -e "${BLUE}[TEST $TESTS_TOTAL]${NC} $1"
}

pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    echo -e "  ${GREEN}PASS${NC}: $1"
}

fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo -e "  ${RED}FAIL${NC}: $1"
}

assert_eq() {
    if [ "$1" = "$2" ]; then
        pass "$3"
    else
        fail "$3 (expected '$2', got '$1')"
    fi
}

assert_contains() {
    if echo "$1" | grep -q "$2"; then
        pass "$3"
    else
        fail "$3 (expected to contain '$2')"
    fi
}

assert_not_contains() {
    if ! echo "$1" | grep -q "$2"; then
        pass "$3"
    else
        fail "$3 (expected NOT to contain '$2')"
    fi
}

assert_http_code() {
    local url="$1"
    local expected="$2"
    local desc="$3"
    local actual=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
    assert_eq "$actual" "$expected" "$desc"
}

# ============================================
# SECTION 1: BUILD TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 1: BUILD TESTS ===${NC}\n"

log_test "Go Unit Tests"
cd "$PROJECT_DIR"
mkdir -p "$GODIR/build" "$GODIR/pkg/mod"
if docker run --rm \
    -v "$PROJECT_DIR":/build \
    -v "$GODIR":/go \
    -w /build \
    -e CGO_ENABLED=0 \
    -e GOCACHE=/go/build \
    -e GOMODCACHE=/go/pkg/mod \
    golang:alpine sh -c 'go test ./... 2>&1' > "$TEMP_DIR/test-output.log" 2>&1; then
    pass "All Go unit tests passed"
else
    fail "Go unit tests failed (see $TEMP_DIR/test-output.log)"
fi

log_test "Build Server Binary"
if docker run --rm \
    -v "$PROJECT_DIR":/build \
    -v "$GODIR":/go \
    -v "$TEMP_DIR/binaries":/out \
    -w /build \
    -e CGO_ENABLED=0 \
    -e GOCACHE=/go/build \
    -e GOMODCACHE=/go/pkg/mod \
    golang:alpine sh -c 'go build -trimpath -tags netgo -ldflags "-w -s -X main.Version=test" -o /out/caspaste ./src/server' 2>&1; then
    pass "Server binary built successfully"
else
    fail "Failed to build server binary"
    exit 1
fi

log_test "Build CLI Binary"
if docker run --rm \
    -v "$PROJECT_DIR":/build \
    -v "$GODIR":/go \
    -v "$TEMP_DIR/binaries":/out \
    -w /build \
    -e CGO_ENABLED=0 \
    -e GOCACHE=/go/build \
    -e GOMODCACHE=/go/pkg/mod \
    golang:alpine sh -c 'go build -trimpath -tags netgo -ldflags "-w -s -X main.Version=test" -o /out/caspaste-cli ./src/client' 2>&1; then
    pass "CLI binary built successfully"
else
    fail "Failed to build CLI binary"
    exit 1
fi

log_test "Verify binaries exist"
if [ -f "$TEMP_DIR/binaries/caspaste" ] && [ -f "$TEMP_DIR/binaries/caspaste-cli" ]; then
    pass "Both binaries exist"
else
    fail "Missing binaries"
    exit 1
fi

# ============================================
# SECTION 2: SERVER STARTUP TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 2: SERVER STARTUP TESTS ===${NC}\n"

log_test "Start server container"
# Create DB directory structure matching Dockerfile
mkdir -p "$TEMP_DIR/data/db/sqlite"
docker run -d --name "$CONTAINER_NAME" \
    -v "$TEMP_DIR/binaries/caspaste:/usr/local/bin/caspaste" \
    -v "$TEMP_DIR/data:/data" \
    -v "$TEMP_DIR/config:/config" \
    -e CASPASTE_PUBLIC=true \
    -e CASPASTE_DATA_DIR=/data \
    -e CASPASTE_CONFIG_DIR=/config \
    -e CASPASTE_DB_DIR=/data/db/sqlite \
    -e CASPASTE_DB_DRIVER=sqlite \
    -e TZ=America/New_York \
    -p $SERVER_PORT:80 \
    alpine:latest sh -c 'apk add --no-cache tzdata >/dev/null 2>&1 && /usr/local/bin/caspaste --port 80 2>&1' >/dev/null 2>&1

if [ $? -eq 0 ]; then
    pass "Container started"
else
    fail "Failed to start container"
    exit 1
fi

# Wait for server to be ready
log_test "Wait for server startup"
for i in {1..30}; do
    if curl -s "http://localhost:$SERVER_PORT/api/healthz" >/dev/null 2>&1; then
        pass "Server is ready (took ${i}s)"
        break
    fi
    if [ $i -eq 30 ]; then
        fail "Server failed to start within 30s"
        docker logs "$CONTAINER_NAME"
        exit 1
    fi
    sleep 1
done

log_test "Check startup logs for timezone warning"
LOGS=$(docker logs "$CONTAINER_NAME" 2>&1)
assert_not_contains "$LOGS" "invalid timezone" "No timezone warning with tzdata installed"

log_test "Verify server banner"
assert_contains "$LOGS" "CasPaste" "Server banner displayed"
assert_contains "$LOGS" "Status:      Ready" "Server status is Ready"

# ============================================
# SECTION 3: API ENDPOINT TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 3: API ENDPOINT TESTS ===${NC}\n"

BASE_URL="http://localhost:$SERVER_PORT"

log_test "Health endpoint"
HEALTH=$(curl -s "$BASE_URL/api/healthz")
assert_contains "$HEALTH" '"status":"healthy"' "Health status is healthy"

log_test "Server info endpoint"
INFO=$(curl -s "$BASE_URL/api/v1/getServerInfo")
assert_contains "$INFO" '"software":"CasPaste"' "Software name in response"
assert_contains "$INFO" '"titleMaxlength":100' "Title max length in response"
assert_contains "$INFO" '"bodyMaxlength":52428800' "Body max length in response"
assert_contains "$INFO" '"syntaxes":\[' "Syntaxes array in response"

log_test "Create paste (basic)"
PASTE1=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=Hello World Test" \
    --data-urlencode "syntax=plaintext")
assert_contains "$PASTE1" '"id":' "Paste ID returned"
assert_contains "$PASTE1" '"url":' "Paste URL returned"
PASTE1_ID=$(echo "$PASTE1" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

log_test "Get paste"
PASTE1_GET=$(curl -s "$BASE_URL/api/v1/get?id=$PASTE1_ID")
assert_contains "$PASTE1_GET" '"body":"Hello World Test"' "Paste body matches"
assert_contains "$PASTE1_GET" '"syntax":"plaintext"' "Paste syntax matches"

log_test "Create paste with title"
PASTE2=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "title=My Test Title" \
    --data-urlencode "body=Test content" \
    --data-urlencode "syntax=plaintext")
PASTE2_ID=$(echo "$PASTE2" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
PASTE2_GET=$(curl -s "$BASE_URL/api/v1/get?id=$PASTE2_ID")
assert_contains "$PASTE2_GET" '"title":"My Test Title"' "Paste title matches"

log_test "Create paste with expiration"
PASTE3=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=Expiring paste" \
    --data-urlencode "syntax=plaintext" \
    --data-urlencode "expiration=600")
assert_contains "$PASTE3" '"deleteTime":' "Delete time returned"
DELETE_TIME=$(echo "$PASTE3" | grep -o '"deleteTime":[0-9]*' | cut -d':' -f2)
if [ "$DELETE_TIME" -gt 0 ]; then
    pass "Delete time is set correctly"
else
    fail "Delete time should be > 0"
fi

log_test "Syntax case-insensitivity (lowercase python)"
PASTE4=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=print('hello')" \
    --data-urlencode "syntax=python")
assert_contains "$PASTE4" '"id":' "Paste created with lowercase python"
PASTE4_ID=$(echo "$PASTE4" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
PASTE4_GET=$(curl -s "$BASE_URL/api/v1/get?id=$PASTE4_ID")
assert_contains "$PASTE4_GET" '"syntax":"Python"' "Syntax normalized to Python"

log_test "Syntax case-insensitivity (uppercase JAVASCRIPT)"
PASTE5=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=console.log('test')" \
    --data-urlencode "syntax=JAVASCRIPT")
PASTE5_ID=$(echo "$PASTE5" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
PASTE5_GET=$(curl -s "$BASE_URL/api/v1/get?id=$PASTE5_ID")
assert_contains "$PASTE5_GET" '"syntax":"JavaScript"' "Syntax normalized to JavaScript"

log_test "One-use paste (burn after reading)"
PASTE6=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=Secret message" \
    --data-urlencode "syntax=plaintext" \
    --data-urlencode "oneUse=true")
PASTE6_ID=$(echo "$PASTE6" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
# First read should succeed
PASTE6_GET1=$(curl -s "$BASE_URL/api/v1/get?id=$PASTE6_ID")
assert_contains "$PASTE6_GET1" '"oneUse":true' "First read shows oneUse=true"
# Second read should fail
PASTE6_GET2=$(curl -s "$BASE_URL/api/v1/get?id=$PASTE6_ID")
assert_contains "$PASTE6_GET2" '"code":404' "Second read returns 404"

log_test "List pastes"
LIST=$(curl -s "$BASE_URL/api/v1/list")
assert_contains "$LIST" '"id":' "List contains paste IDs"

log_test "Non-existent paste returns 404"
NOT_FOUND=$(curl -s "$BASE_URL/api/v1/get?id=nonexistent123")
assert_contains "$NOT_FOUND" '"code":404' "404 for non-existent paste"

log_test "Invalid syntax returns 400"
INVALID_SYNTAX=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=test" \
    --data-urlencode "syntax=invalid_syntax_xyz")
assert_contains "$INVALID_SYNTAX" '"code":400' "400 for invalid syntax"

log_test "Empty body returns 400"
EMPTY_BODY=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "title=Empty" \
    --data-urlencode "syntax=plaintext")
assert_contains "$EMPTY_BODY" '"code":400' "400 for empty body"

log_test "File upload"
echo "Test file content" > "$TEMP_DIR/test-upload.txt"
UPLOAD=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    -F "file=@$TEMP_DIR/test-upload.txt")
assert_contains "$UPLOAD" '"id":' "File upload returns paste ID"
UPLOAD_ID=$(echo "$UPLOAD" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
UPLOAD_GET=$(curl -s "$BASE_URL/api/v1/get?id=$UPLOAD_ID")
assert_contains "$UPLOAD_GET" '"isFile":true' "File marked as isFile"
assert_contains "$UPLOAD_GET" '"syntax":"plaintext"' "File defaults to plaintext syntax"

log_test "XSS prevention in author URL"
XSS_TEST=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=test" \
    --data-urlencode "syntax=plaintext" \
    --data-urlencode "authorURL=javascript:alert('xss')")
assert_contains "$XSS_TEST" '"code":400' "400 for javascript: URL"

log_test "Valid author URL accepted"
VALID_URL=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=test" \
    --data-urlencode "syntax=plaintext" \
    --data-urlencode "author=Test User" \
    --data-urlencode "authorURL=https://example.com")
assert_contains "$VALID_URL" '"id":' "Valid author URL accepted"

log_test "Unicode content"
UNICODE=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "title=日本語タイトル" \
    --data-urlencode "body=Hello 世界! Привет мир!" \
    --data-urlencode "syntax=plaintext")
UNICODE_ID=$(echo "$UNICODE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
UNICODE_GET=$(curl -s "$BASE_URL/api/v1/get?id=$UNICODE_ID")
assert_contains "$UNICODE_GET" '日本語' "Unicode title preserved"
assert_contains "$UNICODE_GET" '世界' "Unicode body preserved"

log_test "URL shortener"
URL_SHORT=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "url=true" \
    --data-urlencode "originalURL=https://example.com/long/path" \
    --data-urlencode "body=placeholder")
assert_contains "$URL_SHORT" '"id":' "URL shortener creates paste"

# ============================================
# SECTION 4: WEB UI TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 4: WEB UI TESTS ===${NC}\n"

log_test "Homepage"
assert_http_code "$BASE_URL/" "200" "Homepage returns 200"

log_test "About page"
assert_http_code "$BASE_URL/about" "200" "About page returns 200"

log_test "About/authors"
assert_http_code "$BASE_URL/about/authors" "200" "Authors page returns 200"

log_test "About/license"
assert_http_code "$BASE_URL/about/license" "200" "License page returns 200"

log_test "About/source_code"
assert_http_code "$BASE_URL/about/source_code" "200" "Source code page returns 200"

log_test "Settings page"
assert_http_code "$BASE_URL/settings" "200" "Settings page returns 200"

log_test "Terms page"
assert_http_code "$BASE_URL/terms" "200" "Terms page returns 200"

log_test "Docs page"
assert_http_code "$BASE_URL/docs" "200" "Docs page returns 200"

log_test "API docs"
assert_http_code "$BASE_URL/docs/apiv1" "200" "API docs returns 200"

log_test "Login page"
assert_http_code "$BASE_URL/login" "200" "Login page returns 200"

log_test "CSS file"
assert_http_code "$BASE_URL/style.css" "200" "CSS file returns 200"

log_test "JavaScript file"
assert_http_code "$BASE_URL/main.js" "200" "JS file returns 200"

log_test "Manifest file"
assert_http_code "$BASE_URL/manifest.json" "200" "Manifest returns 200"

log_test "Raw paste endpoint"
RAW=$(curl -s "$BASE_URL/raw/$PASTE1_ID")
assert_eq "$RAW" "Hello World Test" "Raw paste content correct"

log_test "Paste view page"
assert_http_code "$BASE_URL/$PASTE1_ID" "200" "Paste view returns 200"

# ============================================
# SECTION 5: CLI TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 5: CLI TESTS ===${NC}\n"

cli_run() {
    docker run --rm \
        -v "$TEMP_DIR/binaries/caspaste-cli:/usr/local/bin/caspaste-cli" \
        --network host \
        -e CASPASTE_SERVER="http://localhost:$SERVER_PORT" \
        alpine:latest /usr/local/bin/caspaste-cli "$@" 2>&1 || true
}

log_test "CLI version"
CLI_VERSION=$(cli_run version)
assert_contains "$CLI_VERSION" "caspaste-cli" "CLI version output"

log_test "CLI help"
CLI_HELP=$(cli_run help)
assert_contains "$CLI_HELP" "CasPaste CLI" "CLI help output"

log_test "CLI health"
CLI_HEALTH=$(cli_run health)
assert_contains "$CLI_HEALTH" "healthy" "CLI health check"

log_test "CLI server info"
CLI_INFO=$(cli_run info)
assert_contains "$CLI_INFO" "Title Max Length: 100" "CLI shows correct title limit"
assert_contains "$CLI_INFO" "Body Max Length:" "CLI shows body limit"
assert_contains "$CLI_INFO" "50.0 MB" "CLI shows correct body size"

log_test "CLI create paste"
CLI_NEW=$(docker run --rm \
    -v "$TEMP_DIR/binaries/caspaste-cli:/usr/local/bin/caspaste-cli" \
    --network host \
    -e CASPASTE_SERVER="http://localhost:$SERVER_PORT" \
    alpine:latest sh -c 'echo "CLI test paste" | /usr/local/bin/caspaste-cli new' 2>&1 || true)
assert_contains "$CLI_NEW" "Paste created" "CLI creates paste"
assert_contains "$CLI_NEW" "ID:" "CLI shows paste ID"

log_test "CLI list pastes"
CLI_LIST=$(cli_run list)
assert_contains "$CLI_LIST" "ID" "CLI list shows header"

# ============================================
# SECTION 6: DATABASE TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 6: DATABASE TESTS ===${NC}\n"

log_test "Verify SQLite database created"
# In Docker, database is at /data/db/sqlite/caspaste.db (CASPASTE_DB_DIR)
# SQLite also serves as cache/backup when using PostgreSQL/MySQL
if docker exec "$CONTAINER_NAME" ls /data/db/sqlite/caspaste.db >/dev/null 2>&1 || \
   docker exec "$CONTAINER_NAME" ls /data/caspaste.db >/dev/null 2>&1; then
    pass "SQLite database file exists"
else
    fail "SQLite database file not found"
fi

log_test "Paste expiration cleanup"
# Create a paste that expires in 1 second
EXPIRE_PASTE=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=Expiring soon" \
    --data-urlencode "syntax=plaintext" \
    --data-urlencode "expiration=1")
EXPIRE_ID=$(echo "$EXPIRE_PASTE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
# Wait for cleanup (default cleanup period is 5m but paste should be gone on next access after expiry)
sleep 3
EXPIRE_GET=$(curl -s "$BASE_URL/api/v1/get?id=$EXPIRE_ID")
assert_contains "$EXPIRE_GET" '"code":404' "Expired paste is deleted"

# ============================================
# SECTION 7: CONTENT-TYPE TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 7: CONTENT-TYPE TESTS ===${NC}\n"

assert_content_type() {
    local url="$1"
    local expected="$2"
    local desc="$3"
    # Use -D - to get headers from actual response (not HEAD which may not work for all endpoints)
    local actual=$(curl -s -D - -o /dev/null "$url" 2>/dev/null | grep -i "^content-type:" | tr -d '\r' | cut -d' ' -f2-)
    if echo "$actual" | grep -qi "$expected"; then
        pass "$desc"
    else
        fail "$desc (expected '$expected', got '$actual')"
    fi
}

# Frontend HTML pages
log_test "Frontend Content-Type: Homepage"
assert_content_type "$BASE_URL/" "text/html" "Homepage returns text/html"

log_test "Frontend Content-Type: About page"
assert_content_type "$BASE_URL/about" "text/html" "About page returns text/html"

log_test "Frontend Content-Type: Settings page"
assert_content_type "$BASE_URL/settings" "text/html" "Settings page returns text/html"

log_test "Frontend Content-Type: Login page"
assert_content_type "$BASE_URL/login" "text/html" "Login page returns text/html"

log_test "Frontend Content-Type: Docs page"
assert_content_type "$BASE_URL/docs" "text/html" "Docs page returns text/html"

log_test "Frontend Content-Type: Terms page"
assert_content_type "$BASE_URL/terms" "text/html" "Terms page returns text/html"

log_test "Frontend Content-Type: Paste view"
assert_content_type "$BASE_URL/$PASTE1_ID" "text/html" "Paste view returns text/html"

# Raw paste endpoint (text/plain)
log_test "Raw paste Content-Type"
assert_content_type "$BASE_URL/raw/$PASTE1_ID" "text/plain" "Raw paste returns text/plain"

# API endpoints (application/json)
log_test "API Content-Type: Health endpoint"
assert_content_type "$BASE_URL/api/healthz" "application/json" "Health API returns application/json"

log_test "API Content-Type: Server info"
assert_content_type "$BASE_URL/api/v1/getServerInfo" "application/json" "Server info API returns application/json"

log_test "API Content-Type: Get paste"
assert_content_type "$BASE_URL/api/v1/get?id=$PASTE1_ID" "application/json" "Get paste API returns application/json"

log_test "API Content-Type: List pastes"
assert_content_type "$BASE_URL/api/v1/list" "application/json" "List API returns application/json"

# Static files
log_test "Static Content-Type: CSS"
assert_content_type "$BASE_URL/style.css" "text/css" "CSS returns text/css"

log_test "Static Content-Type: JavaScript (main.js)"
assert_content_type "$BASE_URL/main.js" "application/javascript" "main.js returns application/javascript"

log_test "Static Content-Type: Manifest"
assert_content_type "$BASE_URL/manifest.json" "application/manifest+json" "manifest.json returns application/manifest+json"

log_test "Static Content-Type: Service Worker"
assert_content_type "$BASE_URL/sw.js" "application/javascript" "sw.js returns application/javascript"

# Text files (robots.txt, security.txt)
log_test "Text file Content-Type: robots.txt"
assert_content_type "$BASE_URL/robots.txt" "text/plain" "robots.txt returns text/plain"

log_test "Text file Content-Type: security.txt"
assert_content_type "$BASE_URL/.well-known/security.txt" "text/plain" "security.txt returns text/plain"

log_test "Sitemap Content-Type: sitemap.xml"
assert_content_type "$BASE_URL/sitemap.xml" "text/xml" "sitemap.xml returns text/xml"

# Verify robots.txt content
log_test "robots.txt content"
ROBOTS=$(curl -s "$BASE_URL/robots.txt")
assert_contains "$ROBOTS" "User-agent" "robots.txt has User-agent"

# Verify security.txt content
log_test "security.txt content"
SECURITY=$(curl -s "$BASE_URL/.well-known/security.txt")
assert_contains "$SECURITY" "Contact" "security.txt has Contact field"

# ============================================
# SECTION 8: SECURITY TESTS
# ============================================
echo -e "\n${YELLOW}=== SECTION 8: SECURITY TESTS ===${NC}\n"

log_test "SQL injection attempt stored as text"
SQL_INJ=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body='; DROP TABLE pastes; --" \
    --data-urlencode "syntax=plaintext")
SQL_ID=$(echo "$SQL_INJ" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
SQL_GET=$(curl -s "$BASE_URL/api/v1/get?id=$SQL_ID")
assert_contains "$SQL_GET" "DROP TABLE" "SQL stored as text (not executed)"

log_test "XSS in title stored escaped"
XSS_TITLE=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "title=<script>alert(1)</script>" \
    --data-urlencode "body=test" \
    --data-urlencode "syntax=plaintext")
XSS_ID=$(echo "$XSS_TITLE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
XSS_GET=$(curl -s "$BASE_URL/api/v1/get?id=$XSS_ID")
assert_contains "$XSS_GET" "script" "XSS attempt stored (will be escaped in HTML)"

log_test "Title length limit enforced"
LONG_TITLE=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "title=$(python3 -c 'print("A"*200)')" \
    --data-urlencode "body=test" \
    --data-urlencode "syntax=plaintext")
assert_contains "$LONG_TITLE" '"code":413' "413 for title exceeding limit"

log_test "data: URL rejected"
DATA_URL=$(curl -s -X POST "$BASE_URL/api/v1/new" \
    --data-urlencode "body=test" \
    --data-urlencode "syntax=plaintext" \
    --data-urlencode "authorURL=data:text/html,<script>alert(1)</script>")
assert_contains "$DATA_URL" '"code":400' "400 for data: URL"

# ============================================
# RESULTS SUMMARY
# ============================================
echo -e "\n${BLUE}========================================${NC}"
echo -e "${BLUE}TEST RESULTS SUMMARY${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "Total:  $TESTS_TOTAL"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}ALL TESTS PASSED!${NC}"
    exit 0
else
    echo -e "${RED}SOME TESTS FAILED${NC}"
    exit 1
fi
