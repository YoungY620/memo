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

# Build tag for testing (enables export_testing.go)
TEST_TAGS=-tags testing

# Run all tests
test: test-unit test-integration

# Run unit tests (fast, no external dependencies)
test-unit:
	go test -v -race -timeout 60s $(TEST_TAGS) ./tests/analyzer/... ./tests/internal/... ./tests/mcp/...

# Run integration tests (may build binary, slower)
test-integration:
	go test -v -race -timeout 300s $(TEST_TAGS) ./tests/integration/...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic $(TEST_TAGS) ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run specific test package
test-analyzer:
	go test -v -race $(TEST_TAGS) ./tests/analyzer/...

test-mcp:
	go test -v -race $(TEST_TAGS) ./tests/mcp/...

test-internal:
	go test -v -race $(TEST_TAGS) ./tests/internal/...

# Run benchmarks
test-bench:
	go test -v -run=^$$ -bench=. $(TEST_TAGS) ./tests/...

# ============== Linting ==============

# Install golangci-lint if not present
lint-install:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

# Run all linters (golangci-lint includes errcheck, staticcheck, etc.)
lint: lint-install
	golangci-lint run --timeout=5m ./...

# Quick lint - only changed files (faster for development)
lint-fast: lint-install
	golangci-lint run --timeout=2m --new ./...

# Run go vet only (built-in, fast)
vet:
	go vet ./...

# Run errcheck specifically (catches unchecked error returns)
errcheck: lint-install
	golangci-lint run --enable=errcheck --disable-all ./...

# Run staticcheck (comprehensive static analysis)
staticcheck: lint-install
	golangci-lint run --enable=staticcheck --disable-all ./...

# ============== Pre-commit Checks ==============

# Run all checks before committing (recommended: make pre-commit)
pre-commit: fmt vet lint test-unit
	@echo ""
	@echo "✅ All pre-commit checks passed!"
	@echo ""

# Format code
fmt:
	go fmt ./...
	@echo "Code formatted"

# Quick check (faster, for rapid iteration)
check: fmt vet lint-fast test-unit
	@echo ""
	@echo "✅ Quick checks passed!"
	@echo ""

# ============== CI Helpers ==============

# Used by CI to run all checks
ci: lint test

# Verify build works on all platforms (for CI)
ci-build-check:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o /dev/null .
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o /dev/null .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o /dev/null .
	@echo "Build check passed for all platforms"

# ============== Help ==============

help:
	@echo "Available targets:"
	@echo ""
	@echo "  Build:"
	@echo "    make build        - Build for current platform"
	@echo "    make build-all    - Build for all platforms"
	@echo "    make install      - Build and install to ~/.local/bin"
	@echo "    make clean        - Remove build artifacts"
	@echo ""
	@echo "  Test:"
	@echo "    make test         - Run all tests"
	@echo "    make test-unit    - Run unit tests only"
	@echo "    make test-bench   - Run benchmarks"
	@echo "    make test-coverage - Generate coverage report"
	@echo ""
	@echo "  Lint:"
	@echo "    make lint         - Run all linters (golangci-lint)"
	@echo "    make lint-fast    - Lint only changed files"
	@echo "    make vet          - Run go vet"
	@echo "    make errcheck     - Check for unchecked errors"
	@echo "    make fmt          - Format code"
	@echo ""
	@echo "  Pre-commit:"
	@echo "    make pre-commit   - Run all checks before committing"
	@echo "    make check        - Quick checks (faster)"
	@echo ""
