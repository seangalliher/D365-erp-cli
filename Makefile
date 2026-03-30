.DEFAULT_GOAL := build

BINARY_NAME := d365
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -ldflags="-s -w -X github.com/seangalliher/d365-erp-cli/cmd.version=$(VERSION) -X github.com/seangalliher/d365-erp-cli/cmd.commit=$(COMMIT) -X github.com/seangalliher/d365-erp-cli/cmd.date=$(DATE)"

.PHONY: build test lint test-coverage clean install release fmt vet

## build: Build the CLI binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## build-all: Build the CLI binary
build-all: build

## install: Install the binary to GOPATH/bin
install:
	go install $(LDFLAGS) .

## test: Run all unit tests in parallel with race detection
test:
	go test -race -count=1 -parallel 8 -timeout 120s ./...

## test-v: Run tests with verbose output
test-v:
	go test -v -race -count=1 -parallel 8 -timeout 120s ./...

## test-coverage: Run tests with coverage report
test-coverage:
	go test -race -count=1 -parallel 8 -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML coverage report: go tool cover -html=coverage.out -o coverage.html"

## test-integration: Run integration tests (requires D365 sandbox)
test-integration:
	go test -v -tags=integration -parallel 4 -timeout 300s ./test/integration/...

## lint: Run linters
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	gofmt -s -w .
	goimports -w .

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe d365-formd d365-formd.exe
	rm -f coverage.out coverage.html
	rm -rf dist/

## release: Build release binaries with GoReleaser (snapshot)
release:
	goreleaser build --snapshot --clean

## release-publish: Create a release (requires GITHUB_TOKEN)
release-publish:
	goreleaser release --clean

## cross-build: Build for all platforms
cross-build:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/d365-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/d365-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/d365-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/d365-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/d365-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o dist/d365-windows-arm64.exe .

## help: Show this help
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
