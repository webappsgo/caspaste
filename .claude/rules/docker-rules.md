# Docker Rules (PART 27)

⚠️ **These rules are NON-NEGOTIABLE. Violations are bugs.** ⚠️

## CRITICAL - NEVER DO
- NEVER place `Dockerfile` or `docker-compose.yml` in project root — always under `docker/`
- NEVER modify `ENTRYPOINT` or `CMD` — all customization goes in `entrypoint.sh`
- NEVER add `LABEL` blocks to a Dockerfile — all OCI metadata applied by CI at build time (`labels:`/`annotations:`)
- NEVER include `build:` or `version:` in docker-compose files
- NEVER use `.env`, `.env.example`, `.env.sample` files, or list-style env (`- KEY=value`) — map style with inline defaults only
- NEVER set `MODE`/`DEBUG` in production `docker-compose.yml` (binary defaults to production); only `.dev.yml`/`.test.yml` set them
- NEVER run `docker compose` from the project directory or mount volumes to `{project_root}/volumes/` — always a temp dir
- NEVER commit runtime `./volumes/` content — only `docker/rootfs/` (build-time overlay) is committed
- NEVER push `:dev` or `:test` tags to the production registry
- AI assistants must NEVER use `docker-compose.dev.yml` or `docker-compose.yml` directly — human use only; AI uses `docker-compose.test.yml` (prefer `tests/run_tests.sh` / `tests/docker.sh`)

## CRITICAL - ALWAYS DO
- ALWAYS use multi-stage builds: builder (`casjaysdev/go:latest`) + runtime (`alpine:latest`, or `debian:latest` for AIO)
- ALWAYS use `docker/rootfs/` for the build-time container overlay (mirrors container filesystem)
- ALWAYS use `entrypoint.sh` at `/usr/local/bin/entrypoint.sh` as the container startup path, run via `tini -p SIGTERM`
- ALWAYS use `STOPSIGNAL SIGRTMIN+3` and a `HEALTHCHECK` (start 90s, interval 10s, timeout 5s, retries 3) calling `{binary} --status`
- ALWAYS expose internal port **80**
- ALWAYS mount exactly two compose volumes: `./volumes/config:/config:z` and `./volumes/data:/data:z` (`:z` in production; omit in dev temp dirs)
- ALWAYS include the `x-logging` anchor (`json-file`, `max-size: 5m`, `max-file: 1`) and apply it to every service
- ALWAYS hardcode compose environment values with sane defaults (`${VAR:-default}`) — stack must work with zero config
- ALWAYS run docker compose from a temp dir (`mktemp -d "${TMPDIR:-/tmp}/{project_org}/{internal_name}-XXXXXX"`), never the repo

## KEY DECISIONS (pre-answered)
| Question | Answer | Spec Reference |
|----------|--------|----------------|
| Builder base image | `casjaysdev/go:latest` | Dockerfile Requirements |
| Runtime base image (standard) | `alpine:latest` | Dockerfile Requirements |
| Runtime base image (all-in-one) | `debian:latest` (glibc needed for postgres/valkey/tor) | AIO Dockerfile |
| Internal port | `80`, always | Container Configuration |
| Init system | `tini` | Dockerfile Requirements |
| Prod external port bind | `172.17.0.1:{port}:80` | Port Mapping |
| Dev external port bind | `{port}:80` (all interfaces) | Port Mapping |
| Who owns Tor | The server binary, not a separate service | Tor in Container |
| AIO database | PostgreSQL (unix socket, no TCP) | All-in-One Database and Cache |
| AIO cache | Valkey (unix socket, AOF persistence) | All-in-One Database and Cache |
| Where do labels get applied | CI (`docker build --label` / `docker/metadata-action`), never `LABEL` in Dockerfile | OCI Meta Labels |
| Standard vs AIO image tag | `:latest` (standard) vs `:latest-aio` (all-in-one) | Image Names |

## TERMINOLOGY
| Term | Meaning |
|------|---------|
| Build-time `docker/rootfs/` | Container overlay committed to git, copied via `COPY docker/rootfs/ /` |
| Runtime `./volumes/` | Host bind-mount dirs created next to the compose file at run time; never committed |
| All-in-One (AIO) | Single container running app + embedded Postgres + Valkey + Tor via supervisord |
| Multi-Service | Separate containers per service (app, db, cache) |
| `internal_name` | Registry/image name; may differ from `project_name` |

## QUICK REFERENCE
- Directory layout: `docker/Dockerfile`, `Dockerfile.dev`, `Dockerfile.aio`, `docker-compose.yml`, `docker-compose.dev.yml` (human only), `docker-compose.test.yml` (AI/CI), `all-in-one.yml`, `rootfs/usr/local/bin/entrypoint.sh`
- Build context is always the project root (`.`); Dockerfile referenced with `-f docker/Dockerfile`
- Entrypoint does ONLY: set env defaults, optionally start supervisord, exec the binary, handle signals — binary owns dirs/perms/users/Tor
- Container paths: `/config/{project_name}/`, `/data/{project_name}/`, `/data/db/{sqlite,postgres,valkey}/`, `/data/log/{project_name}/`, `/data/backups/{project_name}/`
- Service naming: `{project_name}` / `{project_name}-app`, `{project_name}-db`, `{project_name}-cache`, `{project_name}-search`, `{project_name}-queue`, `{project_name}-proxy`
- Image tags: `:latest`, `:{version}`, `:{YYMM}`, `:{commit}` pushed to `{PLATFORM_CONTAINER_REGISTRY}/{project_org}/{internal_name}`; local-only `:dev`/`:test` never pushed
- Release images built for `linux/amd64` and `linux/arm64`
- Test workflow: prefer `tests/run_tests.sh` / `tests/docker.sh` over invoking `docker-compose.test.yml` directly

---
For complete details, see AI.md PART 27
