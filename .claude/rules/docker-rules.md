# Docker Rules (PART 27) — Cheatsheet

Full spec: AI.md PART 27

## Dockerfile Rules

- Builder: FROM casjaysdev/go:latest AS builder
- Runtime: FROM alpine:latest
- NO LABEL blocks — OCI annotations via docker/metadata-action in workflows
- NO directory creation (binary handles all setup)
- STOPSIGNAL SIGRTMIN+3
- ENTRYPOINT ["tini", "-p", "SIGTERM", "--", "/usr/local/bin/entrypoint.sh"]
- HEALTHCHECK --start-period=10m --interval=5m --timeout=15s
- ENV MODE=development (default in image)
- ARG OFFICIAL_SITE (read from site.txt or env)
- Binary path: /usr/local/bin/caspaste

## Build Arguments (all Dockerfiles)

- ARG TARGETARCH
- ARG VERSION=dev
- ARG BUILD_DATE
- ARG COMMIT_ID
- ARG OFFICIAL_SITE

## Compose Rules

- NEVER: `version:` field, `build:` field
- ALWAYS: `name:` at top, `pull_policy: always`, `restart: always`
- ALWAYS: `x-logging: &default-logging` anchor + use it in every service
- ALWAYS: `./volumes/config:/config:z` and `./volumes/data:/data:z`
- Service name: caspaste (matches project name)
- Container name: caspaste-app
- Port: 172.17.0.1:64580:80
- hostname: ${BASE_HOST_NAME:-$HOSTNAME}
- MODE=production in production compose

## Service Naming

| Type | Service Name | Container Name |
|------|-------------|----------------|
| App | caspaste | caspaste-app |
| DB | caspaste-db | caspaste-db |
| Cache | caspaste-cache | caspaste-cache |

## Compose Files

- docker-compose.yml — production, HUMAN USE ONLY
- docker-compose.dev.yml — development, HUMAN USE ONLY
- docker-compose.test.yml — AI testing ONLY (copy to temp dir)

## Entrypoint Rules

- MINIMAL: only set env, start services, start binary, handle signals
- NEVER: create directories, set permissions, manage Tor
- Export: TZ, CONFIG_DIR, DATA_DIR
- Check: DEBUG env var to add --debug flag
- Signal handling: cleanup() trap for SIGTERM/SIGINT/SIGQUIT
