# =============================================================================
# Build Stage — Compile Go binary
# =============================================================================
FROM casjaysdev/go:latest AS builder

# Install git (required for go mod download) and bash (for build scripts)
RUN apk add --no-cache git bash

ARG TARGETARCH
ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_ID
ARG OFFICIAL_SITE

WORKDIR /app

# Copy go.mod first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build server binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w \
      -X 'main.Version=${VERSION}' \
      -X 'main.CommitID=${COMMIT_ID}' \
      -X 'main.BuildDate=${BUILD_DATE}' \
      -X 'main.OfficialSite=${OFFICIAL_SITE}'" \
    -o /app/binary/caspaste ./src/server

# =============================================================================
# Runtime Stage — Minimal Alpine image
# =============================================================================
FROM alpine:latest

# ARGs for build-time values (set by docker build --build-arg)
ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_ID

# No LABEL blocks — all image metadata is applied as OCI annotations at build time.
# Pass --annotation flags to docker buildx build (or use docker/metadata-action
# annotations: output in GitHub Actions). Annotations attach to the manifest index
# and are visible on multiarch pulls. LABEL applies only to per-platform layers.

# Install required packages
# NOTE: Tor binary installed but NOT configured here — binary handles all Tor setup (see PART 32)
RUN apk add --no-cache \
    git \
    curl \
    bash \
    tini \
    tor

# NO directory creation — binary handles ALL setup based on env vars
# Docker volume mounts auto-create mount points

# Copy server binary from builder stage (standard image = server only, per AI.md PART 27)
COPY --from=builder /app/binary/caspaste /usr/local/bin/caspaste

# Copy build-time overlay (entrypoint.sh) from docker/rootfs/ into image
COPY docker/rootfs/ /

# Copy Dockerfile to image for reference
COPY docker/Dockerfile /root/Dockerfile

# Make all binaries/scripts in /usr/local/bin executable
RUN chmod 755 /usr/local/bin/*

# Set environment
# NOTE: Tor available (binary installed) but server binary controls Tor startup (see PART 32)
# DATABASE_DIR: SQLite location (binary auto-detects container, but explicit is clearer)
ENV MODE=development \
    TZ=America/New_York \
    DATABASE_DIR=/data/db/sqlite

# Expose internal port (always 80)
EXPOSE 80

# Stop signal for graceful shutdown
STOPSIGNAL SIGRTMIN+3

# Health check (long start period for services that need initialization)
HEALTHCHECK --start-period=10m --interval=5m --timeout=15s --retries=3 \
    CMD /usr/local/bin/caspaste --status || exit 1

# Use tini as init with signal propagation
# -p SIGTERM: propagate SIGTERM to child processes
ENTRYPOINT [ "tini", "-p", "SIGTERM", "--", "/usr/local/bin/entrypoint.sh" ]
