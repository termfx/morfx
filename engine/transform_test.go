package engine

import (
	"context"
	"testing"

	"github.com/oxhq/morfx/core"
)

func TestEngineTransformReplacePreview(t *testing.T) {
	e, err := New(Config{WriteMode: WriteModePreview})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	res, err := e.Transform(context.Background(), TransformRequest{
		Language: "go",
		Source:   "package main\nfunc Hello() { println(\"x\") }\n",
		Op: core.TransformOp{
			Method:      "replace",
			Target:      core.AgentQuery{Type: "call", Name: "println"},
			Replacement: "log.Println",
		},
	})
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}
	if res.MatchCount == 0 {
		t.Fatalf("expected at least one match")
	}
	if res.Modified == "" {
		t.Fatalf("expected modified output")
	}
}
