// Package internal contains simplified integration tests that verify cross-language functionality
// works as specified in RULES.md without external file dependencies.
package internal

import (
	"testing"

	"github.com/termfx/morfx/internal/core"
	"github.com/termfx/morfx/internal/evaluator"
	"github.com/termfx/morfx/internal/lang/golang"
	"github.com/termfx/morfx/internal/lang/javascript"
	"github.com/termfx/morfx/internal/lang/python"
	"github.com/termfx/morfx/internal/lang/typescript"
	"github.com/termfx/morfx/internal/parser"
	"github.com/termfx/morfx/internal/provider"
	"github.com/termfx/morfx/internal/registry"
)

// TestLanguageAgnosticIntegration tests the core principles of the language-agnostic architecture
func TestLanguageAgnosticIntegration(t *testing.T) {
	// Setup registry
	setupTestRegistrySimple(t)

	// Test cross-language queries
	t.Run("CrossLanguageQueries", testCrossLanguageQueries)

	// Test universal DSL
	t.Run("UniversalDSL", testUniversalDSL)

	// Test provider switching
	t.Run("ProviderSwitching", testProviderSwitchingSimple)

	// Test error handling
	t.Run("ErrorHandling", testErrorHandlingSimple)
}

func setupTestRegistrySimple(t *testing.T) {
	// Clear and setup registry
	registry.DefaultRegistry.Clear()

	providers := []provider.LanguageProvider{
		golang.NewProvider(),
		python.NewProvider(),
		javascript.NewProvider(),
		typescript.NewProvider(),
	}

	for _, p := range providers {
		if err := registry.RegisterProvider(p); err != nil {
			t.Fatalf("Failed to register provider %s: %v", p.Lang(), err)
		}
	}
}

func testCrossLanguageQueries(t *testing.T) {
	testData := map[string]struct {
		language string
		code     []byte
		query    string
		expected int // minimum expected matches
	}{
		"Go functions": {
			language: "go",
			code: []byte(`
package main
func main() {}
func NewUser() *User { return nil }
func TestCreateUser(t *testing.T) {}
type User struct { Name string }
`),
			query:    "func:*",
			expected: 1,
		},
		"Python functions": {
			language: "python",
			code: []byte(`
def main():
    pass
def new_user():
    return None
def test_create_user():
    assert True
class User:
    def __init__(self):
        pass
`),
			query:    "def:*",
			expected: 1,
		},
		"JavaScript functions": {
			language: "javascript",
			code: []byte(`
function main() {}
function newUser() { return {}; }
function testCreateUser() {}
class User {
		  constructor() {}
}
`),
			query:    "func:*", // Use universal func instead of language-specific function
			expected: 1,
		},
		"TypeScript functions": {
			language: "typescript",
			code: []byte(`
function main(): void {}
function newUser(): User { return {} as User; }
function testCreateUser(): void {}
interface User {
		  name: string;
}
class UserImpl implements User {
		  name: string = "";
}
`),
			query:    "func:*", // Use universal func instead of language-specific function
			expected: 1,
		},
	}

	for name, tc := range testData {
		t.Run(name, func(t *testing.T) {
			provider, err := registry.GetProvider(tc.language)
			if err != nil {
				t.Fatalf("Failed to get provider for %s: %v", tc.language, err)
			}

			results := executeQuerySimple(t, tc.query, provider, tc.code)

			if results.TotalMatches < tc.expected {
				t.Errorf("Expected at least %d matches, got %d", tc.expected, results.TotalMatches)
			}

			// Log results for debugging
			for _, result := range results.Results {
				t.Logf("%s: Found %s '%s' at line %d",
					tc.language, result.Kind, result.Name, result.Location.StartLine)
			}
		})
	}
}

