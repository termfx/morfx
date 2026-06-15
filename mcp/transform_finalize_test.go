package mcp

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
	"github.com/oxhq/morfx/mcp/types"
)

func TestFinalizeTransform_AutoApplyDelegatesToEngineLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	eng, err := engine.New(engine.Config{
		AllowedRoots:  []string{tmpDir},
		WriteMode:     engine.WriteModeStage,
		EnableStaging: true,
		StageDir:      filepath.Join(tmpDir, ".morfx-stages"),
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	config := DefaultConfig()
	config.DatabaseURL = "skip"
	config.AutoApplyThreshold = 0.8
	config.LogWriter = io.Discard
	config.Engine = eng

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

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
	stageID, ok := resp["id"].(string)
	if !ok || stageID == "" {
		t.Fatalf("expected stage identifier in response, got %v", resp["id"])
	}

	contents, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(contents) != modified {
		t.Fatalf("expected file to be updated, got %s", string(contents))
	}

	stage, err := eng.GetStage(context.Background(), stageID)
	if err != nil {
		t.Fatalf("GetStage() error = %v", err)
	}
	if stage.Status != engine.StageStatusApplied {
		t.Fatalf("expected applied stage, got %s", stage.Status)
	}
}

func TestFinalizeTransform_StagesThroughEngineLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	eng, err := engine.New(engine.Config{
		AllowedRoots:  []string{tmpDir},
		WriteMode:     engine.WriteModeStage,
		EnableStaging: true,
		StageDir:      filepath.Join(tmpDir, ".morfx-stages"),
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	config := DefaultConfig()
	config.DatabaseURL = "skip"
	config.AutoApplyThreshold = 0.9
	config.LogWriter = io.Discard
	config.Engine = eng

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
	stageID, hasID := resp["id"].(string)
	if !hasID || stageID == "" {
		t.Fatal("expected stage identifier in response")
	}

	contents, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(contents) != original {
		t.Fatalf("expected file to remain unchanged, got %s", string(contents))
	}

	stage, err := eng.GetStage(context.Background(), stageID)
	if err != nil {
		t.Fatalf("GetStage() error = %v", err)
	}
	if stage.Status != engine.StageStatusPending {
		t.Fatalf("expected pending stage, got %s", stage.Status)
	}
}
