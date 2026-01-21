# ACL Engine Makefile
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Module
MODULE=github.com/xflash-panda/acl-engine

# Directories
PKG_DIR=./pkg/...

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build test test-v test-cover bench lint fmt vet tidy clean help

# Default target
all: fmt vet test

## Build
build: ## Build all packages
	$(GOBUILD) $(PKG_DIR)

## Testing
test: ## Run tests
	$(GOTEST) $(PKG_DIR)

test-v: ## Run tests with verbose output
	$(GOTEST) -v $(PKG_DIR)

test-cover: ## Run tests with coverage
	$(GOTEST) -cover -coverprofile=coverage.out $(PKG_DIR)
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-race: ## Run tests with race detector
	$(GOTEST) -race $(PKG_DIR)

bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem $(PKG_DIR)

bench-cpu: ## Run benchmarks with CPU profiling
	$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof $(PKG_DIR)
	@echo "CPU profile: cpu.prof (use 'go tool pprof cpu.prof')"

bench-mem: ## Run benchmarks with memory profiling
	$(GOTEST) -bench=. -benchmem -memprofile=mem.prof $(PKG_DIR)
	@echo "Memory profile: mem.prof (use 'go tool pprof mem.prof')"

## Code Quality
fmt: ## Format code
	$(GOFMT) -s -w .

fmt-check: ## Check code formatting
	@test -z "$$($(GOFMT) -l .)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)

vet: ## Run go vet
	$(GOCMD) vet $(PKG_DIR)

lint: ## Run golangci-lint (requires golangci-lint installed)
	$(GOLINT) run $(PKG_DIR)

## Dependencies
tidy: ## Tidy go modules
	$(GOMOD) tidy

deps: ## Download dependencies
	$(GOMOD) download

deps-update: ## Update dependencies
	$(GOGET) -u $(PKG_DIR)
	$(GOMOD) tidy

## Debugging
debug-test: ## Run tests with dlv debugger (requires dlv installed)
	dlv test $(PKG_DIR)

## Cleaning
clean: ## Clean build artifacts and test cache
	$(GOCMD) clean -testcache
	rm -f coverage.out coverage.html
	rm -f cpu.prof mem.prof
	rm -f *.test

## Help
help: ## Show this help
	@echo "ACL Engine - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
