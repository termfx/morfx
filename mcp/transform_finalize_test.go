package mcp

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
)

func TestFinalizeTransform_AutoApplyStateless(t *testing.T) {
	config := DefaultConfig()
	config.DatabaseURL = "skip"
	config.AutoApplyThreshold = 0.8
	config.LogWriter = io.Discard

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "auto.go")
	original := "package main\n\nfunc greet() string { return \"world\" }\n"
	if err := os.WriteFile(targetFile, []byte(original), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	modified := "package main\n\nfunc greet() string { return \"universe\" }\n"

	req := types.TransformRequest{
		Language:       "go",
		Operation:      "replace",
		Target:         core.AgentQuery{Type: "function", Name: "greet"},
		Path:           targetFile,
		OriginalSource: original,
		Result: core.TransformResult{
			Modified: modified,
			Confidence: core.ConfidenceScore{
				Score: 0.95,
				Level: "high",
			},
			MatchCount: 1,
		},
		ResponseText: "test response",
	}

	resp, err := server.FinalizeTransform(context.Background(), req)
	if err != nil {
		t.Fatalf("finalize transform failed: %v", err)
	}

	if status, _ := resp["result"].(string); status != "applied" {
		t.Fatalf("expected result 'applied', got %v", resp["result"])
	}

	contents, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(contents) != modified {
		t.Fatalf("expected file to be updated, got %s", string(contents))
	}
}

func TestFinalizeTransform_StagingWithoutAutoApply(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "morfx.db")

	config := DefaultConfig()
	config.DatabaseURL = dbPath
	config.AutoApplyThreshold = 0.9
	config.LogWriter = io.Discard

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	targetFile := filepath.Join(tmpDir, "pending.go")
	original := "package main\n\nfunc demo() {}\n"
	if err := os.WriteFile(targetFile, []byte(original), 0o644); err != nil {
		t.Fatalf("failed to write original file: %v", err)
	}

	modified := "package main\n\nfunc demo() { println(\"demo\") }\n"

	req := types.TransformRequest{
		Language:       "go",
		Operation:      "replace",
		Target:         core.AgentQuery{Type: "function", Name: "demo"},
		Path:           targetFile,
		OriginalSource: original,
		Result: core.TransformResult{
			Modified: modified,
			Confidence: core.ConfidenceScore{
				Score: 0.5,
				Level: "medium",
			},
			MatchCount: 1,
		},
		ResponseText: "pending response",
	}

	resp, err := server.FinalizeTransform(context.Background(), req)
	if err != nil {
		t.Fatalf("finalize transform failed: %v", err)
	}

	if status, _ := resp["result"].(string); status != "staged" {
		t.Fatalf("expected result 'staged', got %v", resp["result"])
	}
	if _, hasID := resp["id"].(string); !hasID {
		t.Fatal("expected stage identifier in response")
	}

	contents, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(contents) != original {
		t.Fatalf("expected file to remain unchanged, got %s", string(contents))
	}
}
