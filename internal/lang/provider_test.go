package lang

import (
	"fmt"
	"testing"
)

// mockProvider is a simple implementation of LanguageProvider for testing.
type mockProvider struct {
	Name string
}

func (p *mockProvider) GetQuery(nodeType, nodeName string) (string, bool) {
	if nodeType == "test" {
		return fmt.Sprintf("query_from_%s_for_%s", p.Name, nodeName), true
	}
	return "", false
}

func TestRegister(t *testing.T) {
	// Clean up providers map for isolated test
	providers = make(map[string]LanguageProvider)

	// Test case 1: Successful registration
	t.Run("SuccessfulRegistration", func(t *testing.T) {
		Register("mock", &mockProvider{Name: "mock1"})
		if _, ok := providers["mock"]; !ok {
			t.Error("Register failed: provider 'mock' was not registered")
		}
	})

	// Test case 2: Panic on nil provider
	t.Run("PanicOnNilProvider", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Register did not panic on nil provider")
			}
		}()
		Register("nil_provider", nil)
	})
}

func TestRegister_Duplicate(t *testing.T) {
	// Clean up providers map for isolated test
	providers = make(map[string]LanguageProvider)
	Register("duplicate", &mockProvider{Name: "original"})

	// Test case: Panic on duplicate provider registration
	t.Run("PanicOnDuplicateProvider", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Register did not panic on duplicate provider registration")
			}
		}()
		Register("duplicate", &mockProvider{Name: "imposter"})
	})
}

func TestGetQueryForLanguage(t *testing.T) {
	// Clean up and set up a known provider
	providers = make(map[string]LanguageProvider)
	Register("mock_lang", &mockProvider{Name: "mock_lang"})

	// Test case 1: Successful query retrieval
	t.Run("SuccessfulQueryRetrieval", func(t *testing.T) {
		query, err := GetQueryForLanguage("mock_lang", "test", "MyNode")
		if err != nil {
			t.Fatalf("GetQueryForLanguage failed unexpectedly: %v", err)
		}
		expectedQuery := "query_from_mock_lang_for_MyNode"
		if query != expectedQuery {
			t.Errorf("Expected query '%s', got '%s'", expectedQuery, query)
		}
	})

	// Test case 2: Unsupported language
	t.Run("UnsupportedLanguage", func(t *testing.T) {
		_, err := GetQueryForLanguage("unsupported_lang", "test", "MyNode")
		if err == nil {
			t.Error("Expected error for unsupported language, but got nil")
		}
		expectedErrorMsg := "unsupported language: unsupported_lang"
		if err.Error() != expectedErrorMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
		}
	})

	// Test case 3: Unsupported node type
	t.Run("UnsupportedNodeType", func(t *testing.T) {
		_, err := GetQueryForLanguage("mock_lang", "unsupported_type", "MyNode")
		if err == nil {
			t.Error("Expected error for unsupported node type, but got nil")
		}
		expectedErrorMsg := "unsupported node type 'unsupported_type' for language 'mock_lang'"
		if err.Error() != expectedErrorMsg {
			t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, err.Error())
		}
	})
}
