.PHONY: build build-windows build-linux build-darwin test test-coverage lint clean install help

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X github.com/sharkusmanch/ludusavi-runner/pkg/version.Version=$(VERSION) -X github.com/sharkusmanch/ludusavi-runner/pkg/version.Commit=$(COMMIT) -X github.com/sharkusmanch/ludusavi-runner/pkg/version.Date=$(DATE)"

# Binary name
BINARY := ludusavi-runner

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Default target
all: lint test build

## build: Build binary for current platform
build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY) ./cmd/ludusavi-runner

## build-windows: Build Windows binary
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)_windows_amd64.exe ./cmd/ludusavi-runner

## build-linux: Build Linux binary
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)_linux_amd64 ./cmd/ludusavi-runner

## build-darwin: Build macOS binary
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)_darwin_amd64 ./cmd/ludusavi-runner
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY)_darwin_arm64 ./cmd/ludusavi-runner

## build-all: Build binaries for all platforms
build-all: build-windows build-linux build-darwin

## test: Run unit tests
test:
	$(GOTEST) -v -race ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## test-integration: Run integration tests
test-integration:
	$(GOTEST) -v -race -tags=integration ./...

## lint: Run linters
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found, installing..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOCMD) fmt ./...
	@which goimports > /dev/null || go install golang.org/x/tools/cmd/goimports@latest
	goimports -w .

## tidy: Tidy and verify dependencies
tidy:
	$(GOMOD) tidy
	$(GOMOD) verify

## clean: Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

## install: Install binary to GOPATH/bin
install:
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) ./cmd/ludusavi-runner

## run: Run the application
run: build
	./bin/$(BINARY)

## validate: Run validate command
validate: build
	./bin/$(BINARY) validate

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
