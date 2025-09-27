package tools

import (
	"context"
	"testing"
	"time"
)

func TestApplyTool_Execute(t *testing.T) {
	server := newMockServer()
	setStaging(server, true)
	tool := NewApplyTool(server)

	// Create some test stages
	stage1 := map[string]any{
		"id":        "stage1",
		"operation": "replace",
		"timestamp": time.Now().Unix(),
	}
	stage2 := map[string]any{
		"id":        "stage2",
		"operation": "delete",
		"timestamp": time.Now().Unix(),
	}
	stage3 := map[string]any{
		"id":        "stage3",
		"operation": "insert",
		"timestamp": time.Now().Unix(),
	}

	addTestStage(server, "stage1", stage1)
	addTestStage(server, "stage2", stage2)
	addTestStage(server, "stage3", stage3)

	tests := []struct {
		name      string
		params    map[string]any
		expectErr bool
		errMsg    string
		setup     func()
	}{
		{
			name: "apply_specific_stage",
			params: map[string]any{
				"id": "stage1",
			},
			expectErr: false,
		},
		{
			name: "apply_specific_stage_with_sampling",
			params: map[string]any{
				"id": "stage2",
			},
			expectErr: false,
			setup: func() {
				server.samplingResults = []map[string]any{{"summary": "ok"}}
			},
		},
		{
			name: "apply_latest_stage",
			params: map[string]any{
				"latest": true,
			},
			expectErr: false,
		},
		{
			name: "apply_all_stages",
			params: map[string]any{
				"all": true,
			},
			expectErr: false,
		},
		{
			name:      "apply_without_params",
			params:    map[string]any{},
			expectErr: false, // Should default to latest
		},
		{
			name: "apply_non_existent_stage",
			params: map[string]any{
				"id": "non_existent",
			},
			expectErr: true,
			errMsg:    "failed to load stage",
		},
		{
			name: "apply_with_conflicting_params",
			params: map[string]any{
				"id":     "stage1",
				"all":    true,
				"latest": true,
			},
			expectErr: true,
			errMsg:    "conflicting",
		},
		{
			name: "apply_when_no_stages",
			params: map[string]any{
				"latest": true,
			},
			expectErr: true,
			errMsg:    "no stages",
			setup: func() {
				clearStages(server)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset stages before each test
			clearStages(server)
			addTestStage(server, "stage1", stage1)
			addTestStage(server, "stage2", stage2)
			addTestStage(server, "stage3", stage3)

			if tt.setup != nil {
				tt.setup()
			}

			params := createTestParams(tt.params)
			server.samplingRequests = nil
			result, err := tool.handle(context.Background(), params)

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

			structured, ok := resultMap["structuredContent"].(map[string]any)
			if !ok {
				t.Fatal("structuredContent missing from apply result")
			}
			mode, _ := structured["mode"].(string)
			if mode == "" {
				t.Error("structuredContent.mode should be populated")
			}

			applied := toStringSlice(resultMap["applied"])
			if mode != "all" && len(applied) == 0 {
				t.Errorf("expected applied stages for mode %s", mode)
			}

			switch tt.name {
			case "apply_specific_stage_with_sampling":
				if len(server.samplingRequests) == 0 {
					t.Error("expected sampling request to be issued")
				}
				if _, ok := structured["sampling"].(map[string]any); !ok {
					t.Error("expected sampling data in structuredContent")
				}
			case "apply_all_stages":
				if len(applied) == 0 {
					t.Error("expected applied stages for all mode")
				}
			}

			if !hasContentArray(resultMap) {
				t.Error("Result should include content array")
			}
			server.samplingResults = nil
			server.samplingErr = nil
		})
	}
}

func TestApplyTool_StagingDisabled(t *testing.T) {
	server := newMockServer()
	setStaging(server, false) // Staging disabled
	tool := NewApplyTool(server)

	params := createTestParams(map[string]any{
		"latest": true,
	})

	_, err := tool.handle(context.Background(), params)
	assertError(t, err, "staging is not enabled")
}

func TestApplyTool_ApplyOrder(t *testing.T) {
	server := newMockServer()
	setStaging(server, true)
	tool := NewApplyTool(server)

	// Create stages with timestamps
	now := time.Now()
	stages := []struct {
		id        string
		timestamp time.Time
	}{
		{"stage1", now.Add(-3 * time.Minute)},
		{"stage2", now.Add(-2 * time.Minute)},
		{"stage3", now.Add(-1 * time.Minute)},
	}

	for _, s := range stages {
		addTestStage(server, s.id, map[string]any{
			"id":        s.id,
			"timestamp": s.timestamp.Unix(),
		})
	}

	// Apply all stages
	params := createTestParams(map[string]any{
		"all": true,
	})

	result, err := tool.handle(context.Background(), params)
	assertNoError(t, err)

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected apply result map, got %T", result)
	}
	structured, ok := resultMap["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("structuredContent missing from apply all response")
	}
	if mode, _ := structured["mode"].(string); mode != "all" {
		t.Errorf("expected mode 'all', got %s", structured["mode"])
	}
	count := 0
	switch v := structured["appliedCount"].(type) {
	case int:
		count = v
	case float64:
		count = int(v)
	default:
		t.Errorf("expected numeric appliedCount, got %T", structured["appliedCount"])
	}
	if count != len(stages) {
		t.Errorf("Expected %d stages applied, got %d", len(stages), count)
	}
	applied := toStringSlice(resultMap["applied"])
	if len(applied) != len(stages) {
		t.Errorf("expected %d applied stage IDs, got %d", len(stages), len(applied))
	}

	// Verify stages were cleared after application
	stageCount := getStageCount(server)
	if stageCount != 0 {
		t.Errorf("Expected 0 stages after apply all, got %d", stageCount)
	}
}

func TestApplyTool_Schema(t *testing.T) {
	server := newMockServer()
	tool := NewApplyTool(server)

	// Verify tool metadata
	if tool.Name() != "apply" {
		t.Errorf("Expected name 'apply', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Tool should have a description")
	}

	schema := tool.InputSchema()

	// Verify schema structure
	if schema["type"] != "object" {
		t.Errorf("Schema type should be 'object', got %v", schema["type"])
	}

	// Apply tool should have optional parameters
	required, ok := schema["required"].([]string)
	if ok && len(required) > 0 {
		t.Error("Apply tool should not have required parameters")
	}

	// Verify properties exist
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Schema should have properties")
	}

	// Check for expected properties
	expectedProps := []string{"id", "latest", "all"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Schema missing property '%s'", prop)
		}
	}
}

func TestApplyTool_InvalidJSON(t *testing.T) {
	server := newMockServer()
	tool := NewApplyTool(server)

	// Test with invalid JSON
	invalidJSON := []byte(`{"invalid": json}`)
	_, err := tool.handle(context.Background(), invalidJSON)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
