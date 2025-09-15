# Morfx Makefile
# Enterprise-grade code transformations for AI agents

# Variables
BINARY_NAME = morfx
BUILD_DIR = bin
CMD_DIR = cmd/morfx
COVERAGE_DIR = coverage
GO_FILES = $(shell find . -name '*.go' -type f -not -path "./vendor/*" -not -path "./.git/*")
PACKAGES = $(shell go list ./... | grep -v /vendor/)

# Go tools path
GOPATH = $(shell go env GOPATH)
GOBIN = $(GOPATH)/bin

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Tools versions
GOLANGCI_VERSION = v1.62.2
GOFUMPT_VERSION = latest
GOLINES_VERSION = latest
GOTESTSUM_VERSION = latest

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m # No Color

.PHONY: all help clean

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Run lint, test and build
all: lint test build

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: Run quality checks (lint, vet, test)
.PHONY: audit
audit: lint vet test

## lint: Run golangci-lint with auto-fix
.PHONY: lint
lint: tools/golangci-lint
	@echo "$(GREEN)Running linters...$(NC)"
	@$(GOBIN)/golangci-lint run --fix --config .golangci.yml ./...

## lint-strict: Run lint without auto-fix (CI mode)
.PHONY: lint-strict
lint-strict: tools/golangci-lint
	@echo "$(YELLOW)Running strict lint (no auto-fix)...$(NC)"
	@$(GOBIN)/golangci-lint run --config .golangci.yml ./...

## modernize: Auto-fix and modernize entire codebase
.PHONY: modernize
modernize: tools/gofumpt tools/golines
	@echo "$(GREEN)Modernizing codebase...$(NC)"
	@echo "  → Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "  → Running gopls modernize analyzer..."
	@go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix -test ./...
	@echo "  → Formatting with gofumpt..."
	@$(GOBIN)/gofumpt -l -w -extra .
	@echo "  → Fixing long lines..."
	@$(GOBIN)/golines -w -m 120 --ignore-generated --reformat-tags .
	@echo "  → Running go fix..."
	@go fix ./...
	@echo "  → Organizing imports..."
	@$(GOBIN)/goimports -w -local github.com/termfx/morfx .
	@echo "  → Running go vet..."
	@go vet ./...
	@echo "$(GREEN)✓ Modernization complete!$(NC)"

## fmt: Format code
.PHONY: fmt
fmt: tools/gofumpt
	@echo "$(GREEN)Formatting code...$(NC)"
	@$(GOBIN)/gofumpt -l -w .
	@go fmt ./...

## vet: Run go vet
.PHONY: vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	@go vet ./...

## sec: Run security audit
.PHONY: sec
sec: tools/gosec
	@echo "$(YELLOW)Running security audit...$(NC)"
	@$(GOBIN)/gosec -fmt=json -out=security-report.json ./... || true
	@$(GOBIN)/gosec ./...

# ==================================================================================== #
# TESTING
# ==================================================================================== #

## test: Run all tests
.PHONY: test
test: tools/gotestsum
	@echo "$(GREEN)Running tests...$(NC)"
	@$(GOBIN)/gotestsum --format=testname -- -race -coverprofile=coverage.out ./...

## test-unit: Run unit tests only
.PHONY: test-unit
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	@go test -short -race ./...

## test-integration: Run integration tests
.PHONY: test-integration
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	@go test -race -tags=integration ./...

## test-coverage: Run tests with coverage report
.PHONY: test-coverage
test-coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@go test -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out
	@echo "$(GREEN)Coverage report: $(COVERAGE_DIR)/coverage.html$(NC)"

## coverage-check: Enforce coverage thresholds
.PHONY: coverage-check
coverage-check: test-coverage
	@echo "$(GREEN)Checking coverage thresholds...$(NC)"
	@go run tools/coverage-check/main.go $(COVERAGE_DIR)/coverage.out

## coverage-badge: Generate coverage badge
.PHONY: coverage-badge
coverage-badge: test-coverage
	@echo "$(GREEN)Generating coverage badge...$(NC)"
	@go run tools/coverage-badge.go $(COVERAGE_DIR)/coverage.out $(COVERAGE_DIR)/badge.svg

## coverage-report: Generate detailed coverage report with component breakdown
.PHONY: coverage-report
coverage-report: test-coverage
	@echo "$(GREEN)Generating detailed coverage report...$(NC)"
	@go run tools/coverage-report/main.go $(COVERAGE_DIR)/coverage.out $(COVERAGE_DIR)/report.md

