package mcp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReplaceTool_RealMode_CommitsMatchingFilesEvenWhenSomeFilesDoNotMatch(t *testing.T) {
	t.Setenv("MORFX_STATE_DIR", t.TempDir())

	readOnlyDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatalf("chmod read-only dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755)
	defer os.Chdir(originalWD)
	if err := os.Chdir(readOnlyDir); err != nil {
		t.Fatalf("chdir read-only dir: %v", err)
	}

	workspace := t.TempDir()
	onePath := filepath.Join(workspace, "one.go")
	twoPath := filepath.Join(workspace, "two.go")
	writeTestFile(t, onePath, "package main\n\nfunc main() {}\n")
	writeTestFile(t, twoPath, "package main\n\nfunc helper() {}\n")

	server := newBatchToolTestServer(t)
	defer server.Close()

	params, err := json.Marshal(map[string]any{
		"scope": map[string]any{
			"path":    workspace,
			"include": []string{"*.go"},
		},
		"target": map[string]any{
			"type": "function",
			"name": "main",
		},
		"replacement": "func renamedMain() {}",
		"dry_run":     false,
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	result, err := server.toolRegistry.Execute(context.Background(), "file_replace", params)
	if err != nil {
		t.Fatalf("file_replace failed: %v", err)
	}

	if got := string(mustReadFile(t, onePath)); !strings.Contains(got, "renamedMain") {
		t.Fatalf("expected one.go to be updated, got:\n%s", got)
	}
	if got := string(mustReadFile(t, twoPath)); !strings.Contains(got, "helper") {
		t.Fatalf("expected two.go to remain unchanged, got:\n%s", got)
	}

	text := toolText(t, result)
	if !strings.Contains(text, "Files modified: 1") {
		t.Fatalf("expected success summary for one modified file, got:\n%s", text)
	}
}

func TestFileDeleteTool_RealMode_CommitsMatchingFilesEvenWhenSomeFilesDoNotMatch(t *testing.T) {
	t.Setenv("MORFX_STATE_DIR", t.TempDir())

	readOnlyDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chmod(readOnlyDir, 0o555); err != nil {
		t.Fatalf("chmod read-only dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755)
	defer os.Chdir(originalWD)
	if err := os.Chdir(readOnlyDir); err != nil {
		t.Fatalf("chdir read-only dir: %v", err)
	}

	workspace := t.TempDir()
	onePath := filepath.Join(workspace, "one.go")
	twoPath := filepath.Join(workspace, "two.go")
	writeTestFile(t, onePath, "package main\n\nfunc main() {}\n")
	writeTestFile(t, twoPath, "package main\n\nfunc helper() {}\n")

	server := newBatchToolTestServer(t)
	defer server.Close()

	params, err := json.Marshal(map[string]any{
		"scope": map[string]any{
			"path":    workspace,
			"include": []string{"*.go"},
		},
		"target": map[string]any{
			"type": "function",
			"name": "helper",
		},
		"dry_run": false,
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	result, err := server.toolRegistry.Execute(context.Background(), "file_delete", params)
	if err != nil {
		t.Fatalf("file_delete failed: %v", err)
	}

	if got := string(mustReadFile(t, onePath)); !strings.Contains(got, "func main()") {
		t.Fatalf("expected one.go to remain unchanged, got:\n%s", got)
	}
	if got := string(mustReadFile(t, twoPath)); strings.Contains(got, "helper") {
		t.Fatalf("expected helper to be deleted, got:\n%s", got)
	}

	text := toolText(t, result)
	if !strings.Contains(text, "Files modified: 1") {
		t.Fatalf("expected success summary for one modified file, got:\n%s", text)
	}
}

func TestFileDeleteTool_DryRun_UsesWouldModifyLabel(t *testing.T) {
	t.Setenv("MORFX_STATE_DIR", t.TempDir())

	workspace := t.TempDir()
	twoPath := filepath.Join(workspace, "two.go")
	writeTestFile(t, twoPath, "package main\n\nfunc helper() {}\n")

	server := newBatchToolTestServer(t)
	defer server.Close()

	params, err := json.Marshal(map[string]any{
		"scope": map[string]any{
			"path":    workspace,
			"include": []string{"*.go"},
		},
		"target": map[string]any{
			"type": "function",
			"name": "helper",
		},
		"dry_run": true,
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}

	result, err := server.toolRegistry.Execute(context.Background(), "file_delete", params)
	if err != nil {
		t.Fatalf("file_delete dry run failed: %v", err)
	}

	if got := string(mustReadFile(t, twoPath)); !strings.Contains(got, "helper") {
		t.Fatalf("expected dry run to leave file untouched, got:\n%s", got)
	}

	text := toolText(t, result)
	if !strings.Contains(text, "Files that would be modified: 1") {
		t.Fatalf("expected dry-run wording, got:\n%s", text)
	}
}

func newBatchToolTestServer(t *testing.T) *StdioServer {
	t.Helper()

	config := DefaultConfig()
	config.DatabaseURL = "skip"
	config.Debug = false
	config.LogWriter = io.Discard

	server, err := NewStdioServer(config)
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	return server
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func toolText(t *testing.T, result any) string {
	t.Helper()

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type %T", result)
	}

	contentItems, ok := resultMap["content"].([]map[string]any)
	if ok && len(contentItems) > 0 {
		if text, ok := contentItems[0]["text"].(string); ok {
			return text
		}
	}

	contentAny, ok := resultMap["content"].([]any)
	if ok && len(contentAny) > 0 {
		if item, ok := contentAny[0].(map[string]any); ok {
			if text, ok := item["text"].(string); ok {
				return text
			}
		}
	}

	t.Fatalf("tool response missing text content: %#v", result)
	return ""
}
