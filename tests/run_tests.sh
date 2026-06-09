#!/usr/bin/env bash
# Auto-detect runtime (Incus preferred, Docker fallback) and run tests
# Per AI.md PART 29: tests/run_tests.sh — required entry point

set -eo pipefail

# Detect project info from git remote or directory path (NEVER hardcode)
PROJECTNAME=$(git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)(\.git)?$|\1|' || basename "$PWD")
PROJECTORG=$(git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)/[^/]+(\.git)?$|\1|' || basename "$(dirname "$PWD")")

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

log() { echo "[run_tests] $(date '+%Y-%m-%dT%H:%M:%SZ') $*"; }

case "${1:-}" in
    -h|--help)
        echo "Usage: $0 [--docker|--incus]"
        echo ""
        echo "Auto-detects Incus or Docker and runs integration tests."
        echo "Options:"
        echo "  --docker   Force Docker testing"
        echo "  --incus    Force Incus testing"
        echo ""
        echo "Project: $PROJECTNAME (org: $PROJECTORG)"
        exit 0
        ;;
    --docker)
        log "Forced Docker testing"
        exec "$SCRIPT_DIR/docker.sh"
        ;;
    --incus)
        log "Forced Incus testing"
        exec "$SCRIPT_DIR/incus.sh"
        ;;
esac

# Auto-detect: prefer Incus (full OS + systemd), fall back to Docker
if command -v incus &>/dev/null && incus list &>/dev/null 2>&1; then
    log "Incus available — running full OS tests (Debian + systemd)"
    exec "$SCRIPT_DIR/incus.sh"
else
    log "Incus not available — running Docker tests (Alpine, no systemd)"
    exec "$SCRIPT_DIR/docker.sh"
fi
