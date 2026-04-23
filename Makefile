# Morfx Makefile
# Enterprise-grade code transformations for AI agents

# Variables
BINARY_NAME = morfx
BUILD_DIR = bin
DIST_DIR = dist
CMD_DIR = cmd/morfx
COVERAGE_DIR = coverage
STANDALONE_TOOLS = query replace delete insert_before insert_after append file_query file_replace file_delete apply
RELEASE_PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64
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
LDFLAGS = -ldflags "-X github.com/oxhq/morfx/internal/buildinfo.Version=$(VERSION) -X github.com/oxhq/morfx/internal/buildinfo.Commit=$(COMMIT) -X github.com/oxhq/morfx/internal/buildinfo.BuildTime=$(BUILD_TIME)"

# Tools versions
GOLANGCI_VERSION = v2.11.4
GOFUMPT_VERSION = latest
GOLINES_VERSION = latest
GOTESTSUM_VERSION = latest
GOLANGCI_LINT = go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
LINT_BASE_REV ?= HEAD~

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m # No Color

.PHONY: all help clean build-standalone smoke-standalone dogfood-tfx verify release-artifacts

## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## all: Run lint, test and build
all: lint test build-standalone

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: Run quality checks (lint, vet, test, standalone smoke)
.PHONY: audit
audit: lint vet test smoke-standalone

## lint: Run golangci-lint with auto-fix
.PHONY: lint
lint:
	@echo "$(GREEN)Running linters...$(NC)"
	@$(GOLANGCI_LINT) run --fix --config .golangci.yml ./...

## lint-strict: Run lint without auto-fix (CI mode)
.PHONY: lint-strict
lint-strict:
	@echo "$(YELLOW)Running strict lint (no auto-fix)...$(NC)"
	@$(GOLANGCI_LINT) run --config .golangci.yml ./...

## lint-ci: Run the blocking CI lint gate
.PHONY: lint-ci
lint-ci:
	@echo "$(YELLOW)Running CI lint gate...$(NC)"
	@changed_files="$$(git diff --name-only $(LINT_BASE_REV) -- '*.go')"; \
	if [ -n "$$changed_files" ]; then \
		unformatted="$$(gofmt -l $$changed_files)"; \
		if [ -n "$$unformatted" ]; then \
			printf '%s\n' "$$unformatted"; \
			exit 1; \
		fi; \
	fi
	@$(GOLANGCI_LINT) run --config .golangci.yml \
		--enable-only errcheck,errorlint,govet,ineffassign,staticcheck,unused \
		--tests=false \
		--new-from-rev=$(LINT_BASE_REV) ./...

## modernize: Auto-fix and modernize entire codebase
.PHONY: modernize
modernize: tools/gofumpt tools/golines tools/gopls
	@echo "$(GREEN)Modernizing codebase...$(NC)"
	@echo "  → Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "  → Running gopls fixes..."
	@$(GOBIN)/gopls fix ./...
	@echo "  → Formatting with gofumpt..."
	@$(GOBIN)/gofumpt -l -w -extra .
	@echo "  → Fixing long lines..."
	@$(GOBIN)/golines -w -m 120 --ignore-generated --reformat-tags .
	@echo "  → Running go fix..."
	@go fix ./...
	@echo "  → Organizing imports..."
	@$(GOBIN)/goimports -w -local github.com/oxhq/morfx .
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

## stress: Run long-haul stress harness
.PHONY: stress
stress:
	@echo "$(GREEN)Running stress harness...$(NC)"
	@tools/scripts/stress.sh

## test-cov-silent: Run tests with coverage (minimal output)
.PHONY: test-cov-silent
test-cov-silent:
	@echo "$(GREEN)Running coverage check...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	@go test -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic $(shell go list ./... | grep -v /tools/) | grep -E '(FAIL|PASS.*\[no tests to run\]|coverage:)' || true
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html 2>/dev/null
	@COVERAGE=$$(go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep "total:" | awk '{print $$3}') && \
	echo "$(GREEN)✓ Total coverage: $$COVERAGE$(NC)"

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

