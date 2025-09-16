package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/termfx/morfx/models"
)

// setupTestServerWithDB creates a test server with an in-memory database
func setupTestServerWithDB(t *testing.T) *StdioServer {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.Session{}, &models.Stage{}, &models.Apply{}); err != nil {
		t.Fatalf("Failed to migrate database: %v", err)
	}

	config := DefaultConfig()
	config.DatabaseURL = ":memory:"
	config.Debug = false

	server := &StdioServer{
		config:  config,
		db:      db,
		staging: NewStagingManager(db, config),
		session: &models.Session{ID: "test-session"},
	}

	// Create the session in database
	db.Create(server.session)

	return server
}

func TestHandleApplyTool_InvalidParams(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Test with invalid JSON
	invalidJSON := json.RawMessage(`{"invalid": json}`)

	result, err := server.handleApplyTool(invalidJSON)

	if result != nil {
		t.Error("Expected nil result for invalid params")
	}

	if err == nil {
		t.Error("Expected error for invalid params")
	}

	mcpErr, ok := err.(*MCPError)
	if !ok {
		t.Errorf("Expected MCPError, got %T", err)
	} else if mcpErr.Code != InvalidParams {
		t.Errorf("Expected InvalidParams error code, got %d", mcpErr.Code)
	}
}

func TestHandleApplyTool_NoStaging(t *testing.T) {
	// Create server without staging
	config := DefaultConfig()
	config.DatabaseURL = "skip"

	server := &StdioServer{
		config:  config,
		staging: nil, // No staging available
	}

	params := json.RawMessage(`{"id": "test-stage"}`)

	result, err := server.handleApplyTool(params)

	if result != nil {
		t.Error("Expected nil result when staging unavailable")
	}

	if err == nil {
		t.Error("Expected error when staging unavailable")
	}

	mcpErr, ok := err.(*MCPError)
	if !ok {
		t.Errorf("Expected MCPError, got %T", err)
	} else if mcpErr.Code != InternalError {
		t.Errorf("Expected InternalError, got %d", mcpErr.Code)
	}
}

func TestHandleApplyTool_SpecificStage(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create a test stage
	stage := &models.Stage{
		ID:              "test-stage-123",
		SessionID:       server.session.ID,
		Operation:       "replace",
		TargetType:      "function",
		TargetName:      "testFunction",
		ConfidenceLevel: "high",
		ConfidenceScore: 0.95,
		Status:          "pending",
		ExpiresAt:       time.Now().Add(1 * time.Hour),
		Diff:            "+new line\n-old line",
	}

	err := server.staging.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	// Test applying specific stage
	params := json.RawMessage(`{"id": "test-stage-123"}`)

	result, err := server.handleApplyTool(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "applied" {
		t.Errorf("Expected result 'applied', got %v", resultMap["result"])
	}

	// Verify stage was applied by checking the content
	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Error("Expected content in response")
	} else {
		text, ok := content[0]["text"].(string)
		if !ok {
			t.Error("Expected text content")
		} else if !contains(text, "Applied stage test-stage-123") {
			t.Error("Expected applied stage confirmation in response")
		}
	}
}

func TestApplySpecificStage_StageNotFound(t *testing.T) {
	server := setupTestServerWithDB(t)

	result, err := server.applySpecificStage("nonexistent-stage")

	if result != nil {
		t.Error("Expected nil result for nonexistent stage")
	}

	if err == nil {
		t.Error("Expected error for nonexistent stage")
	}

	mcpErr, ok := err.(*MCPError)
	if !ok {
		t.Errorf("Expected MCPError, got %T", err)
	} else if mcpErr.Code != StageNotFound {
		t.Errorf("Expected StageNotFound error, got %d", mcpErr.Code)
	}
}

