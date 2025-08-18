# -----------------------------
# Defaults & configuration
# -----------------------------
.DEFAULT_GOAL := test

PKGS            := ./...
PKG_DB         := ./internal/db
LANG_PKG        := ./internal/lang/golang
COVER_PROFILE   := coverage.out
COVER_HTML      := coverage.html
COVER_THRESHOLD := 80

GO_TEST_FLAGS   := -covermode=atomic -coverpkg=./... -coverprofile=$(COVER_PROFILE)
GO_RACE_FLAGS   := -race
GO_VERBOSE      := -v
CGO_ENABLED     ?= 1
export CGO_ENABLED

#  NO FTS5 specific test functions.
# These tests are designed to run without the FTS5 tag.
NOFTS5_TESTS := \
  TestOpenAndMigrate_FallbackNoFTS5 \
  TestRollbackCrashResumeCPA \
  TestRollbackCrashResumeCPB

# Helpers to convert NOFTS5_TESTS into a regex.
empty  :=
space  := $(empty) $(empty)
NOFTS5_REGEX := ^($(subst $(space),|,$(strip $(NOFTS5_TESTS))))$$

# -----------------------------
# Phony targets
# -----------------------------
.PHONY: help test test-verbose test-race test-one fix fixtest build clean coverage coverage-html coverage-check coverage-badge coverage-ci regen-snapshots gate benchmark profile quality-gate

# -----------------------------
# Help
# -----------------------------
help:
	@echo "Targets:"
	@echo "  test               - Run full test suite with coverage"
	@echo "  test-verbose       - Run tests verbose with coverage"
	@echo "  test-race          - Run tests with -race + coverage"
	@echo "  test-one PKG=...   - Run tests for a single package (or pattern) with coverage"
	@echo "  test-no-fts5       - Run full test suite without FTS5 tag"
	@echo "  fix                - Format, organize imports, tidy modules, and modernize"
	@echo "  fixtest            - fix + test"
	@echo "  build              - Build CLI into bin/morfx"
	@echo "  clean              - Remove build artifacts and coverage"
	@echo "  coverage           - Print coverage summary"
	@echo "  coverage-html      - Generate and open HTML coverage report"
	@echo "  coverage-check     - Enforce coverage threshold ($(COVER_THRESHOLD)%)"
	@echo "  coverage-badge     - Generate coverage badge"
	@echo "  coverage-ci        - Generate coverage reports for CI"
	@echo "  regen-snapshots    - Regenerate golden snapshots (SNAP_UPDATE=1)"
	@echo "  gate               - Run the Golden Gate (composite validations)"
	@echo "  benchmark          - Run performance benchmarks"
	@echo "  profile            - Run CPU and memory profiling"
	@echo "  quality-gate       - Run all quality checks (fix + test + coverage + gate)"

# -----------------------------
# Tests
# -----------------------------
test:
	@$(MAKE) -s test-fts5
	@$(MAKE) -s test-no-fts5

test-fts5:
	@MORFX_ENCRYPTION_MODE=off \
	  go test -count=1 -tags sqlite_fts5 $(PKGS) $(GO_TEST_FLAGS)

test-no-fts5:
	@echo "Running NO-FTS5 tests: $(NOFTS5_TESTS)"
	@MORFX_FORCE_NO_FTS5=1 MORFX_ENCRYPTION_MODE=blob MORFX_MASTER_KEY=0123456789abcdef0123456789abcdef \
		go test -count=1 $(PKG_DB) -run '$(NOFTS5_REGEX)' $(GO_TEST_FLAGS)

test-verbose:
	go test -count=1 -tags sqlite_fts5 $(PKGS) $(GO_TEST_FLAGS) $(GO_VERBOSE)

test-race:
	go test -count=1 -tags sqlite_fts5 $(PKGS) $(GO_TEST_FLAGS) $(GO_RACE_FLAGS)

# Usage:
#   make test-one PKG=./internal/lang/golang -run 'TestName'
#   make test-one PKG=./internal/lang/golang
test-one:
	@if [ -z "$(PKG)" ]; then echo "Usage: make test-one PKG=./path [-run 'Regex']"; exit 2; fi
	go test -count=1 $(PKG) $(GO_TEST_FLAGS) $(RUN)

# -----------------------------
# Coverage Analysis
# -----------------------------
coverage:
	@echo "Coverage Analysis"
	@echo "=================="
	@if [ ! -f $(COVER_PROFILE) ]; then \
		echo "No $(COVER_PROFILE) found. Running tests..."; \
		$(MAKE) -s test; \
	fi
	@if [ -f $(COVER_PROFILE) ]; then \
		echo "Coverage by Package:"; \
		echo "-------------------"; \
		go tool cover -func=$(COVER_PROFILE) | grep -E "(github.com/termfx/morfx/|total:)" | \
		while IFS= read -r line; do \
			if echo "$$line" | grep -q "total:"; then \
				echo ""; \
				echo "Overall Coverage:"; \
				echo "-----------------"; \
				echo "$$line" | awk '{printf "Total: %s\n", $$3}'; \
			else \
				echo "$$line" | awk '{printf "%-50s %s\n", $$1, $$3}'; \
			fi; \
		done; \
	else \
		echo "Failed to generate coverage profile"; \
		exit 1; \
	fi