## build-standalone: Build standalone JSON tools
.PHONY: build-standalone
build-standalone: build
	@echo "$(GREEN)Building standalone tools...$(NC)"
	@for tool in $(STANDALONE_TOOLS); do \
		echo "  → $$tool"; \
		go build -o $(BUILD_DIR)/$$tool ./cmd/$$tool; \
	done
	@echo "$(GREEN)✓ Standalone tools built in $(BUILD_DIR)/$(NC)"

## smoke-standalone: Run fixture-based smoke checks for standalone tools
.PHONY: smoke-standalone
smoke-standalone: build-standalone
	@echo "$(GREEN)Running standalone smoke checks...$(NC)"
	@bash tools/scripts/smoke-standalone.sh

## dogfood-tfx: Run standalone Morfx against the local TFX repo without mutating it
.PHONY: dogfood-tfx
dogfood-tfx: build-standalone
	@echo "$(GREEN)Running external dogfood against TFX...$(NC)"
	@bash tools/scripts/dogfood-tfx.sh

## build-all: Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "$(GREEN)Building for all platforms...$(NC)"
	@rm -rf $(BUILD_DIR)/release
	@mkdir -p $(BUILD_DIR)/release
	@for platform in $(RELEASE_PLATFORMS); do \
		goos=$${platform%/*}; \
		goarch=$${platform#*/}; \
		ext=""; \
		if [ "$$goos" = "windows" ]; then ext=".exe"; fi; \
		bundle="$(BUILD_DIR)/release/$(BINARY_NAME)-$$goos-$$goarch"; \
		echo "  → $$bundle"; \
		mkdir -p "$$bundle"; \
		GOOS=$$goos GOARCH=$$goarch go build $(LDFLAGS) -o "$$bundle/$(BINARY_NAME)$$ext" $(CMD_DIR)/main.go; \
		for tool in $(STANDALONE_TOOLS); do \
			GOOS=$$goos GOARCH=$$goarch go build -o "$$bundle/$$tool$$ext" ./cmd/$$tool; \
		done; \
			cp README.md docs/standalone-tools.md docs/standalone-recipes.md LICENSE "$$bundle"/; \
		done
	@echo "$(GREEN)✓ Release bundles ready in $(BUILD_DIR)/release/$(NC)"

## install: Install the binary to GOPATH/bin
.PHONY: install
install:
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	@go install $(LDFLAGS) $(CMD_DIR)/main.go

## verify: Run strict local verification
.PHONY: verify
verify: deps test build-standalone smoke-standalone vet
	@echo "$(GREEN)✓ Verification complete!$(NC)"

## release-artifacts: Build host release artifacts into dist/
.PHONY: release-artifacts
release-artifacts: build-standalone
	@echo "$(GREEN)Packaging release artifacts...$(NC)"
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)
	@printf "%s\n" "$(VERSION)" > $(DIST_DIR)/VERSION
	@go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@for tool in $(STANDALONE_TOOLS); do \
		go build -o $(DIST_DIR)/$$tool ./cmd/$$tool; \
	done
		@cp README.md docs/standalone-tools.md docs/standalone-recipes.md LICENSE tfx.yaml $(DIST_DIR)/
	@if command -v shasum >/dev/null 2>&1; then \
		shasum -a 256 $(DIST_DIR)/* > $(DIST_DIR)/SHA256SUMS; \
	else \
		sha256sum $(DIST_DIR)/* > $(DIST_DIR)/SHA256SUMS; \
	fi
	@echo "$(GREEN)✓ Release artifacts ready in $(DIST_DIR)/$(NC)"

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

## demo: Run AST transformation demo
.PHONY: demo
demo: build
	@echo "$(GREEN)Running Morfx Demo...$(NC)"
	@go run ./demo/cmd run

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
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION); \
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
	@rm -rf $(DIST_DIR)
	@rm -rf artifacts
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
ci: deps test build-standalone smoke-standalone vet
	@echo "$(GREEN)✓ CI pipeline complete!$(NC)"

## release: Create a new release
.PHONY: release
release: verify build-all release-artifacts
	@echo "$(GREEN)Release artifacts ready in $(DIST_DIR)/ and $(BUILD_DIR)/release/$(NC)"

.DEFAULT_GOAL := help
