package registry

import (
	"fmt"
	"sync"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/termfx/morfx/internal/types"
)

// MockLanguageProvider implements the MorfxLanguageProvider interface for testing
type MockLanguageProvider struct {
	lang       string
	aliases    []string
	extensions []string
	language   *sitter.Language
}

func (m *MockLanguageProvider) Lang() string {
	return m.lang
}

func (m *MockLanguageProvider) Aliases() []string {
	return m.aliases
}

func (m *MockLanguageProvider) Extensions() []string {
	return m.extensions
}

func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	return m.language
}

func (m *MockLanguageProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping {
	return []types.NodeMapping{}
}

func (m *MockLanguageProvider) TranslateQuery(q *types.Query) (string, error) {
	return "(function_declaration)", nil
}

func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return map[string]string{}
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	return "function"
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
	return "file"
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node {
	return node
}

func (m *MockLanguageProvider) IsBlockLevelNode(nodeType string) bool {
	return false
}

func (m *MockLanguageProvider) GetDefaultIgnorePatterns() (files []string, symbols []string) {
	return []string{}, []string{}
}

func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	return types.NodeKind(dslKind)
}

func (m *MockLanguageProvider) GetSupportedDSLKinds() []string {
	return []string{"function", "variable", "class"}
}

func TestNewLanguageRegistry(t *testing.T) {
	registry := NewLanguageRegistry()

	if registry == nil {
		t.Error("Expected non-nil registry")
	}

	if len(registry.providers) != 0 {
		t.Error("Expected empty registry")
	}

	// Test that aliases and extensions are cleaned up
	_, err := registry.GetProvider("golang")
	if err == nil {
		t.Error("Expected error when getting provider by alias after clear")
	}

	_, err = registry.GetProviderByExtension(".go")
	if err == nil {
		t.Error("Expected error when getting provider by extension after clear")
	}
}

