package providers

import (
	"testing"

	"github.com/termfx/morfx/core"
)

// MockProvider for testing
type MockProvider struct {
	language   string
	extensions []string
}

func (m *MockProvider) Language() string {
	return m.language
}

func (m *MockProvider) Extensions() []string {
	return m.extensions
}

func (m *MockProvider) Query(source string, query core.AgentQuery) core.QueryResult {
	return core.QueryResult{
		Matches: []core.Match{},
	}
}

func (m *MockProvider) Transform(source string, op core.TransformOp) core.TransformResult {
	return core.TransformResult{
		Modified: source,
	}
}

func (m *MockProvider) Validate(source string) ValidationResult {
	return ValidationResult{Valid: true}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()

	if registry == nil {
		t.Error("NewRegistry should return non-nil registry")
	}

	if registry.providers == nil {
		t.Error("Registry providers map should be initialized")
	}
}

func TestRegisterProvider(t *testing.T) {
	registry := NewRegistry()
	mockProvider := &MockProvider{
		language:   "go",
		extensions: []string{".go"},
	}

	registry.Register(mockProvider)

	// Verify provider was registered
	provider, exists := registry.Get("go")
	if !exists {
		t.Error("Provider should be registered")
	}

	if provider.Language() != "go" {
		t.Errorf("Expected language 'go', got '%s'", provider.Language())
	}
}

func TestGetProvider(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name     string
		language string
		setup    func()
		exists   bool
	}{
		{
			name:     "existing provider",
			language: "go",
			setup: func() {
				registry.Register(&MockProvider{
					language:   "go",
					extensions: []string{".go"},
				})
			},
			exists: true,
		},
		{
			name:     "non-existing provider",
			language: "rust",
			setup:    func() {},
			exists:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			provider, exists := registry.Get(tt.language)

			if exists != tt.exists {
				t.Errorf("Expected exists=%v, got %v", tt.exists, exists)
			}

			if tt.exists && provider.Language() != tt.language {
				t.Errorf("Expected language '%s', got '%s'", tt.language, provider.Language())
			}
		})
	}
}

func TestMultipleProviders(t *testing.T) {
	registry := NewRegistry()

	// Register multiple providers
	providers := []*MockProvider{
		{language: "go", extensions: []string{".go"}},
		{language: "javascript", extensions: []string{".js", ".jsx"}},
		{language: "python", extensions: []string{".py"}},
	}

	for _, p := range providers {
		registry.Register(p)
	}

	// Test all providers are accessible
	for _, expected := range providers {
		provider, exists := registry.Get(expected.language)
		if !exists {
			t.Errorf("Provider %s should exist", expected.language)
		}

		if provider.Language() != expected.language {
			t.Errorf("Expected language %s, got %s", expected.language, provider.Language())
		}
	}
}

func TestProviderOverwrite(t *testing.T) {
	registry := NewRegistry()

	// Register initial provider
	provider1 := &MockProvider{language: "go", extensions: []string{".go"}}
	registry.Register(provider1)

	// Register new provider with same language
	provider2 := &MockProvider{language: "go", extensions: []string{".go", ".mod"}}
	registry.Register(provider2)

	// Should get the latest provider
	retrieved, exists := registry.Get("go")
	if !exists {
		t.Error("Provider should exist")
	}

	// Check it's the second provider by extensions
	if len(retrieved.Extensions()) != 2 {
		t.Error("Should have gotten the second provider with 2 extensions")
	}
}
