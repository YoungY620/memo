.PHONY: build build-all clean install

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
	rm -rf dist $(BINARY)

install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed $(BINARY) to /usr/local/bin"