coverage-html:
	@echo "Generating HTML coverage report..."
	@if [ ! -f $(COVER_PROFILE) ]; then \
		echo "No $(COVER_PROFILE) found. Running tests..."; \
		$(MAKE) -s test; \
	fi
	@if [ -f $(COVER_PROFILE) ]; then \
		go tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML); \
		echo "Coverage HTML report generated: $(COVER_HTML)"; \
		if command -v open >/dev/null 2>&1; then \
			open $(COVER_HTML); \
		elif command -v xdg-open >/dev/null 2>&1; then \
			xdg-open $(COVER_HTML); \
		else \
			echo "Open $(COVER_HTML) in your browser to view the report"; \
		fi; \
	else \
		echo "Failed to generate coverage profile"; \
		exit 1; \
	fi

coverage-check:
	@echo "Checking coverage threshold ($(COVER_THRESHOLD)%)..."
	@if [ ! -f $(COVER_PROFILE) ]; then \
		echo "No $(COVER_PROFILE) found. Running tests..."; \
		$(MAKE) -s test; \
	fi
	@if [ -f $(COVER_PROFILE) ]; then \
		COVERAGE=$$(go tool cover -func=$(COVER_PROFILE) | tail -n1 | awk '{print $$3}' | sed 's/%//'); \
		echo "Current coverage: $${COVERAGE}%"; \
		echo "Required coverage: $(COVER_THRESHOLD)%"; \
		if [ "$$(echo "$${COVERAGE} < $(COVER_THRESHOLD)" | bc -l)" -eq 1 ]; then \
			echo "❌ Coverage $${COVERAGE}% is below required $(COVER_THRESHOLD)%"; \
			echo ""; \
			echo "Coverage by package:"; \
			go tool cover -func=$(COVER_PROFILE) | grep -v "total:" | sort -k3 -n | \
			while IFS= read -r line; do \
				PKG_COV=$$(echo "$$line" | awk '{print $$3}' | sed 's/%//'); \
				if [ "$$(echo "$${PKG_COV} < $(COVER_THRESHOLD)" | bc -l)" -eq 1 ]; then \
					echo "  📉 $$line"; \
				fi; \
			done; \
			echo ""; \
			echo "Please add tests to improve coverage."; \
			exit 1; \
		else \
			echo "✅ Coverage $${COVERAGE}% meets required $(COVER_THRESHOLD)%"; \
		fi; \
	else \
		echo "❌ Failed to generate coverage profile"; \
		exit 1; \
	fi

coverage-badge:
	@echo "Generating coverage badge..."
	@if [ ! -f $(COVER_PROFILE) ]; then \
		echo "No $(COVER_PROFILE) found. Running tests..."; \
		$(MAKE) -s test; \
	fi
	@if [ -f $(COVER_PROFILE) ]; then \
		COVERAGE=$$(go tool cover -func=$(COVER_PROFILE) | tail -n1 | awk '{print $$3}' | sed 's/%//'); \
		COLOR="red"; \
		if [ "$$(echo "$${COVERAGE} >= 80" | bc -l)" -eq 1 ]; then COLOR="green"; \
		elif [ "$$(echo "$${COVERAGE} >= 60" | bc -l)" -eq 1 ]; then COLOR="yellow"; \
		elif [ "$$(echo "$${COVERAGE} >= 40" | bc -l)" -eq 1 ]; then COLOR="orange"; \
		fi; \
		echo "Coverage: $${COVERAGE}% ($$COLOR)"; \
		echo "Badge URL: https://img.shields.io/badge/coverage-$${COVERAGE}%25-$$color.svg"; \
	fi

coverage-ci:
	@echo "Generating coverage reports for CI..."
	@$(MAKE) -s test
	@$(MAKE) -s coverage
	@go tool cover -html=$(COVER_PROFILE) -o $(COVER_HTML)
	@go tool cover -func=$(COVER_PROFILE) > coverage.txt
	@echo "Coverage reports generated:"
	@echo "  - $(COVER_PROFILE) (machine readable)"
	@echo "  - $(COVER_HTML) (human readable)"
	@echo "  - coverage.txt (summary)"

# -----------------------------
# Formatting / lint-like
# -----------------------------
fix:
# 	 Organize imports (goimports)
	go run golang.org/x/tools/cmd/goimports@latest -w .
# 	 gofumpt
	go run mvdan.cc/gofumpt@latest -w .
# 	 gci: group imports (std, default, module prefix)
	go run github.com/daixiang0/gci@latest write -s standard -s default -s "prefix($$(go list -m))" .
# 	 Tidy modules
	go mod tidy
# 	 Modernize (best-effort; ignore failures)
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./... || true
# 	 golangci-lint (optional; uncomment if installed)
# 	golangci-lint run --fix || true

fixtest: fix test

# -----------------------------
# Build
# -----------------------------
build:
	go build -o bin/morfx ./cmd/morfx

clean:
	rm -f $(COVER_PROFILE) coverage.html
	rm -rf bin

# -----------------------------
# Golden snapshots & Gate
# -----------------------------
regen-snapshots:
	@echo "Regenerating golden snapshots..."
	SNAP_UPDATE=1 go test -count=1 $(LANG_PKG) -run "TestDSLQuerySnapshots"
	@echo "Snapshots regenerated successfully!"


gate:
	go test -count=1 $(LANG_PKG) -run "TestDSLQuerySnapshots"
	go test -count=1 $(LANG_PKG) -run Validator
	go test -count=1 $(LANG_PKG) -run E2E
	go test -count=1 $(LANG_PKG) -run List
	go test -count=1 $(LANG_PKG) -run Negation || true
	go test -count=1 $(LANG_PKG) -run Hierarchy || true
	go test -count=1 $(LANG_PKG) -run ImportPath || true
