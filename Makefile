.DEFAULT_GOAL := test

GO_TEST_FLAGS := -covermode=atomic -coverpkg=./... -coverprofile=coverage.out
GO_RACE_FLAGS := -race
GO_VERBOSE    := -v

.PHONY: test test-verbose test-race coverage clean demo build-demo fixtest fix build

test:
	go test ./... $(GO_TEST_FLAGS)

test-verbose:
	go test ./... $(GO_TEST_FLAGS) $(GO_VERBOSE)

test-race:
	go test ./... $(GO_TEST_FLAGS) $(GO_RACE_FLAGS)

test-one:
	go test $(GO_TEST_FLAGS) $(filter-out $@,$(MAKECMDGOALS))
fix:
	goimports -w .
	gofumpt -w .
	gci write -s standard -s default -s "prefix($(shell go list -m))" .
	go mod tidy
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...
# 	golangci-lint run --fix || true

fixtest: fix test

build:
	go build -o bin/fileman ./cmd/fileman

