package main

import (
	"context"
	"testing"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
)

type fileReplaceEngineSpy struct {
	called bool
	req    engine.FileReplaceRequest
	result engine.FileReplaceResult
	err    error
}

func (s *fileReplaceEngineSpy) FileReplace(_ context.Context, req engine.FileReplaceRequest) (engine.FileReplaceResult, error) {
	s.called = true
	s.req = req
	return s.result, s.err
}

func TestRunFileReplaceRoutesToEngine(t *testing.T) {
	root := t.TempDir()
	spy := &fileReplaceEngineSpy{
		result: engine.FileReplaceResult{
			FilesScanned:  2,
			FilesModified: 1,
			TotalMatches:  3,
			Details: []core.FileTransformDetail{{
				FilePath:   root + "/main.go",
				MatchCount: 3,
				Modified:   true,
			}},
		},
	}

	payload, err := runFileReplace(context.Background(), spy, fileReplaceRequest{
		Scope:       &core.FileScope{Path: root, Language: "go"},
		TargetDSL:   "func:Hello",
		Replacement: "func Renamed() {}",
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("runFileReplace() error = %v", err)
	}
	if !spy.called {
		t.Fatal("expected runFileReplace() to call engine")
	}
	if spy.req.Scope.Path != root {
		t.Fatalf("engine scope path = %q, want %q", spy.req.Scope.Path, root)
	}
	if spy.req.Op.Method != "replace" {
		t.Fatalf("engine method = %q, want %q", spy.req.Op.Method, "replace")
	}
	if spy.req.DryRun != true {
		t.Fatal("expected dry run request to be forwarded")
	}
	if got := payload["matches"]; got != 3 {
		t.Fatalf("payload matches = %v, want 3", got)
	}
}