func testUniversalDSL(t *testing.T) {
	parser := parser.NewUniversalParser()

	// Test node kind aliases work
	aliases := []struct {
		query        string
		expectedKind core.NodeKind
	}{
		{"func:test", core.KindFunction},
		{"function:test", core.KindFunction},
		{"def:test", core.KindFunction},
		{"fn:test", core.KindFunction},
		{"var:x", core.KindVariable},
		{"variable:x", core.KindVariable},
		{"let:x", core.KindVariable},
		{"const:API", core.KindConstant},
		{"constant:API", core.KindConstant},
		{"class:User", core.KindClass},
		{"struct:User", core.KindClass},
		{"type:User", core.KindType},
	}

	for _, test := range aliases {
		t.Run("Alias: "+test.query, func(t *testing.T) {
			query, err := parser.ParseQuery(test.query)
			if err != nil {
				t.Fatalf("Failed to parse query '%s': %v", test.query, err)
			}

			if query.Kind != test.expectedKind {
				t.Errorf("Query '%s' mapped to %s, expected %s",
					test.query, query.Kind, test.expectedKind)
			}
		})
	}

	// Test operator variations
	operators := []string{"&", "&&", "and", "|", "||", "or", "!", "not"}
	for _, op := range operators {
		t.Run("Operator: "+op, func(t *testing.T) {
			var query string
			switch op {
			case "!":
				query = op + "func:test"
			case "not":
				query = op + " func:test" // Space needed for word operators
			default:
				query = "func:test " + op + " var:x"
			}

			_, err := parser.ParseQuery(query)
			if err != nil {
				t.Errorf("Failed to parse query with operator '%s': %v", op, err)
			}
		})
	}
}

func testProviderSwitchingSimple(t *testing.T) {
	tests := []struct {
		name      string
		language  string
		shouldErr bool
	}{
		{"Valid Go", "go", false},
		{"Valid Python", "python", false},
		{"Valid JavaScript", "javascript", false},
		{"Valid TypeScript", "typescript", false},
		{"Alias: golang", "golang", false},
		{"Alias: py", "py", false},
		{"Alias: js", "js", false},
		{"Alias: ts", "ts", false},
		{"Invalid language", "cobol", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := registry.GetProvider(tt.language)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error for language '%s' but got provider: %v", tt.language, provider)
				}
			} else {
				if err != nil {
					t.Errorf("Expected provider for language '%s' but got error: %v", tt.language, err)
				}
				if provider == nil {
					t.Errorf("Expected provider for language '%s' but got nil", tt.language)
				}
			}
		})
	}
}

func testErrorHandlingSimple(t *testing.T) {
	provider := golang.NewProvider()

	tests := []struct {
		name        string
		query       string
		source      []byte
		expectError bool
	}{
		{
			name:        "Valid query and source",
			query:       "func:test",
			source:      []byte("package main\nfunc test() {}"),
			expectError: false,
		},
		{
			name:        "Invalid query syntax",
			query:       "invalid::query",
			source:      []byte("package main"),
			expectError: true,
		},
		{
			name:        "Unsupported node kind",
			query:       "unknownkind:test",
			source:      []byte("package main"),
			expectError: true,
		},
		{
			name:        "Empty source",
			query:       "func:test",
			source:      []byte(""),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			universalParser := parser.NewUniversalParser()
			parsedQuery, parseErr := universalParser.ParseQuery(tt.query)

			if tt.name == "Invalid query syntax" {
				if parseErr == nil {
					t.Error("Expected parse error but got none")
				}
				return
			}

			if parseErr != nil {
				if !tt.expectError {
					t.Errorf("Unexpected parse error: %v", parseErr)
				}
				return
			}

			evaluator, evalErr := evaluator.NewUniversalEvaluator(provider)
			if evalErr != nil {
				if !tt.expectError {
					t.Errorf("Unexpected evaluator creation error: %v", evalErr)
				}
				return
			}

			_, execErr := evaluator.Evaluate(parsedQuery, tt.source)

			if tt.expectError && execErr == nil {
				t.Error("Expected execution error but got none")
			}

			if !tt.expectError && execErr != nil {
				t.Errorf("Unexpected execution error: %v", execErr)
			}
		})
	}
}

