# CasPaste Makefile - Local Development
# Targets: build, release, docker, test, local, help
# All Go builds/tests run inside Docker (golang:alpine)

GH ?= gh

# Project info
NAME := caspaste
CLI_NAME := caspaste-cli
ORGANIZATION := casjay-forks
MAIN_GO := ./src/server
CLI_MAIN_GO := ./src/client

# Version management
VERSION_FILE := release.txt
DEFAULT_VERSION := 1.0.0

# Get version: env var > release.txt > git tag > default
ifdef VERSION
    APP_VERSION := $(VERSION)
else ifneq (,$(wildcard $(VERSION_FILE)))
    APP_VERSION := $(shell cat $(VERSION_FILE) | tr -d '[:space:]')
else
    GIT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')
    APP_VERSION := $(if $(GIT_VERSION),$(GIT_VERSION),$(DEFAULT_VERSION))
endif

# Directories
BUILD_DIR := ./binaries
RELEASE_DIR := ./releases

# Go directories (configurable via environment)
GODIR ?= $(HOME)/.local/share/go
GOCACHEDIR ?= $(HOME)/.local/share/go/build

# Build info per AI.md PART 26
COMMIT_ID := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -w -s -X main.Version=$(APP_VERSION) -X main.CommitID=$(COMMIT_ID) -X main.BuildDate=$(BUILD_DATE) -X main.OfficialSite=https://lp.pste.us -extldflags -static
STATIC_FLAGS := -tags netgo -ldflags "$(LDFLAGS)"

# Docker build environment
DOCKER_IMAGE := golang:alpine
DOCKER_OPTS := --rm \
	-v "$(CURDIR)":/build \
	-v "$(GODIR)":/go \
	-w /build \
	-e CGO_ENABLED=0 \
	-e GOCACHE=/go/build \
	-e GOMODCACHE=/go/pkg/mod

# For local builds, match the runtime machine's architecture
DOCKER_RUN_LOCAL = docker run --platform linux/$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') $(DOCKER_OPTS) $(DOCKER_IMAGE)

# For cross-platform builds
DOCKER_RUN = docker run $(DOCKER_OPTS) $(DOCKER_IMAGE)

# Platforms: os_arch (8 platforms per AI.md PART 26)
PLATFORMS := \
    linux_amd64 \
    linux_arm64 \
    darwin_amd64 \
    darwin_arm64 \
    windows_amd64 \
    windows_arm64 \
    freebsd_amd64 \
    freebsd_arm64

.PHONY: build release docker test local dev help clean

# Default target
help:
	@echo "CasPaste Makefile - Local Development"
	@echo "====================================="
	@echo ""
	@echo "Targets:"
	@echo "  make dev     - Quick build to temp dir (no version info, debugging)"
	@echo "  make local   - Build for current OS/arch only (fast, with version)"
	@echo "  make build   - Build all binaries for all OS/arch (./binaries/)"
	@echo "  make test    - Run all tests"
	@echo "  make release - Build production binaries and create GitHub release"
	@echo "  make docker  - Build and push Docker images to ghcr.io"
	@echo "  make clean   - Remove build artifacts"
	@echo ""
	@echo "Version: $(APP_VERSION)"
	@echo "Go cache: $(GODIR)"
	@echo "Note: All Go builds run inside Docker (golang:alpine)"
	@echo ""

# Quick dev build to temp directory (no version info)
# Per AI.md PART 28: make dev for quick debugging
TEMP_DIR := /tmp/$(ORGANIZATION)/$(NAME)-dev
dev:
	@mkdir -p $(TEMP_DIR) $(GODIR)/build $(GODIR)/pkg/mod
	@echo "Building $(NAME) (dev) to $(TEMP_DIR)..."
	@docker run --rm \
		-v "$(CURDIR)":/build \
		-v "$(GODIR)":/go \
		-v "$(TEMP_DIR)":/out \
		-w /build \
		-e CGO_ENABLED=0 \
		-e GOCACHE=/go/build \
		-e GOMODCACHE=/go/pkg/mod \
		--platform linux/$$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') \
		$(DOCKER_IMAGE) sh -c '\
			go mod tidy && \
			go build -trimpath -tags netgo -ldflags "-w -s" -o /out/$(NAME) $(MAIN_GO) && \
			go build -trimpath -tags netgo -ldflags "-w -s" -o /out/$(CLI_NAME) $(CLI_MAIN_GO)'
	@echo "Built: $(TEMP_DIR)/$(NAME) $(TEMP_DIR)/$(CLI_NAME)"

# Build for runtime machine's architecture
local:
	@if [ ! -f $(VERSION_FILE) ]; then echo "$(APP_VERSION)" > $(VERSION_FILE); fi
	@mkdir -p $(BUILD_DIR) $(GODIR)/build $(GODIR)/pkg/mod
	@echo "Building $(NAME) v$(APP_VERSION) for $$(uname -m)..."
	@$(DOCKER_RUN_LOCAL) sh -c '\
		go mod tidy && \
		go build -trimpath $(STATIC_FLAGS) -o $(BUILD_DIR)/$(NAME) $(MAIN_GO) && \
		go build -trimpath $(STATIC_FLAGS) -o $(BUILD_DIR)/$(CLI_NAME) $(CLI_MAIN_GO)'
	@echo "Built: $(BUILD_DIR)/$(NAME) $(BUILD_DIR)/$(CLI_NAME)"

