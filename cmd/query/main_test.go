package main

import (
	"context"
	"testing"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
)

type queryEngineSpy struct {
	called bool
	req    engine.QueryRequest
	result engine.QueryResult
	err    error
}

func (s *queryEngineSpy) Query(_ context.Context, req engine.QueryRequest) (engine.QueryResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

func TestRunQueryRoutesToEngine(t *testing.T) {
	src := "package main\nfunc Hello() {}\n"
	spy := &queryEngineSpy{
		result: engine.QueryResult{
			Matches: 1,
			Results: []core.Match{{
				Type: "function",
				Name: "Hello",
				Location: core.Location{
					Line:   2,
					Column: 1,
				},
			}},
		},
	}

	payload, err := runQuery(context.Background(), spy, queryRequest{
		Language: "go",
		Source:   &src,
		DSL:      "func:Hello",
	})
	if err != nil {
		t.Fatalf("runQuery() error = %v", err)
	}
	if !spy.called {
		t.Fatal("expected runQuery() to call engine")
	}
	if spy.req.Language != "go" {
		t.Fatalf("engine language = %q, want %q", spy.req.Language, "go")
	}
	if spy.req.Source != src {
		t.Fatalf("engine source = %q, want %q", spy.req.Source, src)
	}
	if spy.req.DSL != "func:Hello" {
		t.Fatalf("engine DSL = %q, want %q", spy.req.DSL, "func:Hello")
	}
	if got := payload["matches"]; got != 1 {
		t.Fatalf("payload matches = %v, want 1", got)
	}
}
