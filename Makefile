BINARY    := jira
BUILD_DIR := bin
CMD       := ./cmd/jira
GO        := go

VERSION ?= dev
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X github.com/caiocesarps/jira-cli/internal/version.Version=$(VERSION) \
	-X github.com/caiocesarps/jira-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/caiocesarps/jira-cli/internal/version.Date=$(DATE)

PLATFORMS := \
	darwin/amd64 \
	darwin/arm64 \
	linux/amd64 \
	linux/arm64 \
	windows/amd64

.PHONY: build install clean test build-all version

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	$(GO) install -ldflags "$(LDFLAGS)" $(CMD)

test:
	$(GO) test ./...

clean:
	rm -rf $(BUILD_DIR)

build-all: clean
	@set -e; \
	for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/}; \
		output=$(BUILD_DIR)/$(BINARY)-$${GOOS}-$${GOARCH}; \
		if [ "$${GOOS}" = "windows" ]; then output=$${output}.exe; fi; \
		echo "Building $${GOOS}/$${GOARCH} -> $${output}"; \
		GOOS=$${GOOS} GOARCH=$${GOARCH} $(GO) build -ldflags "$(LDFLAGS)" -o $${output} $(CMD); \
	done

version: build
	$(BUILD_DIR)/$(BINARY) version
