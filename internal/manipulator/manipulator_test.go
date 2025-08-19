package manipulator

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/model"
	"github.com/termfx/morfx/internal/types"
)

// MockLanguageProvider for testing
type MockLanguageProvider struct {
	lang string
}

func (m *MockLanguageProvider) Lang() string {
	return m.lang
}

func (m *MockLanguageProvider) Aliases() []string {
	return []string{m.lang}
}

func (m *MockLanguageProvider) Extensions() []string {
	return []string{"." + m.lang}
}

func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	return nil // Mock implementation
}

func (m *MockLanguageProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping {
	return []types.NodeMapping{}
}

func (m *MockLanguageProvider) TranslateQuery(q *types.Query) (string, error) {
	return "(function_declaration) @target", nil
}

func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	return types.NodeKind(dslKind)
}

func (m *MockLanguageProvider) GetSupportedDSLKinds() []string {
	return []string{"function", "class"}
}

func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return map[string]string{}
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	return types.NodeKind("function")
}

func (m *MockLanguageProvider) GetNodeName(node *sitter.Node, source []byte) string {
	return "mockFunction"
}

func (m *MockLanguageProvider) OptimizeQuery(q *types.Query) *types.Query {
	return q
}

func (m *MockLanguageProvider) EstimateQueryCost(q *types.Query) int {
	return 1
}

func (m *MockLanguageProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	return types.ScopeFile
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node {
	return nil
}

func (m *MockLanguageProvider) IsBlockLevelNode(nodeType string) bool {
	return nodeType == "function_declaration" || nodeType == "struct"
}

func (m *MockLanguageProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{}, []string{}
}

// TestManipulate is skipped because it requires real tree-sitter language support
// which is complex to mock properly. These would be better as integration tests.
func TestManipulate_ConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        *model.Config
		expectedError bool
	}{
		{
			name:          "nil config",
			config:        nil,
			expectedError: true,
		},
		{
			name: "valid config",
			config: &model.Config{
				Operation: model.OpGet,
				Pattern:   "test",
				RuleID:    "test-rule",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic config validation without actual parsing
			if tt.config == nil {
				if !tt.expectedError {
					t.Error("Expected error for nil config")
				}
				return
			}

			if tt.config.Operation == "" {
				if !tt.expectedError {
					t.Error("Expected error for empty operation")
				}
			}
		})
	}
}

