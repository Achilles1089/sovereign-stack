VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/sovereign-stack/sovereign/cmd.Version=$(VERSION)"
BINARY := sovereign
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: build install test lint clean release

## Build for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## Install to $GOPATH/bin
install:
	go install $(LDFLAGS) .

## Run all tests
test:
	go test ./... -v -count=1

## Run linter
lint:
	golangci-lint run ./...

## Build for all platforms
release: clean
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		output=$(BINARY)-$$os-$$arch; \
		echo "Building $$output..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o dist/$$output . ; \
	done
	@echo "Release binaries in dist/"

## Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## Run the CLI
run: build
	./$(BINARY)

## Show help
help:
	@echo "Sovereign Stack Build System"
	@echo ""
	@echo "  make build    - Build for current platform"
	@echo "  make install  - Install to GOPATH/bin"
	@echo "  make test     - Run all tests"
	@echo "  make lint     - Run linter"
	@echo "  make release  - Build for all platforms"
	@echo "  make clean    - Clean build artifacts"