func TestApplySpecificStage_AlreadyApplied(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create an already applied stage
	stage := &models.Stage{
		ID:        "applied-stage",
		SessionID: server.session.ID,
		Status:    "applied",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := server.staging.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	result, err := server.applySpecificStage("applied-stage")

	if result != nil {
		t.Error("Expected nil result for already applied stage")
	}

	if err == nil {
		t.Error("Expected error for already applied stage")
	}

	mcpErr, ok := err.(*MCPError)
	if !ok {
		t.Errorf("Expected MCPError, got %T", err)
	} else if mcpErr.Code != AlreadyApplied {
		t.Errorf("Expected AlreadyApplied error, got %d", mcpErr.Code)
	}
}

func TestApplySpecificStage_Expired(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create an expired stage
	stage := &models.Stage{
		ID:        "expired-stage",
		SessionID: server.session.ID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	err := server.staging.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	result, err := server.applySpecificStage("expired-stage")

	if result != nil {
		t.Error("Expected nil result for expired stage")
	}

	if err == nil {
		t.Error("Expected error for expired stage")
	}

	mcpErr, ok := err.(*MCPError)
	if !ok {
		t.Errorf("Expected MCPError, got %T", err)
	} else if mcpErr.Code != StageExpired {
		t.Errorf("Expected StageExpired error, got %d", mcpErr.Code)
	}
}

func TestApplyAllStages_NoPendingStages(t *testing.T) {
	server := setupTestServerWithDB(t)

	result, err := server.applyAllStages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "completed" {
		t.Errorf("Expected result 'completed', got %v", resultMap["result"])
	}

	// Check content mentions no pending stages
	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Error("Expected content in response")
	} else {
		text, ok := content[0]["text"].(string)
		if !ok {
			t.Error("Expected text content")
		} else if !contains(text, "No pending stages") {
			t.Error("Expected 'No pending stages' message")
		}
	}
}

func TestApplyAllStages_WithMixedStages(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create multiple stages with different statuses
	stages := []*models.Stage{
		{
			ID:        "pending-stage-1",
			SessionID: server.session.ID,
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "expired-stage",
			SessionID: server.session.ID,
			Status:    "pending",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		},
		{
			ID:        "pending-stage-2",
			SessionID: server.session.ID,
			Status:    "pending",
			ExpiresAt: time.Now().Add(2 * time.Hour),
		},
	}

	for _, stage := range stages {
		err := server.staging.CreateStage(stage)
		if err != nil {
			t.Fatalf("Failed to create stage %s: %v", stage.ID, err)
		}
	}

	result, err := server.applyAllStages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "completed" {
		t.Errorf("Expected result 'completed', got %v", resultMap["result"])
	}

	// Should have applied 2 stages and failed 1 (expired)
	applied, ok := resultMap["applied"].(int)
	if !ok || applied != 2 {
		t.Errorf("Expected 2 applied stages, got %v", resultMap["applied"])
	}

	failed, ok := resultMap["failed"].(int)
	if !ok || failed != 1 {
		t.Errorf("Expected 1 failed stage, got %v", resultMap["failed"])
	}
}

func TestApplyLatestStage_NoPendingStages(t *testing.T) {
	server := setupTestServerWithDB(t)

	result, err := server.applyLatestStage()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "completed" {
		t.Errorf("Expected result 'completed', got %v", resultMap["result"])
	}
}

func TestApplyLatestStage_WithPendingStages(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create test stages (latest first in the list)
	stages := []*models.Stage{
		{
			ID:              "latest-stage",
			SessionID:       server.session.ID,
			Operation:       "insert",
			TargetType:      "line",
			TargetName:      "newLine",
			ConfidenceLevel: "high",
			Status:          "pending",
			ExpiresAt:       time.Now().Add(1 * time.Hour),
		},
		{
			ID:        "older-stage",
			SessionID: server.session.ID,
			Status:    "pending",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		},
	}

	for _, stage := range stages {
		err := server.staging.CreateStage(stage)
		if err != nil {
			t.Fatalf("Failed to create stage %s: %v", stage.ID, err)
		}
	}

	result, err := server.applyLatestStage()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "applied" {
		t.Errorf("Expected result 'applied', got %v", resultMap["result"])
	}
}

func TestListPendingStages_WithStages(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create test stages with different expiration times
	stages := []*models.Stage{
		{
			ID:              "recent-stage",
			SessionID:       server.session.ID,
			Operation:       "replace",
			TargetType:      "function",
			TargetName:      "testFunc",
			ConfidenceLevel: "high",
			ConfidenceScore: 0.95,
			Status:          "pending",
			ExpiresAt:       time.Now().Add(1 * time.Hour),
		},
		{
			ID:              "expiring-soon",
			SessionID:       server.session.ID,
			Operation:       "delete",
			TargetType:      "line",
			TargetName:      "oldLine",
			ConfidenceLevel: "medium",
			ConfidenceScore: 0.75,
			Status:          "pending",
			ExpiresAt:       time.Now().Add(2 * time.Minute), // Expires soon
		},
		{
			ID:              "expired-stage",
			SessionID:       server.session.ID,
			Operation:       "insert",
			TargetType:      "import",
			TargetName:      "newImport",
			ConfidenceLevel: "low",
			ConfidenceScore: 0.60,
			Status:          "pending",
			ExpiresAt:       time.Now().Add(-1 * time.Hour), // Already expired
		},
	}

	for _, stage := range stages {
		err := server.staging.CreateStage(stage)
		if err != nil {
			t.Fatalf("Failed to create stage %s: %v", stage.ID, err)
		}
	}

	result, err := server.listPendingStages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	count, ok := resultMap["count"].(int)
	if !ok || count != 3 {
		t.Errorf("Expected count 3, got %v", resultMap["count"])
	}

	// Check content includes stage information
	content, ok := resultMap["content"].([]map[string]any)
	if !ok || len(content) == 0 {
		t.Error("Expected content in response")
	} else {
		text, ok := content[0]["text"].(string)
		if !ok {
			t.Error("Expected text content")
		} else {
			// Should show count, operation details, confidence, and expiration status
			if !contains(text, "3 pending stage(s)") {
				t.Error("Expected stage count in response")
			}
			if !contains(text, "replace function 'testFunc'") {
				t.Error("Expected operation details in response")
			}
			if !contains(text, "Confidence: high (0.95)") {
				t.Error("Expected confidence information in response")
			}
			if !contains(text, "❌ EXPIRED") {
				t.Error("Expected expired status in response")
			}
			if !contains(text, "⚠️") {
				t.Error("Expected warning for expiring soon stage")
			}
			if !contains(text, "apply {id}") {
				t.Error("Expected usage instructions in response")
			}
		}
	}
}

func TestHandleApplyTool_DefaultToListPendingStages(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create a test stage
	stage := &models.Stage{
		ID:        "default-stage",
		SessionID: server.session.ID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := server.staging.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	// Test with empty params (should default to listing)
	params := json.RawMessage(`{}`)

	result, err := server.handleApplyTool(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	// Should show pending stages
	count, ok := resultMap["count"].(int)
	if !ok || count != 1 {
		t.Errorf("Expected count 1, got %v", resultMap["count"])
	}
}

func TestHandleApplyTool_AllFlag(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create test stages
	stage := &models.Stage{
		ID:        "all-stage",
		SessionID: server.session.ID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := server.staging.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	// Test apply all
	params := json.RawMessage(`{"all": true}`)

	result, err := server.handleApplyTool(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "completed" {
		t.Errorf("Expected result 'completed', got %v", resultMap["result"])
	}
}

func TestHandleApplyTool_LatestFlag(t *testing.T) {
	server := setupTestServerWithDB(t)

	// Create test stage
	stage := &models.Stage{
		ID:        "latest-stage",
		SessionID: server.session.ID,
		Status:    "pending",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	err := server.staging.CreateStage(stage)
	if err != nil {
		t.Fatalf("Failed to create test stage: %v", err)
	}

	// Test apply latest
	params := json.RawMessage(`{"latest": true}`)

	result, err := server.handleApplyTool(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "applied" {
		t.Errorf("Expected result 'applied', got %v", resultMap["result"])
	}
}

func TestApplyAllStages_WithNoSession(t *testing.T) {
	server := setupTestServerWithDB(t)
	server.session = nil // No session

	result, err := server.applyAllStages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "completed" {
		t.Errorf("Expected result 'completed', got %v", resultMap["result"])
	}
}

func TestListPendingStages_WithNoSession(t *testing.T) {
	server := setupTestServerWithDB(t)
	server.session = nil // No session

	result, err := server.listPendingStages()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if resultMap["result"] != "completed" {
		t.Errorf("Expected result 'completed', got %v", resultMap["result"])
	}
}