# Build all platforms
build:
	@if [ ! -f $(VERSION_FILE) ]; then echo "$(APP_VERSION)" > $(VERSION_FILE); fi
	@mkdir -p $(BUILD_DIR) $(GODIR)/build $(GODIR)/pkg/mod
	@echo "Building $(NAME) v$(APP_VERSION) for all platforms..."
	@$(DOCKER_RUN) sh -c '\
		go mod tidy && \
		go build -trimpath $(STATIC_FLAGS) -o $(BUILD_DIR)/$(NAME) $(MAIN_GO) && \
		go build -trimpath $(STATIC_FLAGS) -o $(BUILD_DIR)/$(CLI_NAME) $(CLI_MAIN_GO) && \
		for platform in $(PLATFORMS); do \
			os=$$(echo $$platform | cut -d_ -f1); \
			arch=$$(echo $$platform | cut -d_ -f2); \
			ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
			echo "  $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch go build -trimpath $(STATIC_FLAGS) \
				-o $(BUILD_DIR)/$(NAME)-$$os-$$arch$$ext $(MAIN_GO) || exit 1; \
			GOOS=$$os GOARCH=$$arch go build -trimpath $(STATIC_FLAGS) \
				-o $(BUILD_DIR)/$(CLI_NAME)-$$os-$$arch$$ext $(CLI_MAIN_GO) || exit 1; \
		done'
	@chmod +x $(BUILD_DIR)/$(NAME) $(BUILD_DIR)/$(CLI_NAME)
	@echo "Build complete: $(BUILD_DIR)/"

# Release to GitHub
release:
	@if [ ! -f $(VERSION_FILE) ]; then echo "$(APP_VERSION)" > $(VERSION_FILE); fi
	@mkdir -p $(RELEASE_DIR) $(GODIR)/build $(GODIR)/pkg/mod
	@echo "Building release v$(APP_VERSION)..."
	@$(DOCKER_RUN) sh -c '\
		apk add --no-cache binutils && \
		go mod tidy && \
		for platform in $(PLATFORMS); do \
			os=$$(echo $$platform | cut -d_ -f1); \
			arch=$$(echo $$platform | cut -d_ -f2); \
			ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
			echo "  $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch go build -trimpath $(STATIC_FLAGS) \
				-o $(RELEASE_DIR)/$(NAME)-$$os-$$arch$$ext $(MAIN_GO) || exit 1; \
			GOOS=$$os GOARCH=$$arch go build -trimpath $(STATIC_FLAGS) \
				-o $(RELEASE_DIR)/$(CLI_NAME)-$$os-$$arch$$ext $(CLI_MAIN_GO) || exit 1; \
			if echo "$$os" | grep -qE "linux|freebsd|openbsd"; then \
				strip $(RELEASE_DIR)/$(NAME)-$$os-$$arch$$ext 2>/dev/null || true; \
				strip $(RELEASE_DIR)/$(CLI_NAME)-$$os-$$arch$$ext 2>/dev/null || true; \
			fi; \
		done'
	@# Source archive (no VCS)
	@echo "Creating source archive..."
	@mkdir -p $(RELEASE_DIR)/tmp/$(NAME)-$(APP_VERSION)
	@rsync -a --exclude='.git' --exclude='.github' --exclude='$(BUILD_DIR)' \
		--exclude='$(RELEASE_DIR)' --exclude='.gitignore' --exclude='.gitattributes' \
		--exclude='.go-cache' \
		. $(RELEASE_DIR)/tmp/$(NAME)-$(APP_VERSION)/
	@tar -C $(RELEASE_DIR)/tmp -czf $(RELEASE_DIR)/$(NAME)-$(APP_VERSION)-source.tar.gz $(NAME)-$(APP_VERSION)
	@rm -rf $(RELEASE_DIR)/tmp
	@# Delete existing tag/release
	@$(GH) release delete v$(APP_VERSION) --yes 2>/dev/null || true
	@git tag -d v$(APP_VERSION) 2>/dev/null || true
	@git push origin :refs/tags/v$(APP_VERSION) 2>/dev/null || true
	@# Create release
	@git tag -a v$(APP_VERSION) -m "Release v$(APP_VERSION)"
	@git push origin v$(APP_VERSION)
	@$(GH) release create v$(APP_VERSION) --title "$(NAME) v$(APP_VERSION)" --generate-notes $(RELEASE_DIR)/*
	@echo "Released v$(APP_VERSION)"

# Build and push Docker images (per AI.md PART 27)
docker:
	@if [ ! -f $(VERSION_FILE) ]; then echo "$(APP_VERSION)" > $(VERSION_FILE); fi
	@COMMIT_ID=$$(git rev-parse --short HEAD); \
	BUILD_DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	REPO="ghcr.io/$(ORGANIZATION)/$(NAME)"; \
	echo "Building Docker images v$(APP_VERSION)..."; \
	docker buildx build \
		-f docker/Dockerfile \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(APP_VERSION) \
		--build-arg BUILD_DATE="$$BUILD_DATE" \
		--build-arg COMMIT_ID=$$COMMIT_ID \
		--tag $$REPO:$(APP_VERSION) \
		--tag $$REPO:$$COMMIT_ID \
		--tag $$REPO:latest \
		--push . || exit 1; \
	echo "Pushed: $$REPO:latest $$REPO:$(APP_VERSION) $$REPO:$$COMMIT_ID"

# Run tests
test:
	@mkdir -p $(GODIR)/build $(GODIR)/pkg/mod
	@echo "Running tests..."
	@$(DOCKER_RUN) sh -c 'go mod tidy && go test -v -cover ./...'

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(RELEASE_DIR) .go-cache
	@echo "Clean complete"
