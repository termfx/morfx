package tools

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestRecipeToolDryRunDoesNotMutateFiles(t *testing.T) {
	server := newMockServer()
	tool := NewRecipeTool(server)

	dir := t.TempDir()
	filePath := dir + "/sample.go"
	createTestFile(t, filePath, "package sample\n\nfunc OldName() {}\n")

	params := createTestParams(map[string]any{
		"name":           "rename-old-name",
		"dry_run":        true,
		"min_confidence": 0.85,
		"steps": []map[string]any{
			{
				"name":   "rename old function",
				"method": "replace",
				"scope": map[string]any{
					"path":     dir,
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
	})

	result, err := tool.Handler()(context.Background(), params)
	assertNoError(t, err)

	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if payload["steps_run"] != 1 {
		t.Fatalf("expected one step run, got %+v", payload["steps_run"])
	}
	if payload["dry_run"] != true {
		t.Fatalf("expected dry_run=true, got %+v", payload["dry_run"])
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read sample file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "NewName") {
		t.Fatalf("dry-run recipe mutated file: %s", content)
	}
}
