package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/mcp/types"
	"github.com/termfx/morfx/models"
	"github.com/termfx/morfx/providers"
)

// mockRegistry adapts providers.Registry to core.ProviderRegistry
type mockRegistry struct {
	providers map[string]core.Provider
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		providers: make(map[string]core.Provider),
	}
}

func (r *mockRegistry) Register(p core.Provider) {
	r.providers[p.Language()] = p
}

func (r *mockRegistry) Get(language string) (core.Provider, bool) {
	p, exists := r.providers[language]
	return p, exists
}

// mockServer implements types.ServerInterface for testing
type mockServer struct {
	providerRegistry *providers.Registry
	coreRegistry     *mockRegistry
	fileProcessor    *core.FileProcessor
	staging          any
	safety           any
	sessionID        string
	samplingRequests []map[string]any
	samplingResults  []map[string]any
	samplingErr      error
}

func newMockServer() *mockServer {
	// Create provider registry for MCP
	providerRegistry := providers.NewRegistry()

	// Create core registry for FileProcessor
	coreRegistry := newMockRegistry()

	// Register mock providers in both registries
	providers := []string{"go", "javascript", "typescript", "php"}
	for _, lang := range providers {
		mock := &mockProvider{language: lang}
		providerRegistry.Register(mock)
		coreRegistry.Register(mock)
	}

	// Create file processor using the core registry
	fileProc := core.NewFileProcessor(coreRegistry)

	return &mockServer{
		providerRegistry: providerRegistry,
		coreRegistry:     coreRegistry,
		fileProcessor:    fileProc,
		staging:          &mockStaging{stages: make(map[string]any)},
		safety:           &mockSafety{},
		sessionID:        "mock-session",
	}
}

func (m *mockServer) GetProviders() *providers.Registry {
	return m.providerRegistry
}

func (m *mockServer) GetFileProcessor() *core.FileProcessor {
	return m.fileProcessor
}

func (m *mockServer) GetStaging() any {
	return m.staging
}

func (m *mockServer) GetSafety() any {
	return m.safety
}

func (m *mockServer) GetSessionID() string {
	return m.sessionID
}

func (m *mockServer) ReportProgress(ctx context.Context, progress, total float64, message string) {}

func (m *mockServer) ConfirmApply(ctx context.Context, summary string) error {
	return nil
}

func (m *mockServer) RequestSampling(ctx context.Context, params map[string]any) (map[string]any, error) {
	m.samplingRequests = append(m.samplingRequests, params)
	if m.samplingErr != nil {
		return nil, m.samplingErr
	}
	if len(m.samplingResults) > 0 {
		result := m.samplingResults[0]
		m.samplingResults = m.samplingResults[1:]
		return result, nil
	}
	return nil, nil
}

func (m *mockServer) RequestElicitation(ctx context.Context, params map[string]any) (map[string]any, error) {
	return nil, nil
}

func (m *mockServer) FinalizeTransform(ctx context.Context, req types.TransformRequest) (map[string]any, error) {
	content := req.ResponseText
	if content == "" {
		content = "operation completed"
	}

	blocks := []map[string]any{{
		"type": "text",
		"text": content,
	}}

	resp := map[string]any{
		"content":    blocks,
		"confidence": req.Result.Confidence.Score,
		"matches":    req.Result.MatchCount,
	}

	if req.Path != "" {
		resp["path"] = req.Path
	}

	if staging, ok := m.staging.(*mockStaging); ok && staging.enabled {
		id := fmt.Sprintf("stg_%d", len(staging.stages)+1)
		staging.AddStage(id, req)
		resp["id"] = id
		resp["result"] = "staged"
	} else {
		resp["result"] = "completed"
	}

	if req.Result.Modified != "" {
		resp["modified"] = req.Result.Modified
	}

	return resp, nil
}

// mockStaging implements a simple staging manager for tests
type mockStaging struct {
	enabled bool
	stages  map[string]any
}

func (m *mockStaging) IsEnabled() bool {
	return m.enabled
}

func (m *mockStaging) AddStage(id string, stage any) {
	m.stages[id] = stage
}

func (m *mockStaging) GetStageAny(id string) (any, bool) {
	stage, exists := m.stages[id]
	return stage, exists
}

func (m *mockStaging) GetAllStages() []any {
	stages := make([]any, 0, len(m.stages))
	for _, stage := range m.stages {
		stages = append(stages, stage)
	}
	return stages
}

func (m *mockStaging) ClearStages() {
	m.stages = make(map[string]any)
}

