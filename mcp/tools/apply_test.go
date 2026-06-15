package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxhq/morfx/engine"
)

func TestApplyToolUsesEngineStageLifecycle(t *testing.T) {
	root := t.TempDir()
	server := newEngineOnlyServer(t, engine.Config{
		AllowedRoots: []string{root},
		StageDir:     filepath.Join(root, ".morfx-stages"),
	})
	tool := NewApplyTool(server)

	targetPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(targetPath, []byte("package main\nfunc A() {}\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	stage, err := server.GetEngine().CreateStage(context.Background(), engine.StageCreateRequest{
		Path:      targetPath,
		Language:  "go",
		Operation: "replace",
		Original:  "package main\nfunc A() {}\n",
		Modified:  "package main\nfunc B() {}\n",
	})
	if err != nil {
		t.Fatalf("CreateStage() error = %v", err)
	}

	resp, err := tool.handle(context.Background(), createTestParams(map[string]any{
		"id": stage.ID,
	}))
	assertNoError(t, err)
	if resp == nil {
		t.Fatal("expected response")
	}

	contents, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(contents) != "package main\nfunc B() {}\n" {
		t.Fatalf("expected engine apply to update file, got %q", string(contents))
	}
}

func TestApplyTool_Execute(t *testing.T) {
	server := newMockServer()
	tool := NewApplyTool(server)

	tests := []struct {
		name      string
		setup     func(t *testing.T) []engine.Stage
		params    func(stages []engine.Stage) map[string]any
		expectErr bool
		errMsg    string
		check     func(t *testing.T, result map[string]any, stages []engine.Stage)
	}{
		{
			name: "apply_specific_stage",
			setup: func(t *testing.T) []engine.Stage {
				return []engine.Stage{createPendingEngineStage(t, server, "Specific")}
			},
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{"id": stages[0].ID} },
			expectErr: false,
			check: func(t *testing.T, result map[string]any, stages []engine.Stage) {
				applied := toStringSlice(result["applied"])
				if len(applied) != 1 || applied[0] != stages[0].ID {
					t.Fatalf("expected applied stage %s, got %+v", stages[0].ID, applied)
				}
			},
		},
		{
			name: "apply_specific_stage_with_sampling",
			setup: func(t *testing.T) []engine.Stage {
				server.samplingResults = []map[string]any{{"summary": "ok"}}
				return []engine.Stage{createPendingEngineStage(t, server, "Sampling")}
			},
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{"id": stages[0].ID} },
			expectErr: false,
		},
		{
			name: "apply_latest_stage",
			setup: func(t *testing.T) []engine.Stage {
				return []engine.Stage{
					createPendingEngineStage(t, server, "First"),
					createPendingEngineStage(t, server, "Second"),
					createPendingEngineStage(t, server, "Third"),
				}
			},
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{"latest": true} },
			expectErr: false,
			check: func(t *testing.T, result map[string]any, stages []engine.Stage) {
				applied := toStringSlice(result["applied"])
				if len(applied) != 1 {
					t.Fatalf("expected one applied stage, got %+v", applied)
				}
				if remaining := listPendingEngineStages(t, server); len(remaining) != 2 {
					t.Fatalf("expected 2 pending stages after latest apply, got %d", len(remaining))
				}
			},
		},
		{
			name: "apply_all_stages",
			setup: func(t *testing.T) []engine.Stage {
				return []engine.Stage{
					createPendingEngineStage(t, server, "AllOne"),
					createPendingEngineStage(t, server, "AllTwo"),
					createPendingEngineStage(t, server, "AllThree"),
				}
			},
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{"all": true} },
			expectErr: false,
			check: func(t *testing.T, result map[string]any, stages []engine.Stage) {
				applied := toStringSlice(result["applied"])
				if len(applied) != len(stages) {
					t.Fatalf("expected %d applied stages, got %d", len(stages), len(applied))
				}
				if remaining := listPendingEngineStages(t, server); len(remaining) != 0 {
					t.Fatalf("expected 0 pending stages after apply all, got %d", len(remaining))
				}
			},
		},
		{
			name: "apply_without_params",
			setup: func(t *testing.T) []engine.Stage {
				return []engine.Stage{createPendingEngineStage(t, server, "DefaultLatest")}
			},
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{} },
			expectErr: false,
			check: func(t *testing.T, result map[string]any, stages []engine.Stage) {
				applied := toStringSlice(result["applied"])
				if len(applied) != 1 || applied[0] != stages[0].ID {
					t.Fatalf("expected default latest apply of %s, got %+v", stages[0].ID, applied)
				}
			},
		},
		{
			name:      "apply_non_existent_stage",
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{"id": "non_existent"} },
			expectErr: true,
			errMsg:    "stage not found",
		},
		{
			name: "apply_with_conflicting_params",
			setup: func(t *testing.T) []engine.Stage {
				return []engine.Stage{createPendingEngineStage(t, server, "Conflict")}
			},
			params: func(stages []engine.Stage) map[string]any {
				return map[string]any{"id": stages[0].ID, "all": true, "latest": true}
			},
			expectErr: true,
			errMsg:    "conflicting",
		},
		{
			name:      "apply_when_no_stages",
			params:    func(stages []engine.Stage) map[string]any { return map[string]any{"latest": true} },
			expectErr: true,
			errMsg:    "no stages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearPendingEngineStages(t, server)
			server.samplingRequests = nil
			server.samplingResults = nil
			server.samplingErr = nil

			var stages []engine.Stage
			if tt.setup != nil {
				stages = tt.setup(t)
			}

			result, err := tool.handle(context.Background(), createTestParams(tt.params(stages)))
			if tt.expectErr {
				assertError(t, err, tt.errMsg)
				return
			}

			assertNoError(t, err)
			if result == nil {
				t.Fatal("Expected result but got nil")
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("Expected map result, got %T", result)
			}
			if !hasContentArray(resultMap) {
				t.Fatal("Result should include content array")
			}
			if tt.check != nil {
				tt.check(t, resultMap, stages)
			}
		})
	}
}

func TestApplyTool_Schema(t *testing.T) {
	server := newMockServer()
	tool := NewApplyTool(server)

	if tool.Name() != "apply" {
		t.Errorf("Expected name 'apply', got '%s'", tool.Name())
	}
	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	schema := tool.InputSchema()
	if schema["type"] != "object" {
		t.Errorf("Schema type should be 'object', got %v", schema["type"])
	}

	required, ok := schema["required"].([]string)
	if ok && len(required) > 0 {
		t.Error("Apply tool should not have required parameters")
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	for _, prop := range []string{"id", "latest", "all"} {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Schema missing property '%s'", prop)
		}
	}
}

func TestApplyTool_InvalidJSON(t *testing.T) {
	server := newMockServer()
	tool := NewApplyTool(server)

	_, err := tool.handle(context.Background(), []byte(`{"invalid": json}`))
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
