.PHONY: build build-all clean install update test test-unit test-integration test-coverage lint

BINARY=memo
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

# Default: build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

# Build for all platforms
build-all: clean
	mkdir -p dist
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

clean:
	rm -rf dist $(BINARY) coverage.out coverage.html

install: build
	mkdir -p $(HOME)/.local/bin
	@# Remove old binary first to avoid dyld issues when processes hold references
	rm -f $(HOME)/.local/bin/$(BINARY)
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
	@echo "Installed $(BINARY) to $(HOME)/.local/bin"

update:
	git fetch origin
	git reset --hard origin/main
	$(MAKE) install

# ============== Testing ==============

# Run all tests
test: test-unit test-integration

# Run unit tests (fast, no external dependencies)
test-unit:
	go test -v -race -timeout 60s ./tests/analyzer/... ./tests/internal/... ./tests/mcp/...

# Run integration tests (may build binary, slower)
test-integration:
	go test -v -race -timeout 300s ./tests/integration/...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run specific test package
test-analyzer:
	go test -v -race ./tests/analyzer/...

test-mcp:
	go test -v -race ./tests/mcp/...

test-internal:
	go test -v -race ./tests/internal/...

# ============== Linting ==============

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed, running go vet instead"; \
		go vet ./...; \
	fi

# ============== CI Helpers ==============

# Used by CI to run all checks
ci: lint test

# Verify build works on all platforms (for CI)
ci-build-check:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o /dev/null .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o /dev/null .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o /dev/null .
	@echo "Build check passed for all platforms"