func TestRegisterProvider(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	err := registry.Register(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify provider is registered
	if len(registry.providers) != 1 {
		t.Error("Expected 1 provider")
	}

	// Test that provider can be retrieved by alias
	_, err = registry.GetProvider("golang")
	if err != nil {
		t.Errorf("Expected to find provider by alias, got error: %v", err)
	}

	// Test that provider can be retrieved by extension
	_, err = registry.GetProviderByExtension(".go")
	if err != nil {
		t.Errorf("Expected to find provider by extension, got error: %v", err)
	}
}

func TestRegisterProviderDuplicate(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider1 := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}
	mockProvider2 := &MockLanguageProvider{
		lang:       "go", // Same language
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	// Register first provider
	err := registry.Register(mockProvider1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Try to register duplicate
	err = registry.Register(mockProvider2)
	if err == nil {
		t.Error("Expected error for duplicate provider")
	}
}

func TestGetProvider(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	// Register provider
	err := registry.Register(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test getting by language name
	provider, err := registry.GetProvider("go")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting by alias
	provider, err = registry.GetProvider("golang")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting non-existent provider
	_, err = registry.GetProvider("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestGetProviderByExtension(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go", ".mod"},
	}

	// Register provider
	err := registry.Register(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test getting by extension
	provider, err := registry.GetProviderByExtension(".go")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting by another extension
	provider, err = registry.GetProviderByExtension(".mod")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if provider.Lang() != mockProvider.Lang() {
		t.Error("Expected same provider instance")
	}

	// Test getting non-existent extension
	_, err = registry.GetProviderByExtension(".unknown")
	if err == nil {
		t.Error("Expected error for non-existent extension")
	}
}

func TestListProviders(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider1 := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}
	mockProvider2 := &MockLanguageProvider{
		lang:       "python",
		aliases:    []string{"py"},
		extensions: []string{".py"},
	}

	// Initially empty
	providers := registry.ListProviders()
	if len(providers) != 0 {
		t.Error("Expected empty provider list")
	}

	// Register providers
	err := registry.Register(mockProvider1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = registry.Register(mockProvider2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check list
	providers = registry.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Verify provider names
	names := make(map[string]bool)
	for _, providerName := range providers {
		names[providerName] = true
	}

	if !names["go"] || !names["python"] {
		t.Error("Expected both 'go' and 'python' providers")
	}
}

func TestUnregisterProvider(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	// Register provider
	err := registry.Register(mockProvider)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify it's registered
	if len(registry.providers) != 1 {
		t.Error("Expected 1 provider")
	}

	// Unregister
	err = registry.Unregister("go")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify it's removed
	if len(registry.providers) != 0 {
		t.Error("Expected 0 providers")
	}

	// Test that aliases and extensions are cleaned up
	_, err = registry.GetProvider("golang")
	if err == nil {
		t.Error("Expected error when getting provider by alias after unregister")
	}

	_, err = registry.GetProviderByExtension(".go")
	if err == nil {
		t.Error("Expected error when getting provider by extension after unregister")
	}

	// Try to unregister non-existent provider
	err = registry.Unregister("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestGetSupportedExtensions(t *testing.T) {
	registry := NewLanguageRegistry()
	mockProvider1 := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go", ".mod"},
	}
	mockProvider2 := &MockLanguageProvider{
		lang:       "python",
		aliases:    []string{"py"},
		extensions: []string{".py", ".pyx"},
	}

	// Register providers
	err := registry.Register(mockProvider1)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	err = registry.Register(mockProvider2)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Get supported extensions
	extensions := registry.ListExtensions()
	if len(extensions) != 4 {
		t.Errorf("Expected 4 extensions, got %d", len(extensions))
	}

	// Verify all extensions are present
	expected := map[string]bool{
		".go":  true,
		".mod": true,
		".py":  true,
		".pyx": true,
	}

	for _, ext := range extensions {
		if !expected[ext] {
			t.Errorf("Unexpected extension: %s", ext)
		}
		delete(expected, ext)
	}

	if len(expected) > 0 {
		t.Errorf("Missing extensions: %v", expected)
	}
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewLanguageRegistry()

	// Test concurrent registration and access
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			mockProvider := &MockLanguageProvider{
				lang:       fmt.Sprintf("lang%d", id),
				aliases:    []string{fmt.Sprintf("alias%d", id)},
				extensions: []string{fmt.Sprintf(".ext%d", id)},
			}
			_ = registry.Register(mockProvider)
			_, _ = registry.GetProvider(fmt.Sprintf("lang%d", id))
		}(i)
	}
	wg.Wait()

	// Verify all providers were registered
	providers := registry.ListProviders()
	if len(providers) != 10 {
		t.Errorf("Expected 10 providers, got %d", len(providers))
	}
}

// Benchmark tests
func BenchmarkRegisterProvider(b *testing.B) {
	registry := NewLanguageRegistry()

	for i := 0; b.Loop(); i++ {
		mockProvider := &MockLanguageProvider{
			lang:       fmt.Sprintf("lang%d", i),
			aliases:    []string{fmt.Sprintf("alias%d", i)},
			extensions: []string{fmt.Sprintf(".ext%d", i)},
		}
		_ = registry.Register(mockProvider)
		_ = registry.Unregister(fmt.Sprintf("lang%d", i)) // Clean up
	}
}

func BenchmarkGetProvider(b *testing.B) {
	registry := NewLanguageRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	_ = registry.Register(mockProvider)

	for b.Loop() {
		_, _ = registry.GetProvider("go")
	}
}

func BenchmarkGetProviderByExtension(b *testing.B) {
	registry := NewLanguageRegistry()
	mockProvider := &MockLanguageProvider{
		lang:       "go",
		aliases:    []string{"golang"},
		extensions: []string{".go"},
	}

	_ = registry.Register(mockProvider)

	for b.Loop() {
		_, _ = registry.GetProviderByExtension(".go")
	}
}
