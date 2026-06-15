package main

import (
	"context"
	"testing"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
)

type fileQueryEngineSpy struct {
	called bool
	req    engine.FileQueryRequest
	result engine.FileQueryResult
	err    error
}

func (s *fileQueryEngineSpy) FileQuery(_ context.Context, req engine.FileQueryRequest) (engine.FileQueryResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

func TestRunFileQueryRoutesToEngine(t *testing.T) {
	root := t.TempDir()
	spy := &fileQueryEngineSpy{
		result: engine.FileQueryResult{
			Results: []core.FileMatch{{
				Match: core.Match{
					Type: "function",
					Name: "Hello",
					Location: core.Location{
						Line:   2,
						Column: 1,
					},
				},
				FilePath: root + "/main.go",
			}},
		},
	}

	payload, err := runFileQuery(context.Background(), spy, fileQueryRequest{
		Scope: &core.FileScope{Path: root, Language: "go"},
		DSL:   "func:Hello",
	})
	if err != nil {
		t.Fatalf("runFileQuery() error = %v", err)
	}
	if !spy.called {
		t.Fatal("expected runFileQuery() to call engine")
	}
	if spy.req.Scope.Path != root {
		t.Fatalf("engine scope path = %q, want %q", spy.req.Scope.Path, root)
	}
	if spy.req.DSL != "func:Hello" {
		t.Fatalf("engine DSL = %q, want %q", spy.req.DSL, "func:Hello")
	}
	if got := payload["matches"]; got != 1 {
		t.Fatalf("payload matches = %v, want 1", got)
	}
}