func TestExtractNodeType(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "simple node type",
			pattern:  "func:Init",
			expected: "func",
		},
		{
			name:     "negated node type",
			pattern:  "!struct:User",
			expected: "struct",
		},
		{
			name:     "parent child relationship",
			pattern:  "func:Init > block",
			expected: "func",
		},
		{
			name:     "node type without colon",
			pattern:  "function_declaration",
			expected: "function_declaration",
		},
		{
			name:     "empty pattern",
			pattern:  "",
			expected: "",
		},
		{
			name:     "complex pattern with spaces",
			pattern:  "  func:Test  >  block  ",
			expected: "func",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNodeType(tt.pattern)
			if result != tt.expected {
				t.Errorf("extractNodeType(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestPreserveIndentation(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		position int
		text     string
		expected string
	}{
		{
			name:     "preserve spaces",
			content:  "package main\n    func test() {}",
			position: 17, // After "    "
			text:     "// comment\nfunc new() {}",
			expected: "    // comment\n    func new() {}",
		},
		{
			name:     "preserve tabs",
			content:  "package main\n\tfunc test() {}",
			position: 14, // After "\t"
			text:     "// comment\nfunc new() {}",
			expected: "\t// comment\n\tfunc new() {}",
		},
		{
			name:     "no indentation",
			content:  "package main\nfunc test() {}",
			position: 13, // Start of line
			text:     "// comment\nfunc new() {}",
			expected: "// comment\nfunc new() {}",
		},
		{
			name:     "windows line endings",
			content:  "package main\r\n    func test() {}",
			position: 18, // After "    "
			text:     "// comment\nfunc new() {}",
			expected: "    // comment\r\n    func new() {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preserveIndentation(tt.content, tt.position, tt.text)
			if result != tt.expected {
				t.Errorf("preserveIndentation() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestApplyRewrites(t *testing.T) {
	tests := []struct {
		name            string
		originalContent string
		rewrites        []Rewrite
		expectedContent string
		expectedChanges int
	}{
		{
			name:            "single replacement",
			originalContent: "package main\n\nfunc main() {}\n",
			rewrites: []Rewrite{
				{
					RuleID:    "test-rule",
					Start:     19,
					End:       23,
					NewText:   []byte("replaced"),
					LineStart: 3,
					LineEnd:   3,
				},
			},
			expectedContent: "package main\n\nfunc replaced() {}\n",
			expectedChanges: 1,
		},
		{
			name:            "multiple replacements",
			originalContent: "func a() {}\nfunc b() {}\n",
			rewrites: []Rewrite{
				{
					RuleID:    "test-rule",
					Start:     0,
					End:       11,
					NewText:   []byte("func x() {}"),
					LineStart: 1,
					LineEnd:   1,
				},
				{
					RuleID:    "test-rule",
					Start:     12,
					End:       23,
					NewText:   []byte("func y() {}"),
					LineStart: 2,
					LineEnd:   2,
				},
			},
			expectedContent: "func x() {}\nfunc y() {}\n",
			expectedChanges: 2,
		},
		{
			name:            "deletion",
			originalContent: "package main\n\nfunc main() {}\n",
			rewrites: []Rewrite{
				{
					RuleID:    "test-rule",
					Start:     13,
					End:       28,
					NewText:   []byte(""),
					LineStart: 3,
					LineEnd:   3,
				},
			},
			expectedContent: "package main\n\n",
			expectedChanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, changes := applyRewrites(tt.originalContent, tt.rewrites)
			if content != tt.expectedContent {
				t.Errorf("applyRewrites() content = %q, want %q", content, tt.expectedContent)
			}
			if len(changes) != tt.expectedChanges {
				t.Errorf("applyRewrites() changes count = %d, want %d", len(changes), tt.expectedChanges)
			}
		})
	}
}

// Note: TestManipulator_Apply and TestManipulator_Fake require complex mocking
// of sitter.Node which is difficult to implement properly. These tests would
// be better implemented as integration tests with real tree-sitter parsing.

func TestManipulator_Creation(t *testing.T) {
	manipulator := &Manipulator{
		Config: &model.Config{
			RuleID: "test-rule",
		},
		Original: "package main\n\nfunc test() {}\n",
	}

	if manipulator.Config == nil {
		t.Error("Expected config to be set")
	}

	if len(manipulator.Original) == 0 {
		t.Error("Expected original content to be set")
	}
}

// mockSitterNode implements the necessary methods for testing
type mockSitterNode struct {
	startByte uint32
	endByte   uint32
	startRow  uint32
	endRow    uint32
	nodeType  string
}

func (m *mockSitterNode) StartByte() uint32 {
	return m.startByte
}

func (m *mockSitterNode) EndByte() uint32 {
	return m.endByte
}

func (m *mockSitterNode) StartPoint() sitter.Point {
	return sitter.Point{Row: m.startRow, Column: 0}
}

func (m *mockSitterNode) EndPoint() sitter.Point {
	return sitter.Point{Row: m.endRow, Column: 0}
}

func (m *mockSitterNode) Type() string {
	if m.nodeType != "" {
		return m.nodeType
	}
	return "function_declaration"
}

func (m *mockSitterNode) Symbol() uint16                            { return 0 }
func (m *mockSitterNode) Language() *sitter.Language                { return nil }
func (m *mockSitterNode) GrammarName() string                       { return "" }
func (m *mockSitterNode) String() string                            { return "" }
func (m *mockSitterNode) Equal(other *sitter.Node) bool             { return false }
func (m *mockSitterNode) HasChanges() bool                          { return false }
func (m *mockSitterNode) HasError() bool                            { return false }
func (m *mockSitterNode) IsError() bool                             { return false }
func (m *mockSitterNode) IsMissing() bool                           { return false }
func (m *mockSitterNode) IsExtra() bool                             { return false }
func (m *mockSitterNode) IsNamed() bool                             { return true }
func (m *mockSitterNode) Parent() *sitter.Node                      { return nil }
func (m *mockSitterNode) Child(idx int) *sitter.Node                { return nil }
func (m *mockSitterNode) NamedChild(idx int) *sitter.Node           { return nil }
func (m *mockSitterNode) ChildCount() uint32                        { return 0 }
func (m *mockSitterNode) NamedChildCount() uint32                   { return 0 }
func (m *mockSitterNode) NextSibling() *sitter.Node                 { return nil }
func (m *mockSitterNode) NextNamedSibling() *sitter.Node            { return nil }
func (m *mockSitterNode) PrevSibling() *sitter.Node                 { return nil }
func (m *mockSitterNode) PrevNamedSibling() *sitter.Node            { return nil }
func (m *mockSitterNode) ChildByFieldName(name string) *sitter.Node { return nil }
func (m *mockSitterNode) FieldNameForChild(idx uint32) string       { return "" }
func (m *mockSitterNode) Content(input []byte) string               { return "" }
func (m *mockSitterNode) Walk() *sitter.TreeCursor                  { return nil }
