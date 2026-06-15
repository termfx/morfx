package engine

import (
	"context"
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestEngineQueryFromSource(t *testing.T) {
	e, err := New(Config{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	res, err := e.Query(context.Background(), QueryRequest{
		Language: "go",
		Source:   "package main\nfunc Hello() {}\n",
		Query:    core.AgentQuery{Type: "function", Name: "Hello"},
	})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if res.Matches != 1 {
		t.Fatalf("expected 1 match, got %d", res.Matches)
	}
}
