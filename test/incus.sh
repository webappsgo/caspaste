#!/usr/bin/env bash
# Incus testing script for CasPaste
# Per AI.md PART 29: Incus PREFERRED for full OS testing with systemd

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ORG="casjay-forks"
PROJECT_NAME="caspaste"
CONTAINER_NAME="${PROJECT_NAME}-test"
IMAGE="images:debian/12"

# Check if incus is available
if ! command -v incus &>/dev/null; then
    echo "Error: incus is not installed"
    echo "Install with: apt install incus (Debian/Ubuntu) or see https://linuxcontainers.org/incus/"
    exit 1
fi

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    incus delete "$CONTAINER_NAME" --force 2>/dev/null || true
}
trap cleanup EXIT

# Create container
echo "Creating Incus container: $CONTAINER_NAME"
incus delete "$CONTAINER_NAME" --force 2>/dev/null || true
incus launch "$IMAGE" "$CONTAINER_NAME"

# Wait for container to be ready
echo "Waiting for container..."
sleep 5

# Install dependencies
echo "Installing dependencies..."
incus exec "$CONTAINER_NAME" -- apt-get update -qq
incus exec "$CONTAINER_NAME" -- apt-get install -y -qq curl ca-certificates

# Copy binary to container
echo "Copying binary..."
incus file push "$PROJECT_ROOT/binaries/caspaste" "$CONTAINER_NAME/usr/local/bin/caspaste"
incus exec "$CONTAINER_NAME" -- chmod +x /usr/local/bin/caspaste

# Create directories
incus exec "$CONTAINER_NAME" -- mkdir -p /etc/casjay-forks/caspaste
incus exec "$CONTAINER_NAME" -- mkdir -p /var/lib/casjay-forks/caspaste
incus exec "$CONTAINER_NAME" -- mkdir -p /var/log/casjay-forks/caspaste

# Start server in background
echo "Starting server..."
incus exec "$CONTAINER_NAME" -- sh -c 'nohup caspaste --port 8080 > /var/log/casjay-forks/caspaste/server.log 2>&1 &'
sleep 5

# Check if server is running
echo "Checking server status..."
if incus exec "$CONTAINER_NAME" -- caspaste --status; then
    echo "Server is running"
else
    echo "Server failed to start"
    incus exec "$CONTAINER_NAME" -- cat /var/log/casjay-forks/caspaste/server.log
    exit 1
fi

# Test health endpoint
echo "Testing health endpoint..."
incus exec "$CONTAINER_NAME" -- curl -s http://localhost:8080/healthz | head -10

echo "Incus tests completed successfully"
