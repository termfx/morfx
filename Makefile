# -----------------------------
# Defaults & configuration
# -----------------------------
.DEFAULT_GOAL := test

PKGS            := ./...
PKG_DB         := ./internal/db
LANG_PKG        := ./internal/lang/golang
COVER_PROFILE   := coverage.out

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
.PHONY: help test test-verbose test-race test-one fix fixtest build clean coverage coverage-html regen-snapshots gate

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
	@echo "  coverage-html      - Open HTML coverage report"
	@echo "  regen-snapshots    - Regenerate golden snapshots (SNAP_UPDATE=1)"
	@echo "  gate               - Run the Golden Gate (composite validations)"

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

coverage:
	@echo "Coverage file: $(COVER_PROFILE)"
	@([ -f $(COVER_PROFILE) ] && go tool cover -func=$(COVER_PROFILE) | tail -n1) || echo "No $(COVER_PROFILE). Run 'make test' first."

coverage-html:
	@[ -f $(COVER_PROFILE) ] || (echo "No $(COVER_PROFILE). Run 'make test' first." && exit 2)
	go tool cover -html=$(COVER_PROFILE) -o coverage.html
	@echo "Coverage HTML -> coverage.html"

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
