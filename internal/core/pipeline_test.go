package core

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider implements LanguageProvider for testing
type MockProvider struct {
	name string
}

func (m *MockProvider) Lang() string                          { return m.name }
func (m *MockProvider) Aliases() []string                     { return []string{"mock"} }
func (m *MockProvider) Extensions() []string                  { return []string{".mock"} }
func (m *MockProvider) GetSitterLanguage() any                { return nil }
func (m *MockProvider) NormalizeDSLKind(kind string) NodeKind { return NodeKind(kind) }
func (m *MockProvider) GetSupportedDSLKinds() []string        { return []string{"function", "class"} }
func (m *MockProvider) TranslateQuery(query *Query) (string, error) {
	// For overlap testing, return a query that would match overlapping regions
	if query.Pattern == "overlapping" {
		return "(identifier) @overlap", nil
	}
	return "(function_declaration) @func", nil
}

func (m *MockProvider) TranslateKind(kind NodeKind) []NodeMapping {
	return []NodeMapping{}
}

func (m *MockProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return map[string]string{}
}

func (m *MockProvider) OptimizeQuery(q *Query) *Query {
	return q
}

func (m *MockProvider) GetNodeScope(node *sitter.Node) ScopeType {
	return ScopeFile
}

func (m *MockProvider) FindEnclosingScope(node *sitter.Node, scope ScopeType) *sitter.Node {
	return node
}
func (m *MockProvider) GetNodeKind(node *sitter.Node) NodeKind              { return KindFunction }
func (m *MockProvider) GetNodeName(node *sitter.Node, source []byte) string { return "test_node" }
func (m *MockProvider) IsBlockLevelNode(nodeType string) bool               { return true }
func (m *MockProvider) GetDefaultIgnorePatterns() ([]string, []string) {
	return []string{"*.test.go", "vendor/*"}, []string{"test_*"}
}

func (m *MockProvider) AppendPoint(node any, source []byte) (int, error) {
	if n, ok := node.(*sitter.Node); ok {
		return int(n.EndByte()), nil
	}
	return 0, nil
}

func (m *MockProvider) ValidateSnippet(snippet string, context any, source []byte) error {
	return nil
}

func (m *MockProvider) OrganizeImports(source []byte) ([]byte, error) {
	return source, nil
}

func (m *MockProvider) Format(source []byte) ([]byte, error) {
	return source, nil
}

func (m *MockProvider) QuickCheck(source []byte) []QuickCheckDiagnostic {
	return []QuickCheckDiagnostic{}
}

func (m *MockProvider) EstimateQueryCost(q *Query) int {
	return 1
}

func TestNewPipeline(t *testing.T) {
	provider := &MockProvider{name: "test"}
	pipeline := NewPipeline(provider)

	assert.NotNil(t, pipeline)
	assert.Equal(t, provider, pipeline.provider)
}

func TestPipelineApply_HappyPath(t *testing.T) {
	provider := &MockProvider{name: "test"}
	pipeline := NewPipeline(provider)

	input := Input{
		Language: "test",
		CodeIn:   "function test() {}",
		Query:    "function",
		Op:       Replace,
		Repl:     "function newTest() {}",
		Options: &InputOptions{
			DryRun:      false,
			Interactive: false,
			Fuzz:        false,
		},
	}

	result, err := pipeline.Apply(input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, StatusSuccess, result.Status)
	assert.NotEmpty(t, result.Hash)
	assert.NotZero(t, result.Stats.Duration)
}

func TestPipelineApply_DryRun(t *testing.T) {
	provider := &MockProvider{name: "test"}
	pipeline := NewPipeline(provider)

	input := Input{
		Language: "test",
		CodeIn:   "function test() {}",
		Query:    "function",
		Op:       Replace,
		Repl:     "function newTest() {}",
		Options: &InputOptions{
			DryRun: true,
			Fuzz:   false,
		},
	}

	result, err := pipeline.Apply(input)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, StatusSuccess, result.Status)
}

func TestPipelineApply_OverlapDetection(t *testing.T) {
	// Test the detectOverlaps method directly since creating overlapping edits
	// through the normal pipeline flow is complex without real tree-sitter parsing
	provider := &MockProvider{}
	pipeline := NewPipeline(provider)

	// Create overlapping edits directly
	overlappingEdits := []Edit{
		{
			StartByte: 0,
			EndByte:   15,
			NewText:   "first edit",
			Operation: Replace,
			Priority:  0,
		},
		{
			StartByte: 10, // Overlaps with first edit (10 < 15)
			EndByte:   25,
			NewText:   "second edit",
			Operation: Replace,
			Priority:  1,
		},
	}

	// Test overlap detection
	overlaps := pipeline.detectOverlaps(overlappingEdits)
	if len(overlaps) == 0 {
		t.Error("Expected overlaps to be detected")
	}
	if len(overlaps) != 1 {
		t.Errorf("Expected 1 overlap, got %d", len(overlaps))
	}

	// Test non-overlapping edits
	nonOverlappingEdits := []Edit{
		{
			StartByte: 0,
			EndByte:   10,
			NewText:   "first edit",
			Operation: Replace,
			Priority:  0,
		},
		{
			StartByte: 20, // No overlap (20 > 10)
			EndByte:   30,
			NewText:   "second edit",
			Operation: Replace,
			Priority:  1,
		},
	}

	noOverlaps := pipeline.detectOverlaps(nonOverlappingEdits)
	if len(noOverlaps) != 0 {
		t.Errorf("Expected no overlaps, got %d", len(noOverlaps))
	}
}

func BenchmarkPipelineApply(b *testing.B) {
	provider := &MockProvider{}
	pipeline := NewPipeline(provider)

	input := Input{
		Language: "go",
		CodeIn:   "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
		Query:    "function",
		Op:       "replace",
		Repl:     "func test() { println(\"test\") }",
		Options:  &InputOptions{DryRun: false},
	}

	for b.Loop() {
		_, _ = pipeline.Apply(input)
	}
}
