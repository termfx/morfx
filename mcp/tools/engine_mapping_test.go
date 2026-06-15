package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
	"github.com/oxhq/morfx/mcp/types"
	"github.com/oxhq/morfx/providers"
)

type engineOnlyServer struct {
	engine *engine.Engine
}

func newEngineOnlyServer(t *testing.T, cfg engine.Config) *engineOnlyServer {
	t.Helper()

	e, err := engine.New(cfg)
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}

	return &engineOnlyServer{engine: e}
}

func (s *engineOnlyServer) GetProviders() *providers.Registry {
	panic("legacy provider registry access: tool should use engine")
}

func (s *engineOnlyServer) GetFileProcessor() *core.FileProcessor {
	panic("legacy file processor access: tool should use engine")
}

func (s *engineOnlyServer) GetEngine() *engine.Engine {
	return s.engine
}

func (s *engineOnlyServer) GetStaging() any {
	return nil
}

func (s *engineOnlyServer) GetSafety() any {
	return nil
}

func (s *engineOnlyServer) GetSessionID() string {
	return "engine-only-session"
}

func (s *engineOnlyServer) ReportProgress(ctx context.Context, progress, total float64, message string) {}

func (s *engineOnlyServer) ConfirmApply(ctx context.Context, summary string) error {
	return nil
}

func (s *engineOnlyServer) RequestSampling(ctx context.Context, params map[string]any) (map[string]any, error) {
	return nil, nil
}

func (s *engineOnlyServer) RequestElicitation(ctx context.Context, params map[string]any) (map[string]any, error) {
	return nil, nil
}

func (s *engineOnlyServer) FinalizeTransform(ctx context.Context, req types.TransformRequest) (map[string]any, error) {
	responseText := req.ResponseText
	if responseText == "" {
		responseText = "operation completed"
	}

	resp := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"confidence": req.Result.Confidence.Score,
		"matches":    req.Result.MatchCount,
		"result":     "completed",
	}

	if req.Path != "" {
		resp["path"] = req.Path
	}
	if req.Result.Modified != "" {
		resp["modified"] = req.Result.Modified
	}
	if req.Result.Diff != "" {
		resp["diff"] = req.Result.Diff
	}

	return resp, nil
}

func TestQueryToolRoutesToEngine(t *testing.T) {
	server := newEngineOnlyServer(t, engine.Config{})
	tool := NewQueryTool(server)

	result, err := tool.handle(context.Background(), createTestParams(map[string]any{
		"language": "go",
		"source":   "package main\nfunc Hello() {}\n",
		"query": map[string]any{
			"type": "function",
			"name": "Hello",
		},
	}))
	assertNoError(t, err)

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if got := resultMap["matches"]; got != 1 {
		t.Fatalf("expected 1 match, got %v", got)
	}
}

func TestReplaceToolRoutesToEngine(t *testing.T) {
	server := newEngineOnlyServer(t, engine.Config{})
	tool := NewReplaceTool(server)

	result, err := tool.handle(context.Background(), createTestParams(map[string]any{
		"language": "go",
		"source":   "package main\nfunc Hello() { println(\"x\") }\n",
		"target": map[string]any{
			"type": "call",
			"name": "println",
		},
		"replacement": "log.Println",
	}))
	assertNoError(t, err)

	text := extractContentText(t, result)
	if !strings.Contains(text, "Replace operation completed successfully") {
		t.Fatalf("expected replace response text, got %q", text)
	}
}

func TestFileQueryToolRoutesToEngine(t *testing.T) {
	root := t.TempDir()
	server := newEngineOnlyServer(t, engine.Config{AllowedRoots: []string{root}})
	tool := NewFileQueryTool(server)

	createTestFile(t, filepath.Join(root, "main.go"), "package main\nfunc Hello() {}\n")

	result, err := tool.handle(context.Background(), createTestParams(map[string]any{
		"scope": map[string]any{
			"path":    root,
			"include": []string{"*.go"},
		},
		"query": map[string]any{
			"type": "function",
			"name": "Hello",
		},
	}))
	assertNoError(t, err)

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if got := resultMap["matches"]; got != 1 {
		t.Fatalf("expected 1 match, got %v", got)
	}
}

func TestFileReplaceToolRoutesToEngine(t *testing.T) {
	root := t.TempDir()
	server := newEngineOnlyServer(t, engine.Config{AllowedRoots: []string{root}})
	tool := NewFileReplaceTool(server)

	createTestFile(t, filepath.Join(root, "main.go"), "package main\nfunc Hello() { println(\"x\") }\n")

	result, err := tool.handle(context.Background(), createTestParams(map[string]any{
		"scope": map[string]any{
			"path":    root,
			"include": []string{"*.go"},
		},
		"target": map[string]any{
			"type": "call",
			"name": "println",
		},
		"replacement": "log.Println",
		"dry_run":     true,
	}))
	assertNoError(t, err)

	text := extractContentText(t, result)
	if !strings.Contains(text, "File replace operation completed [DRY RUN]") {
		t.Fatalf("expected dry-run response text, got %q", text)
	}
}

func TestRecipeToolRoutesToEngine(t *testing.T) {
	root := t.TempDir()
	server := newEngineOnlyServer(t, engine.Config{AllowedRoots: []string{root}})
	tool := NewRecipeTool(server)

	filePath := filepath.Join(root, "sample.go")
	createTestFile(t, filePath, "package sample\n\nfunc OldName() {}\n")

	params := map[string]any{
		"name":           "rename-old-name",
		"dry_run":        true,
		"min_confidence": 0.1,
		"steps": []map[string]any{
			{
				"name":   "rename old function",
				"method": "replace",
				"scope": map[string]any{
					"path":     root,
					"include":  []string{"**/*.go"},
					"language": "go",
				},
				"target": map[string]any{
					"type": "function",
					"name": "OldName",
				},
				"replacement": "package sample\n\nfunc NewName() {}\n",
			},
		},
	}
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	result, err := tool.handle(context.Background(), raw)
	assertNoError(t, err)

	text := extractContentText(t, result)
	if !strings.Contains(text, "Recipe rename-old-name completed [DRY RUN]") {
		t.Fatalf("expected recipe response text, got %q", text)
	}
}
