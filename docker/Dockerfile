# syntax=docker/dockerfile:1

# Build stage
FROM golang:alpine AS builder

WORKDIR /build

RUN apk add --no-cache git curl jq ca-certificates tzdata

COPY go.mod go.sum ./
COPY src/ ./src/

RUN go mod tidy && go mod download

ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_ID

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -tags netgo \
    -ldflags "-w -s -X main.Version=${VERSION} -X main.CommitID=${COMMIT_ID} -X 'main.BuildDate=${BUILD_DATE}' -X main.OfficialSite=https://pste.us -extldflags '-static'" \
    -o /caspaste \
    ./src/server

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -tags netgo \
    -ldflags "-w -s -X main.Version=${VERSION} -X main.CommitID=${COMMIT_ID} -X 'main.BuildDate=${BUILD_DATE}' -X main.OfficialSite=https://pste.us -extldflags '-static'" \
    -o /caspaste-cli \
    ./src/client

# Final stage
FROM alpine:latest

ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_ID

# OCI Labels (required per AI.md PART 27)
LABEL maintainer="casjay-forks" \
      org.opencontainers.image.title="caspaste" \
      org.opencontainers.image.description="A simple, fast, and secure self-hosted pastebin service" \
      org.opencontainers.image.authors="casjay-forks" \
      org.opencontainers.image.vendor="casjay-forks" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.url="https://github.com/casjay-forks/caspaste" \
      org.opencontainers.image.documentation="https://github.com/casjay-forks/caspaste/blob/main/README.md" \
      org.opencontainers.image.source="https://github.com/casjay-forks/caspaste" \
      org.opencontainers.image.base.name="alpine:latest" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${COMMIT_ID}" \
      org.opencontainers.image.schema-version="${VERSION}" \
      org.opencontainers.image.vcs-type="Git" \
      com.github.containers.toolbox="false"

# Required packages per AI.md PART 27
RUN apk add --no-cache git curl bash tini tor ca-certificates tzdata

# Copy binaries from builder
COPY --from=builder /caspaste /usr/local/bin/caspaste
COPY --from=builder /caspaste-cli /usr/local/bin/caspaste-cli

# Copy entrypoint and filesystem overlay
COPY docker/rootfs/ /

# Create directories
RUN mkdir -p /config/caspaste /data/caspaste /data/log/caspaste && \
    chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /data/caspaste

# Container paths per AI.md PART 27
ENV PORT=80 \
    CASPASTE_CONFIG_DIR=/config/caspaste \
    CASPASTE_DATA_DIR=/data/caspaste \
    CASPASTE_LOGS_DIR=/data/log/caspaste \
    TZ=America/New_York

EXPOSE 80

VOLUME ["/config", "/data"]

# STOPSIGNAL per AI.md PART 27
STOPSIGNAL SIGRTMIN+3

# ENTRYPOINT per AI.md PART 27 - use tini
ENTRYPOINT ["tini", "-p", "SIGTERM", "--", "/usr/local/bin/entrypoint.sh"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=90s --retries=3 \
    CMD caspaste --status || exit 1
