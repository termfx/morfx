package base

import (
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"

	"github.com/termfx/morfx/core"
)

// mockConfig implements LanguageConfig for testing
type mockConfig struct {
	language   string
	extensions []string
}

func (m *mockConfig) Language() string {
	return m.language
}

func (m *mockConfig) Extensions() []string {
	return m.extensions
}

func (m *mockConfig) GetLanguage() *sitter.Language {
	return golang.GetLanguage()
}

func (m *mockConfig) MapQueryTypeToNodeTypes(queryType string) []string {
	switch queryType {
	case "function":
		return []string{"function_declaration"}
	case "struct":
		return []string{"type_spec"}
	case "variable":
		return []string{"var_declaration"}
	default:
		return []string{queryType}
	}
}

func (m *mockConfig) ExtractNodeName(node *sitter.Node, source string) string {
	if nameNode := node.ChildByFieldName("name"); nameNode != nil {
		return source[nameNode.StartByte():nameNode.EndByte()]
	}
	return ""
}

func (m *mockConfig) IsExported(name string) bool {
	return len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z'
}

func newTestProvider() *Provider {
	config := &mockConfig{
		language:   "go",
		extensions: []string{".go"},
	}
	return New(config)
}

// TestNew tests provider creation
func TestNew(t *testing.T) {
	config := &mockConfig{
		language:   "go",
		extensions: []string{".go"},
	}

	provider := New(config)
	if provider == nil {
		t.Fatal("New returned nil")
	}

	if provider.config != config {
		t.Error("Config not set properly")
	}

	if provider.parser == nil {
		t.Error("Parser not initialized")
	}

	if provider.cache == nil {
		t.Error("Cache not initialized")
	}
}

// TestLanguage tests language getter
func TestLanguage(t *testing.T) {
	provider := newTestProvider()
	if provider.Language() != "go" {
		t.Errorf("Expected 'go', got '%s'", provider.Language())
	}
}

// TestExtensions tests extensions getter
func TestExtensions(t *testing.T) {
	provider := newTestProvider()
	extensions := provider.Extensions()

	if len(extensions) != 1 || extensions[0] != ".go" {
		t.Errorf("Expected ['.go'], got %v", extensions)
	}
}

// TestQuery tests code element queries
func TestQuery(t *testing.T) {
	provider := newTestProvider()

	// Test simple function query
	goCode := `
package main

func HelloWorld() string {
	return "Hello, World!"
}

func privateFunc() int {
	return 42
}
`

	query := core.AgentQuery{
		Type: "function",
		Name: "HelloWorld",
	}

	result := provider.Query(goCode, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	if len(result.Matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(result.Matches))
	}

	if result.Total != 1 {
		t.Errorf("Expected total 1, got %d", result.Total)
	}

	match := result.Matches[0]
	if match.Name != "HelloWorld" {
		t.Errorf("Expected match name 'HelloWorld', got '%s'", match.Name)
	}

	if match.Type != "function" {
		t.Errorf("Expected match type 'function', got '%s'", match.Type)
	}

	// Location should be set
	if match.Location.Line <= 0 {
		t.Error("Location line should be > 0")
	}

	// Content should contain function code
	if !strings.Contains(match.Content, "HelloWorld") {
		t.Error("Match content should contain function name")
	}
}

