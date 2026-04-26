#!/usr/bin/env bash
# Docker testing script for CasPaste
# Per AI.md PART 29: Container-only testing

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ORG="casjay-forks"
PROJECT_NAME="caspaste"

# Create temp directory per AI.md PART 29
mkdir -p "${TMPDIR:-/tmp}/${PROJECT_ORG}"
TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${PROJECT_ORG}/${PROJECT_NAME}-XXXXXX")
echo "Using temp directory: $TEMP_DIR"

# Create runtime directories
mkdir -p "$TEMP_DIR/rootfs/config" "$TEMP_DIR/rootfs/data"

# Copy docker-compose.test.yml
cp "$PROJECT_ROOT/docker/docker-compose.test.yml" "$TEMP_DIR/docker-compose.yml"

# Build test image
echo "Building test image..."
docker build -f "$PROJECT_ROOT/docker/Dockerfile" -t "${PROJECT_NAME}:test" "$PROJECT_ROOT"

# Run tests
echo "Starting container..."
cd "$TEMP_DIR"
docker compose up -d

# Wait for health check
echo "Waiting for health check..."
for i in {1..30}; do
    if docker compose exec -T app caspaste --status 2>/dev/null; then
        echo "Server is healthy"
        break
    fi
    echo "Waiting... ($i/30)"
    sleep 2
done

# Run basic tests
echo "Running basic tests..."

# Test health endpoint
echo "Testing /healthz..."
curl -s "http://localhost:18080/healthz" | head -20

# Cleanup
echo "Cleaning up..."
docker compose down
rm -rf "$TEMP_DIR"

echo "Docker tests completed"
