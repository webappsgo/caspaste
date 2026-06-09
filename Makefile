# CasPaste Makefile - Local Development Only
# Targets: dev, local, build, test, release, docker, clean
# All Go builds/tests run inside Docker (casjaysdev/go:latest)
# DO NOT ADD MORE TARGETS per AI.md PART 26

# Infer PROJECTNAME and PROJECTORG from git remote or directory path (NEVER hardcode)
PROJECTNAME := $(shell git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)(\.git)?$$|\1|' || basename "$$(pwd)")
PROJECTORG  := $(shell git remote get-url origin 2>/dev/null | sed -E 's|.*/([^/]+)/[^/]+(\.git)?$$|\1|' || basename "$$(dirname "$$(pwd)")")

# Version precedence: release.txt > env/default fallback
VERSION ?= $(shell cat release.txt 2>/dev/null || echo "devel")

# Build info — ISO 8601 / RFC 3339 UTC per AI.md PART 26
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_ID  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "N/A")

# Official site URL (OPTIONAL — never guess or assume)
# Sources (in order of precedence):
#   1. File: site.txt in project root (single line, URL only)
#   2. Environment variable: OFFICIALSITE=https://example.com
#   3. Empty (self-hosted projects — users must use --server flag)
OFFICIALSITE := $(shell [ -f site.txt ] && cat site.txt || echo "${OFFICIALSITE:-}")

# Linker flags to embed build info
LDFLAGS := -s -w \
	-X 'main.Version=$(VERSION)' \
	-X 'main.CommitID=$(COMMIT_ID)' \
	-X 'main.BuildDate=$(BUILD_DATE)' \
	-X 'main.OfficialSite=$(OFFICIALSITE)'

# Directories
BINDIR  := binaries
RELDIR  := releases

# Docker — persistent Go state in named volume (NOT a host path)
# go-state:/usr/local/share/go keeps modules cached across builds
GO_DOCKER := docker run --rm -it \
	--name $(PROJECTNAME)-$$(tr -dc 'a-z0-9' </dev/urandom | head -c8) \
	-v $(PWD):/app \
	-v go-state:/usr/local/share/go \
	-w /app \
	-e CGO_ENABLED=0 \
	casjaysdev/go:latest

# Registry for docker target
REGISTRY ?= ghcr.io/$(PROJECTORG)/$(PROJECTNAME)

# Build platforms (8 platforms per AI.md PART 26)
PLATFORMS ?= linux/amd64,linux/arm64,darwin/amd64,darwin/arm64,windows/amd64,windows/arm64,freebsd/amd64,freebsd/arm64

.PHONY: build local release docker test dev clean

# =============================================================================
# BUILD — Build all platforms + local binary (via Docker with cached modules)
# =============================================================================
build: clean
	@mkdir -p $(BINDIR)
	@echo "Building version $(VERSION)..."
	@$(GO_DOCKER) go mod tidy
	@$(GO_DOCKER) go mod download
	@$(GO_DOCKER) sh -c "GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) \
		go build -ldflags \"$(LDFLAGS)\" -o $(BINDIR)/$(PROJECTNAME) ./src/server"
	@if [ -d "src/client" ]; then \
		$(GO_DOCKER) sh -c "GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) \
			go build -ldflags \"$(LDFLAGS)\" -o $(BINDIR)/$(PROJECTNAME)-cli ./src/client"; \
	fi
	@for platform in $$(echo "$(PLATFORMS)" | tr ',' ' '); do \
		OS=$${platform%/*}; \
		ARCH=$${platform#*/}; \
		OUTPUT=$(BINDIR)/$(PROJECTNAME)-$$OS-$$ARCH; \
		[ "$$OS" = "windows" ] && OUTPUT=$$OUTPUT.exe; \
		echo "Building server $$OS/$$ARCH..."; \
		$(GO_DOCKER) sh -c "GOOS=$$OS GOARCH=$$ARCH \
			go build -ldflags \"$(LDFLAGS)\" \
			-o $$OUTPUT ./src/server" || exit 1; \
		if [ -d "src/client" ]; then \
			CLI_OUTPUT=$(BINDIR)/$(PROJECTNAME)-cli-$$OS-$$ARCH; \
			[ "$$OS" = "windows" ] && CLI_OUTPUT=$$CLI_OUTPUT.exe; \
			$(GO_DOCKER) sh -c "GOOS=$$OS GOARCH=$$ARCH \
				go build -ldflags \"$(LDFLAGS)\" \
				-o $$CLI_OUTPUT ./src/client" || exit 1; \
		fi; \
	done
	@echo "Build complete: $(BINDIR)/"

