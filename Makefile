BINARY   := jira
BUILD_DIR := bin
CMD      := ./cmd/jira
GO       := go

.PHONY: build install clean test build-all

build:
	$(GO) build -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	$(GO) install $(CMD)

test:
	$(GO) test ./...

clean:
	rm -rf $(BUILD_DIR)

build-all: clean
	GOOS=darwin  GOARCH=arm64 $(GO) build -o $(BUILD_DIR)/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=darwin  GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=linux   GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY)-linux-amd64   $(CMD)
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY)-windows.exe   $(CMD)
