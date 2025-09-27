//go:build stress

package stress

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/providers"
	golang "github.com/termfx/morfx/providers/golang"
)

func TestStressTransform(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	source := `package main

func Hello() string {
    return "world"
}
`
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	registry := providers.NewRegistry()
	registry.Register(golang.New())

	os.Setenv("MORFX_WORKERS", "16")
	t.Cleanup(func() { os.Unsetenv("MORFX_WORKERS") })

	processor := core.NewFileProcessor(registry)
	scope := core.FileScope{Path: dir, Include: []string{"*.go"}}
	op := core.FileTransformOp{
		TransformOp: core.TransformOp{
			Method: "replace",
			Target: core.AgentQuery{Type: "function", Name: "Hello"},
			Replacement: `func Hello() string {
    return "universe"
}`,
		},
		Scope: scope,
	}

	ctx := context.Background()
	for i := 0; i < 50; i++ {
		result, err := processor.TransformFiles(ctx, op)
		if err != nil {
			t.Fatalf("iteration %d transform error: %v", i, err)
		}
		if result == nil || result.FilesModified == 0 {
			t.Fatalf("iteration %d expected modifications", i)
		}
	}

	prov, _ := registry.Get("go")
	stats := prov.Stats()
	if stats.Active != 0 {
		t.Fatalf("expected parser pool to drain, active=%d", stats.Active)
	}
}