func (m *mockStaging) DeleteAppliedStages(sessionID string) error {
	// In mock, we already delete stages in ApplyStage, so this is a no-op
	return nil
}

func (m *mockStaging) DeleteStage(stageID string) error {
	delete(m.stages, stageID)
	return nil
}

func (m *mockStaging) ListPendingStages(sessionID string) ([]models.Stage, error) {
	// For testing, we'll create simple stages
	var stages []models.Stage
	for id := range m.stages {
		stages = append(stages, models.Stage{
			ID:        id,
			Status:    "pending",
			SessionID: sessionID,
		})
	}
	return stages, nil
}

func (m *mockStaging) GetStage(stageID string) (*models.Stage, error) {
	if _, exists := m.stages[stageID]; exists {
		return &models.Stage{
			ID:        stageID,
			Status:    "pending",
			SessionID: "mock-session",
		}, nil
	}
	return nil, fmt.Errorf("stage not found")
}

func (m *mockStaging) ApplyStage(ctx context.Context, stageID string, autoApplied bool) (*models.Apply, error) {
	if _, exists := m.stages[stageID]; !exists {
		return nil, fmt.Errorf("stage not found")
	}
	delete(m.stages, stageID)
	return &models.Apply{
		ID:          "apply-" + stageID,
		StageID:     stageID,
		AutoApplied: autoApplied,
		AppliedBy:   "test",
	}, nil
}

// mockSafety implements a simple safety manager for tests
type mockSafety struct{}

func (m *mockSafety) ValidatePath(path string) error {
	// Allow all paths in tests
	return nil
}

// mockProvider implements both providers.Provider and core.Provider for testing
type mockProvider struct {
	language string
}

func (m *mockProvider) Language() string {
	return m.language
}

func (m *mockProvider) Extensions() []string {
	switch m.language {
	case "go":
		return []string{".go"}
	case "javascript":
		return []string{".js", ".jsx"}
	case "typescript":
		return []string{".ts", ".tsx"}
	case "php":
		return []string{".php"}
	default:
		return []string{}
	}
}

// Query implements core.Provider
func (m *mockProvider) Query(source string, query core.AgentQuery) core.QueryResult {
	return core.QueryResult{
		Matches: []core.Match{
			{
				Type: query.Type,
				Name: query.Name,
				Location: core.Location{
					Line:    1,
					EndLine: 2,
				},
				Content: "// Found: " + query.Name,
			},
		},
		Total: 1,
	}
}

// Transform implements core.Provider
func (m *mockProvider) Transform(source string, op core.TransformOp) core.TransformResult {
	var modified string

	switch op.Method {
	case "replace":
		modified = op.Replacement
	case "delete":
		modified = ""
	case "insert_before":
		modified = op.Content + "\n" + source
	case "insert_after":
		modified = source + "\n" + op.Content
	case "append":
		modified = source + "\n" + op.Content
	default:
		modified = source
	}

	return core.TransformResult{
		Modified: modified,
		Diff:     "Mock diff",
		Confidence: core.ConfidenceScore{
			Score: 0.9,
			Level: "high",
		},
		MatchCount: 1,
	}
}

// Validate implements providers.Provider
func (m *mockProvider) Validate(source string) providers.ValidationResult {
	return providers.ValidationResult{
		Valid:  true,
		Errors: []string{},
	}
}

func (m *mockProvider) SupportedQueryTypes() []string {
	return []string{"function", "variable", "class"}
}

func (m *mockProvider) Stats() providers.Stats {
	return providers.Stats{}
}

// Helper functions for tests

func createTestParams(params map[string]any) json.RawMessage {
	data, _ := json.Marshal(params)
	return json.RawMessage(data)
}

func createTestFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func assertError(t *testing.T, err error, contains string) {
	t.Helper()
	if err == nil {
		t.Error("Expected error but got none")
		return
	}
	if contains != "" && !strings.Contains(err.Error(), contains) {
		t.Errorf("Expected error containing '%s', got: %v", contains, err)
	}
}

// Additional test helpers for staging operations
func setStaging(server *mockServer, enabled bool) {
	if staging, ok := server.staging.(*mockStaging); ok {
		staging.enabled = enabled
	}
}

func addTestStage(server *mockServer, id string, stage any) {
	if staging, ok := server.staging.(*mockStaging); ok {
		staging.AddStage(id, stage)
	}
}

func clearStages(server *mockServer) {
	if staging, ok := server.staging.(*mockStaging); ok {
		staging.ClearStages()
	}
}

func getStageCount(server *mockServer) int {
	if staging, ok := server.staging.(*mockStaging); ok {
		return len(staging.stages)
	}
	return 0
}