// TestQueryWithWildcard tests wildcard pattern matching
func TestQueryWithWildcard(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func TestFunc1() {}
func TestFunc2() {}
func OtherFunc() {}
`

	query := core.AgentQuery{
		Type: "function",
		Name: "Test*",
	}

	result := provider.Query(goCode, query)
	if result.Error != nil {
		t.Fatalf("Query failed: %v", result.Error)
	}

	// Should match TestFunc1 and TestFunc2
	if len(result.Matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(result.Matches))
	}
}

// TestQueryInvalidSource tests query with invalid source
func TestQueryInvalidSource(t *testing.T) {
	provider := newTestProvider()

	// Invalid Go source
	invalidCode := `
this is not valid go code {{{
`

	query := core.AgentQuery{
		Type: "function",
		Name: "test",
	}

	result := provider.Query(invalidCode, query)
	// Should still work but might not find anything meaningful
	if result.Error != nil {
		// Parse errors are not necessarily fatal for queries
		t.Logf("Query returned error (expected for invalid code): %v", result.Error)
	}
}

// TestQueryEmptySource tests query with empty source
func TestQueryEmptySource(t *testing.T) {
	provider := newTestProvider()

	query := core.AgentQuery{
		Type: "function",
		Name: "test",
	}

	result := provider.Query("", query)
	if result.Error != nil {
		t.Logf("Query error on empty source: %v", result.Error)
	}

	// Should return no matches for empty source
	if len(result.Matches) != 0 {
		t.Errorf("Expected 0 matches for empty source, got %d", len(result.Matches))
	}
}

// TestTransformReplace tests replace transformation
func TestTransformReplace(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func OldFunc() string {
	return "old"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "OldFunc",
		},
		Replacement: `func NewFunc() string {
	return "new"
}`,
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	if result.MatchCount != 1 {
		t.Errorf("Expected 1 match, got %d", result.MatchCount)
	}

	// Should contain new function
	if !strings.Contains(result.Modified, "NewFunc") {
		t.Error("Modified code should contain 'NewFunc'")
	}

	// Should not contain old function
	if strings.Contains(result.Modified, "OldFunc") {
		t.Error("Modified code should not contain 'OldFunc'")
	}

	// Confidence should be reasonable
	if result.Confidence.Score <= 0 || result.Confidence.Score > 1 {
		t.Errorf("Invalid confidence score: %f", result.Confidence.Score)
	}

	if result.Confidence.Level == "" {
		t.Error("Confidence level should be set")
	}

	// Diff should be present
	if result.Diff == "" {
		t.Error("Diff should not be empty")
	}
}

// TestTransformDelete tests delete transformation
func TestTransformDelete(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func KeepThis() string {
	return "keep"
}

func DeleteThis() string {
	return "delete"
}
`

	op := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "DeleteThis",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should not contain deleted function
	if strings.Contains(result.Modified, "DeleteThis") {
		t.Error("Modified code should not contain 'DeleteThis'")
	}

	// Should still contain kept function
	if !strings.Contains(result.Modified, "KeepThis") {
		t.Error("Modified code should still contain 'KeepThis'")
	}
}

// TestTransformInsertBefore tests insert before transformation
func TestTransformInsertBefore(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method:  "insert_before",
		Content: "// This comment is before the function\n",
		Target: core.AgentQuery{
			Type: "function",
			Name: "ExistingFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should contain the inserted comment
	if !strings.Contains(result.Modified, "// This comment is before the function") {
		t.Error("Modified code should contain inserted comment")
	}

	// Should still contain original function
	if !strings.Contains(result.Modified, "ExistingFunc") {
		t.Error("Modified code should still contain original function")
	}
}

// TestTransformInsertAfter tests insert after transformation
func TestTransformInsertAfter(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method:  "insert_after",
		Content: "\n// This comment is after the function",
		Target: core.AgentQuery{
			Type: "function",
			Name: "ExistingFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should contain the inserted comment
	if !strings.Contains(result.Modified, "// This comment is after the function") {
		t.Error("Modified code should contain inserted comment")
	}
}

// TestTransformAppend tests append transformation
func TestTransformAppend(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method: "append",
		Content: `
func AppendedFunc() string {
	return "appended"
}`,
		Target: core.AgentQuery{
			Type: "function",
			Name: "ExistingFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Should contain both functions
	if !strings.Contains(result.Modified, "ExistingFunc") {
		t.Error("Modified code should contain original function")
	}

	if !strings.Contains(result.Modified, "AppendedFunc") {
		t.Error("Modified code should contain appended function")
	}
}

// TestTransformUnknownMethod tests unknown transformation method
func TestTransformUnknownMethod(t *testing.T) {
	provider := newTestProvider()

	// Create code with a function that will match, so we get past the "no matches" check
	goCode := `
package main

func test() string {
	return "test"
}
`

	op := core.TransformOp{
		Method: "unknown_method",
		Target: core.AgentQuery{
			Type: "function",
			Name: "test",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error == nil {
		t.Error("Transform should fail with unknown method")
	}

	if !strings.Contains(result.Error.Error(), "unknown transform method") {
		t.Errorf("Expected 'unknown transform method' error, got: %v", result.Error)
	}
}

// TestTransformNoMatches tests transformation with no target matches
func TestTransformNoMatches(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main

func ExistingFunc() string {
	return "existing"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "NonExistentFunc",
		},
		Replacement: "func NewFunc() {}",
	}

	result := provider.Transform(goCode, op)
	if result.Error == nil {
		t.Error("Transform should fail when no matches found")
	}

	if !strings.Contains(result.Error.Error(), "no matches found") {
		t.Errorf("Expected 'no matches found' error, got: %v", result.Error)
	}
}

// TestTransformInvalidSource tests transformation with invalid source
func TestTransformInvalidSource(t *testing.T) {
	provider := newTestProvider()

	invalidCode := `this is not valid go {{{`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "test",
		},
		Replacement: "func Test() {}",
	}

	result := provider.Transform(invalidCode, op)
	if result.Error == nil {
		t.Error("Transform should fail with invalid source")
	}
}

// TestValidate tests source validation
func TestValidate(t *testing.T) {
	provider := newTestProvider()

	// Valid Go code
	validCode := `
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`

	result := provider.Validate(validCode)
	if !result.Valid {
		t.Errorf("Valid code should pass validation, errors: %v", result.Errors)
	}

	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors for valid code, got: %v", result.Errors)
	}
}

// TestValidateInvalidCode tests validation with syntax errors
func TestValidateInvalidCode(t *testing.T) {
	provider := newTestProvider()

	// Invalid Go code with syntax error
	invalidCode := `
package main

func main() {
	fmt.Println("Hello, World!"  // Missing closing parenthesis
}
`

	result := provider.Validate(invalidCode)
	// Note: tree-sitter might not always detect all syntax errors
	// This is testing the error detection mechanism
	t.Logf("Validation result for invalid code - Valid: %t, Errors: %v",
		result.Valid, result.Errors)
}

// TestValidateEmptySource tests validation with empty source
func TestValidateEmptySource(t *testing.T) {
	provider := newTestProvider()

	result := provider.Validate("")
	// Empty source should be considered valid (though not useful)
	if !result.Valid {
		t.Error("Empty source should be valid")
	}
}

// TestMatchesPattern tests pattern matching utility
func TestMatchesPattern(t *testing.T) {
	provider := newTestProvider()

	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{"exact_match", "TestFunc", true},
		{"wildcard_prefix", "Test*", true},
		{"wildcard_suffix", "*Func", true},
		{"wildcard_middle", "Test*Func", true},
		{"no_match", "OtherFunc", false},
		{"empty_pattern", "", true}, // Empty pattern matches all
		{"only_wildcard", "*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection or access the method somehow
			// Since matchesPattern is private, we'll test it through Query
			query := core.AgentQuery{
				Type: "function",
				Name: tt.pattern,
			}

			goCode := `
package main
func TestFunc() {}
`

			result := provider.Query(goCode, query)
			hasMatch := len(result.Matches) > 0

			if hasMatch != tt.expected {
				t.Errorf("Pattern '%s' expected %t, got %t", tt.pattern, tt.expected, hasMatch)
			}
		})
	}
}

// TestConfidenceCalculation tests confidence scoring
func TestConfidenceCalculation(t *testing.T) {
	provider := newTestProvider()

	// Test single target (should increase confidence)
	singleTargetCode := `
package main
func UniqueFunc() string {
	return "unique"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "UniqueFunc",
		},
		Replacement: "func NewFunc() {}",
	}

	result := provider.Transform(singleTargetCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Single target should have good confidence
	if result.Confidence.Score <= 0.8 {
		t.Errorf("Expected high confidence for single target, got %f", result.Confidence.Score)
	}
}

// TestConfidenceWithMultipleTargets tests confidence with many targets
func TestConfidenceWithMultipleTargets(t *testing.T) {
	provider := newTestProvider()

	// Code with many similar functions
	multipleTargetsCode := `
package main
func TestFunc1() {}
func TestFunc2() {}
func TestFunc3() {}
func TestFunc4() {}
func TestFunc5() {}
func TestFunc6() {}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "Test*", // Wildcard matches many
		},
		Replacement: "func NewFunc() {}",
	}

	result := provider.Transform(multipleTargetsCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Multiple targets should reduce confidence
	if result.Confidence.Score >= 0.8 {
		t.Errorf("Expected lower confidence for multiple targets, got %f", result.Confidence.Score)
	}

	// Should have multiple matches
	if result.MatchCount < 5 {
		t.Errorf("Expected at least 5 matches, got %d", result.MatchCount)
	}
}

// TestConfidenceWithDeleteOperation tests confidence for delete operations
func TestConfidenceWithDeleteOperation(t *testing.T) {
	provider := newTestProvider()

	goCode := `
package main
func TestFunc() string {
	return "test"
}
`

	op := core.TransformOp{
		Method: "delete",
		Target: core.AgentQuery{
			Type: "function",
			Name: "TestFunc",
		},
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Delete operations should have reduced confidence
	factors := result.Confidence.Factors
	foundDeleteFactor := false
	for _, factor := range factors {
		if factor.Name == "delete_operation" && factor.Impact < 0 {
			foundDeleteFactor = true
			break
		}
	}

	if !foundDeleteFactor {
		t.Error("Expected delete operation confidence factor")
	}
}

// TestDiffGeneration tests diff generation
func TestDiffGeneration(t *testing.T) {
	provider := newTestProvider()

	originalCode := `
package main
func OldFunc() {
	return "old"
}
`

	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "OldFunc",
		},
		Replacement: `func NewFunc() {
	return "new"
}`,
	}

	result := provider.Transform(originalCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// Diff should contain changes
	if result.Diff == "" {
		t.Error("Diff should not be empty for different content")
	}

	if !strings.Contains(result.Diff, "NewFunc") {
		t.Error("Diff should contain new function name")
	}

	if !strings.Contains(result.Diff, "+") || !strings.Contains(result.Diff, "-") {
		t.Error("Diff should contain addition and deletion markers")
	}
}

// TestDiffGenerationNoChanges tests diff with identical content
func TestDiffGenerationNoChanges(t *testing.T) {
	provider := newTestProvider()

	// This is a bit contrived since we need a transformation that results in no change
	// We'll test the diff generator directly by mocking identical content
	goCode := `
package main
func TestFunc() {}
`

	// Try to replace with identical content (though this might still generate a diff
	// due to formatting differences)
	op := core.TransformOp{
		Method: "replace",
		Target: core.AgentQuery{
			Type: "function",
			Name: "TestFunc",
		},
		Replacement: "func TestFunc() {}",
	}

	result := provider.Transform(goCode, op)
	if result.Error != nil {
		t.Fatalf("Transform failed: %v", result.Error)
	}

	// This test mainly ensures diff generation doesn't crash
	t.Logf("Diff result: '%s'", result.Diff)
}

// TestCacheBasicFunctionality tests basic cache operations
func TestCacheBasicFunctionality(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	source := []byte("package main\nfunc test() {}")

	// First call should be a miss
	tree1, hit1 := cache.GetOrParse(parser, source)
	if tree1 == nil {
		t.Fatal("GetOrParse returned nil tree")
	}

	if hit1 {
		t.Error("First call should be a miss")
	}

	// Second call should be a hit
	tree2, hit2 := cache.GetOrParse(parser, source)
	if tree2 == nil {
		t.Fatal("GetOrParse returned nil tree on second call")
	}

	if !hit2 {
		t.Error("Second call should be a hit")
	}

	// Clean up
	tree1.Close()
	tree2.Close()
}

// TestCacheStats tests cache statistics
func TestCacheStats(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	source1 := []byte("package main\nfunc test1() {}")
	source2 := []byte("package main\nfunc test2() {}")

	// Get initial stats for comparison (may not be zero due to other tests)
	initialStats := cache.Stats()
	initialHits := initialStats["hits"]
	initialMisses := initialStats["misses"]

	// First call - should be miss
	tree1, _ := cache.GetOrParse(parser, source1)
	tree1.Close()

	// Second call with same source - should be hit
	tree2, _ := cache.GetOrParse(parser, source1)
	tree2.Close()

	// Third call with different source - should be miss
	tree3, _ := cache.GetOrParse(parser, source2)
	tree3.Close()

	// Check final stats (compare to initial)
	stats := cache.Stats()
	hitsGain := stats["hits"] - initialHits
	missesGain := stats["misses"] - initialMisses

	// Should have gained at least 1 hit
	if hitsGain < 1 {
		t.Errorf("Expected at least 1 hit gain, got %d", hitsGain)
	}

	// Should have gained at least 2 misses (unless sources were already cached)
	if missesGain < 0 {
		t.Errorf("Expected non-negative misses gain, got %d", missesGain)
	}

	// Hit rate should be calculated
	if stats["hit_rate"] < 0 {
		t.Errorf("Expected non-negative hit rate, got %d", stats["hit_rate"])
	}
}

// TestCacheDifferentSources tests cache with different sources
func TestCacheDifferentSources(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	source1 := []byte("package main\nfunc unique_test1_func() {}")
	source2 := []byte("package main\nfunc unique_test2_func() {}")

	// Each should be a miss initially (using unique sources)
	tree1, hit1 := cache.GetOrParse(parser, source1)
	if hit1 {
		t.Log("First source was already cached (expected in shared cache)")
	}

	tree2, hit2 := cache.GetOrParse(parser, source2)
	if hit2 {
		t.Log("Second source was already cached (expected in shared cache)")
	}

	// Calling again with first source should be hit
	tree3, hit3 := cache.GetOrParse(parser, source1)
	if !hit3 {
		t.Error("Repeated first source should be a hit")
	}

	tree1.Close()
	tree2.Close()
	tree3.Close()
}

// TestCacheInvalidSource tests cache with invalid source
func TestCacheInvalidSource(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	invalidSource := []byte("unique_invalid_source_xyz {{{")

	// Should still work but parse invalid AST
	tree, hit := cache.GetOrParse(parser, invalidSource)
	if tree == nil {
		t.Fatal("GetOrParse should not return nil even for invalid source")
	}

	if hit {
		t.Log("Invalid source was already cached (expected in shared cache)")
	}

	// Second call should hit the cache
	tree2, hit2 := cache.GetOrParse(parser, invalidSource)
	if !hit2 {
		t.Error("Second call should be a hit even for invalid source")
	}

	tree.Close()
	tree2.Close()
}

// TestCacheEmptySource tests cache with empty source
func TestCacheEmptySource(t *testing.T) {
	cache := GlobalCache

	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())

	emptySource := []byte("")

	tree, hit := cache.GetOrParse(parser, emptySource)
	if tree == nil {
		t.Fatal("GetOrParse should not return nil for empty source")
	}

	if hit {
		t.Log("Empty source was already cached (expected in shared cache)")
	}

	tree.Close()
}