# =============================================================================
# LOCAL — Build local binaries only (fast development builds)
# =============================================================================
local: clean
	@mkdir -p $(BINDIR)
	@echo "Building local binaries version $(VERSION)..."
	@$(GO_DOCKER) go mod tidy
	@$(GO_DOCKER) go mod download
	@$(GO_DOCKER) sh -c "GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) \
		go build -ldflags \"$(LDFLAGS)\" -o $(BINDIR)/$(PROJECTNAME) ./src/server"
	@if [ -d "src/client" ]; then \
		$(GO_DOCKER) sh -c "GOOS=$$(go env GOOS) GOARCH=$$(go env GOARCH) \
			go build -ldflags \"$(LDFLAGS)\" -o $(BINDIR)/$(PROJECTNAME)-cli ./src/client"; \
	fi
	@echo "Local build complete: $(BINDIR)/"

# =============================================================================
# RELEASE — Manual local release (stable only)
# =============================================================================
release: build
	@mkdir -p $(RELDIR)
	@echo "Preparing release $(VERSION)..."
	@echo "$(VERSION)" > $(RELDIR)/version.txt
	@for f in $(BINDIR)/$(PROJECTNAME)-*; do \
		[ -f "$$f" ] || continue; \
		strip "$$f" 2>/dev/null || true; \
		cp "$$f" $(RELDIR)/; \
	done
	@tar --exclude='.git' --exclude='.github' --exclude='.gitea' \
		--exclude='binaries' --exclude='releases' --exclude='*.tar.gz' \
		-czf $(RELDIR)/$(PROJECTNAME)-$(VERSION)-source.tar.gz .
	@gh release delete $(VERSION) --yes 2>/dev/null || true
	@git tag -d $(VERSION) 2>/dev/null || true
	@git push origin :refs/tags/$(VERSION) 2>/dev/null || true
	@gh release create $(VERSION) $(RELDIR)/* \
		--title "$(PROJECTNAME) $(VERSION)" \
		--notes "Release $(VERSION)" \
		--latest
	@echo "Release complete: $(VERSION)"

# =============================================================================
# DOCKER — Build container (set REGISTRY env var to override)
# =============================================================================
docker:
	@echo "Building Docker image $(VERSION)..."
	@docker buildx version > /dev/null 2>&1 || (echo "docker buildx required" && exit 1)
	@docker buildx create --name $(PROJECTNAME)-builder --use 2>/dev/null || \
		docker buildx use $(PROJECTNAME)-builder
	@docker buildx build \
		-f docker/Dockerfile \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION="$(VERSION)" \
		--build-arg BUILD_DATE="$(BUILD_DATE)" \
		--build-arg COMMIT_ID="$(COMMIT_ID)" \
		--build-arg OFFICIAL_SITE="$(OFFICIALSITE)" \
		-t $(REGISTRY):$(VERSION) \
		-t $(REGISTRY):latest \
		.
	@echo "Docker build complete: $(REGISTRY):$(VERSION)"

# =============================================================================
# TEST — Run unit tests with coverage enforcement (via Docker)
# =============================================================================
test:
	@echo "Running tests with coverage..."
	@$(GO_DOCKER) go mod download
	@$(GO_DOCKER) go test -v -cover -coverprofile=coverage.out ./...
	@COVERAGE=$$($(GO_DOCKER) go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ "$$(echo "$$COVERAGE < 80" | bc -l)" -eq 1 ]; then \
		echo "ERROR: Coverage is $$COVERAGE%%, must be >= 80%%"; \
		exit 1; \
	fi
	@echo "Tests complete (>= 80%% required) ✓"

# =============================================================================
# DEV — Quick build to random temp dir (no version info, debugging)
# =============================================================================
dev:
	@$(GO_DOCKER) go mod tidy
	@mkdir -p "$${TMPDIR:-/tmp}/$(PROJECTORG)" && \
		BUILD_DIR=$$(mktemp -d "$${TMPDIR:-/tmp}/$(PROJECTORG)/$(PROJECTNAME)-XXXXXX") && \
		echo "Quick dev build to $$BUILD_DIR..." && \
		$(GO_DOCKER) go build -o $$BUILD_DIR/$(PROJECTNAME) ./src/server && \
		echo "Built: $$BUILD_DIR/$(PROJECTNAME)" && \
		if [ -d "src/client" ]; then \
			$(GO_DOCKER) go build -o $$BUILD_DIR/$(PROJECTNAME)-cli ./src/client && \
			echo "Built: $$BUILD_DIR/$(PROJECTNAME)-cli"; \
		fi && \
		echo "Test: docker run --rm -it --name $(PROJECTNAME)-test -v $$BUILD_DIR:/app alpine:latest /app/$(PROJECTNAME) --help"

# =============================================================================
# CLEAN — Remove build artifacts
# =============================================================================
clean:
	@rm -rf $(BINDIR) $(RELDIR) coverage.out
