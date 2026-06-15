package main

import (
	"context"
	"testing"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
)

type replaceEngineSpy struct {
	called bool
	req    engine.TransformRequest
	result engine.TransformResult
	err    error
}

func (s *replaceEngineSpy) Transform(_ context.Context, req engine.TransformRequest) (engine.TransformResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

func TestRunReplaceRoutesToEngine(t *testing.T) {
	src := "package main\nfunc Hello() {}\n"
	spy := &replaceEngineSpy{
		result: engine.TransformResult{
			MatchCount: 1,
			Modified:   "package main\nfunc Renamed() {}\n",
			Diff:       "--- before\n+++ after\n",
			Confidence: core.ConfidenceScore{Score: 0.9, Level: "high"},
		},
	}

	payload, err := runReplace(context.Background(), spy, replaceRequest{
		Language:    "go",
		Source:      &src,
		TargetDSL:   "func:Hello",
		Replacement: "func Renamed() {}",
	})
	if err != nil {
		t.Fatalf("runReplace() error = %v", err)
	}
	if !spy.called {
		t.Fatal("expected runReplace() to call engine")
	}
	if spy.req.Language != "go" {
		t.Fatalf("engine language = %q, want %q", spy.req.Language, "go")
	}
	if spy.req.Source != src {
		t.Fatalf("engine source = %q, want %q", spy.req.Source, src)
	}
	if spy.req.Op.Method != "replace" {
		t.Fatalf("engine method = %q, want %q", spy.req.Op.Method, "replace")
	}
	if spy.req.Op.Replacement != "func Renamed() {}" {
		t.Fatalf("engine replacement = %q, want %q", spy.req.Op.Replacement, "func Renamed() {}")
	}
	if got := payload["matches"]; got != 1 {
		t.Fatalf("payload matches = %v, want 1", got)
	}
}
