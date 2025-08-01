package golang

import (
	"testing"
)

// TestNew verifies provider instantiation
func TestNew(t *testing.T) {
	provider := New()
	if provider == nil {
		t.Fatal("New() returned nil provider")
	}

	// Verify it implements the interface correctly
	if provider.Lang() != "go" {
		t.Errorf("Expected Lang() to return 'go', got %q", provider.Lang())
	}
}

// TestGoProvider_Lang tests the canonical language name
func TestGoProvider_Lang(t *testing.T) {
	p := &goProvider{}
	if got := p.Lang(); got != "go" {
		t.Errorf("Lang() = %q, want %q", got, "go")
	}
}

// TestGoProvider_Aliases tests language aliases
func TestGoProvider_Aliases(t *testing.T) {
	p := &goProvider{}
	aliases := p.Aliases()
	expected := []string{"go", "golang"}

	if len(aliases) != len(expected) {
		t.Fatalf("Aliases() returned %d items, want %d", len(aliases), len(expected))
	}

	for i, alias := range aliases {
		if alias != expected[i] {
			t.Errorf("Aliases()[%d] = %q, want %q", i, alias, expected[i])
		}
	}
}

// TestGoProvider_GetDefaultIgnorePatterns tests ignore patterns for files and symbols
func TestGoProvider_GetDefaultIgnorePatterns(t *testing.T) {
	p := &goProvider{}
	files, symbols := p.GetDefaultIgnorePatterns()

	expectedFiles := []string{"*_test.go"}
	expectedSymbols := []string{"Test*", "Benchmark*", "Example*"}

	if len(files) != len(expectedFiles) {
		t.Fatalf("GetDefaultIgnorePatterns() files = %v, want %v", files, expectedFiles)
	}

	if len(symbols) != len(expectedSymbols) {
		t.Fatalf("GetDefaultIgnorePatterns() symbols = %v, want %v", symbols, expectedSymbols)
	}

	for i, file := range files {
		if file != expectedFiles[i] {
			t.Errorf("files[%d] = %q, want %q", i, file, expectedFiles[i])
		}
	}

	for i, symbol := range symbols {
		if symbol != expectedSymbols[i] {
			t.Errorf("symbols[%d] = %q, want %q", i, symbol, expectedSymbols[i])
		}
	}
}

// TestGoProvider_IsBlockLevelNode tests block-level node detection
func TestGoProvider_IsBlockLevelNode(t *testing.T) {
	p := &goProvider{}

	tests := []struct {
		nodeType string
		expected bool
	}{
		{"func", true},
		{"const", true},
		{"var", true},
		{"struct", true},
		{"field", true},
		{"call", true},
		{"assign", true},
		{"if", true},
		{"import", true},
		{"block", true},
		{"unknown", false},
		{"", false},
		{"interface", false},
	}

	for _, tt := range tests {
		t.Run("nodeType_"+tt.nodeType, func(t *testing.T) {
			result := p.IsBlockLevelNode(tt.nodeType)
			if result != tt.expected {
				t.Errorf("IsBlockLevelNode(%q) = %v, want %v", tt.nodeType, result, tt.expected)
			}
		})
	}
}

// TestGoProvider_GetSitterLanguage tests Tree-sitter language retrieval
func TestGoProvider_GetSitterLanguage(t *testing.T) {
	p := &goProvider{}
	lang := p.GetSitterLanguage()

	if lang == nil {
		t.Fatal("GetSitterLanguage() returned nil")
	}

	// Verify it's a valid Tree-sitter language by checking it's not nil
	// We can't check Version() as it's not available in this Tree-sitter binding
}

// TestGoProvider_HasNegationPredicates tests negation predicate detection
func TestGoProvider_HasNegationPredicates(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "contains_not_eq",
			query:    `(function_declaration name: (identifier) (#not-eq? @name "test")) @target`,
			expected: true,
		},
		{
			name:     "contains_not_match",
			query:    `(function_declaration name: (identifier) (#not-match? @name "^Test.*")) @target`,
			expected: true,
		},
		{
			name:     "contains_both",
			query:    `(function_declaration (#not-eq? @name "test") (#not-match? @name "^Test.*")) @target`,
			expected: true,
		},
		{
			name:     "contains_regular_predicates",
			query:    `(function_declaration name: (identifier) (#eq? @name "test")) @target`,
			expected: false,
		},
		{
			name:     "contains_match_predicate",
			query:    `(function_declaration name: (identifier) (#match? @name "^Test.*")) @target`,
			expected: false,
		},
		{
			name:     "empty_query",
			query:    "",
			expected: false,
		},
		{
			name:     "no_predicates",
			query:    `(function_declaration) @target`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasNegationPredicates(tt.query)
			if result != tt.expected {
				t.Errorf("HasNegationPredicates(%q) = %v, want %v", tt.query, result, tt.expected)
			}
		})
	}
}

// TestGoProvider_MatchesWildcard tests the exported wildcard matching method
func TestGoProvider_MatchesWildcard(t *testing.T) {
	p := &goProvider{}

	tests := []struct {
		pattern  string
		text     string
		expected bool
	}{
		{"*", "anything", true},
		{"test*", "testing", true},
		{"*test", "unittest", true},
		{"*test*", "testing", true},
		{"test*case", "testcase", true},
		{"exact", "exact", true},
		{"exact", "different", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.text, func(t *testing.T) {
			result := p.MatchesWildcard(tt.pattern, tt.text)
			if result != tt.expected {
				t.Errorf("MatchesWildcard(%q, %q) = %v, want %v", tt.pattern, tt.text, result, tt.expected)
			}
		})
	}
}