## coverage-ci: Coverage for CI/CD (with stricter thresholds)
.PHONY: coverage-ci
coverage-ci:
	@echo "$(GREEN)Running CI coverage checks...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@go test -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep "total:" | awk '{print "Total coverage: " $$3}'
	@go run tools/coverage-check/main.go $(COVERAGE_DIR)/coverage.out --strict

## bench: Run benchmarks
.PHONY: bench
bench:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build: Build the binary
.PHONY: build
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "$(GREEN)✓ Binary: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-all: Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "$(GREEN)Building for all platforms...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)/main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)/main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)/main.go
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)/main.go
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)/main.go

## install: Install the binary to GOPATH/bin
.PHONY: install
install:
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	@go install $(LDFLAGS) $(CMD_DIR)/main.go

# ==================================================================================== #
# DATABASE
# ==================================================================================== #

## db-reset: Reset SQLite database
.PHONY: db-reset
db-reset:
	@echo "$(YELLOW)Resetting SQLite database...$(NC)"
	@rm -f ./.morfx/db/morfx.db
	@echo "$(GREEN)✓ Database reset. Will be recreated on next run.$(NC)"

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: Run the MCP server
.PHONY: run
run: build
	@echo "$(GREEN)Running MCP server...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME) mcp --debug

## watch: Run with file watcher (requires entr)
.PHONY: watch
watch:
	@echo "$(GREEN)Watching for changes...$(NC)"
	@find . -name '*.go' | entr -r make run

## dev: Run all development tools
.PHONY: dev
dev: modernize lint test build
	@echo "$(GREEN)✓ Development checks complete!$(NC)"

# ==================================================================================== #
# TOOLS INSTALLATION
# ==================================================================================== #

## tools: Install all development tools
.PHONY: tools
tools: tools/golangci-lint tools/gofumpt tools/golines tools/gosec tools/gotestsum tools/goimports tools/gopls
	@echo "$(GREEN)✓ All tools installed!$(NC)"

.PHONY: tools/golangci-lint
tools/golangci-lint:
	@if ! [ -f $(GOBIN)/golangci-lint ]; then \
		echo "$(YELLOW)Installing golangci-lint $(GOLANGCI_VERSION)...$(NC)"; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_VERSION); \
	fi

.PHONY: tools/gofumpt
tools/gofumpt:
	@if ! [ -f $(GOBIN)/gofumpt ]; then \
		echo "$(YELLOW)Installing gofumpt...$(NC)"; \
		go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION); \
	fi

.PHONY: tools/golines
tools/golines:
	@if ! [ -f $(GOBIN)/golines ]; then \
		echo "$(YELLOW)Installing golines...$(NC)"; \
		go install github.com/segmentio/golines@$(GOLINES_VERSION); \
	fi

.PHONY: tools/gosec
tools/gosec:
	@if ! [ -f $(GOBIN)/gosec ]; then \
		echo "$(YELLOW)Installing gosec...$(NC)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi

.PHONY: tools/gotestsum
tools/gotestsum:
	@if ! [ -f $(GOBIN)/gotestsum ]; then \
		echo "$(YELLOW)Installing gotestsum...$(NC)"; \
		go install gotest.tools/gotestsum@$(GOTESTSUM_VERSION); \
	fi

.PHONY: tools/goimports
tools/goimports:
	@if ! [ -f $(GOBIN)/goimports ]; then \
		echo "$(YELLOW)Installing goimports...$(NC)"; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi

.PHONY: tools/gopls
tools/gopls:
	@if ! [ -f $(GOBIN)/gopls ]; then \
		echo "$(YELLOW)Installing gopls...$(NC)"; \
		go install golang.org/x/tools/gopls@latest; \
	fi


# ==================================================================================== #
# MAINTENANCE
# ==================================================================================== #

## clean: Clean build artifacts
.PHONY: clean
clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -rf $(COVERAGE_DIR)
	@rm -f coverage.out security-report.json
	@go clean -testcache
	@echo "$(GREEN)✓ Clean complete!$(NC)"

## deps: Download and tidy dependencies
.PHONY: deps
deps:
	@echo "$(GREEN)Tidying dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@go mod verify

## update: Update all dependencies
.PHONY: update
update:
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	@go get -u ./...
	@go mod tidy

## version: Show version info
.PHONY: version
version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"

# ==================================================================================== #
# CI/CD TARGETS
# ==================================================================================== #

## ci: Run CI pipeline
.PHONY: ci
ci: deps lint-strict test build
	@echo "$(GREEN)✓ CI pipeline complete!$(NC)"

## release: Create a new release
.PHONY: release
release: audit build-all
	@echo "$(GREEN)Release artifacts ready in $(BUILD_DIR)/$(NC)"

.DEFAULT_GOAL := help