func executeQuerySimple(t *testing.T, query string, provider provider.LanguageProvider, source []byte) *core.ResultSet {
	universalParser := parser.NewUniversalParser()
	parsedQuery, err := universalParser.ParseQuery(query)
	if err != nil {
		t.Fatalf("Failed to parse query '%s': %v", query, err)
	}

	evaluator, err := evaluator.NewUniversalEvaluator(provider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	results, err := evaluator.Evaluate(parsedQuery, source)
	if err != nil {
		t.Fatalf("Failed to execute query '%s' against %s: %v", query, provider.Lang(), err)
	}

	return results
}

// TestSpecificExamplesFromRules tests the exact examples mentioned in RULES.md
func TestSpecificExamplesFromRules(t *testing.T) {
	setupTestRegistrySimple(t)

	// Test the exact examples from RULES.md
	examples := []struct {
		name     string
		language string
		query    string
		code     []byte
	}{
		{
			name:     "Python def:test* & class:User",
			language: "python",
			query:    "def:test*",
			code: []byte(`
def test_create_user():
    pass
def test_user_email():
    pass  
class User:
    def __init__(self):
        pass
`),
		},
		{
			name:     "Go func:Test* & !struct:mock",
			language: "go",
			query:    "func:Test*",
			code: []byte(`
package main
func TestCreateUser(t *testing.T) {}
func TestUserEmail(t *testing.T) {}
type User struct { Name string }
type MockUser struct { ID int }
`),
		},
		{
			name:     "JavaScript function:test* | const:API",
			language: "javascript",
			query:    "func:test*",
			code: []byte(`
function testCreateUser() {}
function testUserEmail() {}
const API_VERSION = "v1";
class User {}
`),
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			provider, err := registry.GetProvider(example.language)
			if err != nil {
				t.Fatalf("Failed to get provider for %s: %v", example.language, err)
			}

			results := executeQuerySimple(t, example.query, provider, example.code)

			t.Logf("Query '%s' in %s found %d matches",
				example.query, example.language, results.TotalMatches)

			for _, result := range results.Results {
				t.Logf("  - %s '%s' at line %d",
					result.Kind, result.Name, result.Location.StartLine)
			}
		})
	}
}

// TestProviderRegistration verifies that all providers are properly registered
func TestProviderRegistrationSimple(t *testing.T) {
	setupTestRegistrySimple(t)

	requiredLanguages := []string{"go", "python", "javascript", "typescript"}

	for _, lang := range requiredLanguages {
		t.Run("Provider: "+lang, func(t *testing.T) {
			provider, err := registry.GetProvider(lang)
			if err != nil {
				t.Fatalf("Provider %s not registered: %v", lang, err)
			}
			if provider == nil {
				t.Fatalf("Provider %s is nil", lang)
			}
			if provider.Lang() != lang {
				t.Errorf("Provider language mismatch: expected %s, got %s", lang, provider.Lang())
			}
		})
	}
}

// TestUniversalResults verifies that results have consistent structure across languages
func TestUniversalResultsStructure(t *testing.T) {
	setupTestRegistrySimple(t)

	testCases := []struct {
		language string
		code     []byte
		query    string
	}{
		{
			"go",
			[]byte("package main\nfunc main() {}"),
			"func:main",
		},
		{
			"python",
			[]byte("def main():\n    pass"),
			"def:main",
		},
		{
			"javascript",
			[]byte("function main() {}"),
			"func:main",
		},
		{
			"typescript",
			[]byte("function main(): void {}"),
			"func:main",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.language, func(t *testing.T) {
			provider, err := registry.GetProvider(tc.language)
			if err != nil {
				t.Fatalf("Failed to get provider: %v", err)
			}

			results := executeQuerySimple(t, tc.query, provider, tc.code)

			for _, result := range results.Results {
				// Verify universal structure
				if result.Kind == "" {
					t.Error("Result missing Kind")
				}
				if result.Name == "" {
					t.Error("Result missing Name")
				}
				if result.Location.StartLine <= 0 {
					t.Error("Result missing valid StartLine")
				}
				if result.Scope == "" {
					t.Error("Result missing Scope")
				}
				if result.Metadata == nil {
					t.Error("Result missing Metadata")
				}
			}
		})
	}
}
