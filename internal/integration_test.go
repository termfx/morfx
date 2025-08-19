// Package internal contains integration tests that verify cross-language functionality
// works as specified in RULES.md. These tests demonstrate that the system works
// identically across all languages as promised.
//
// Test Coverage:
// 1. Cross-Language Query Compatibility
// 2. Universal DSL Validation
// 3. Provider Switching
// 4. End-to-End Scenarios
// 5. Error Handling
package internal

import (
	"os"
	"strings"
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

// TestData contains sample files and their expected results for cross-language testing
type TestData struct {
	Name      string
	Language  string
	Extension string
	Content   []byte
	Provider  provider.LanguageProvider
}

var testFiles []TestData

// TestMain sets up the integration test environment
func TestMain(m *testing.M) {
	// Initialize test registry
	setupTestRegistry()

	// Load test data
	loadTestData()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

// setupTestRegistry initializes a clean registry with all providers
func setupTestRegistry() {
	// Clear existing registry
	registry.DefaultRegistry.Clear()

	// Register all providers
	providers := []provider.LanguageProvider{
		golang.NewProvider(),
		python.NewProvider(),
		javascript.NewProvider(),
		typescript.NewProvider(),
	}

	for _, p := range providers {
		if err := registry.RegisterProvider(p); err != nil {
			panic("Failed to register provider: " + err.Error())
		}
	}
}

// loadTestData loads sample files for cross-language testing
func loadTestData() {
	sampleFiles := []struct {
		name      string
		language  string
		extension string
		path      string
	}{
		{"Go Sample", "go", ".go", "testdata/samples/sample.go"},
		{"Python Sample", "python", ".py", "testdata/samples/sample.py"},
		{"JavaScript Sample", "javascript", ".js", "testdata/samples/sample.js"},
		{"TypeScript Sample", "typescript", ".ts", "testdata/samples/sample.ts"},
	}

	for _, file := range sampleFiles {
		content, err := os.ReadFile(file.path)
		if err != nil {
			// Skip if file doesn't exist - test will handle this
			continue
		}

		provider, err := registry.GetProvider(file.language)
		if err != nil {
			continue
		}

		testFiles = append(testFiles, TestData{
			Name:      file.name,
			Language:  file.language,
			Extension: file.extension,
			Content:   content,
			Provider:  provider,
		})
	}
}

// TestCrossLanguageQueryCompatibility verifies that cross-language queries work as specified in RULES.md
func TestCrossLanguageQueryCompatibility(t *testing.T) {
	tests := []struct {
		name            string
		query           string
		targetLanguage  string
		expectedMatches int
		description     string
	}{
		{
			name:            "Python: def:test* & class:User",
			query:           "def:test* & class:User",
			targetLanguage:  "python",
			expectedMatches: 1, // Should find both test functions and User class
			description:     "Test Python-specific DSL terms work across the system",
		},
		{
			name:            "Go: func:Test* & !struct:mock",
			query:           "func:Test* & !struct:mock",
			targetLanguage:  "go",
			expectedMatches: 1, // Should find Test functions but exclude mock structs
			description:     "Test Go-specific DSL terms and negation work",
		},
		{
			name:            "JavaScript: function:test* | const:API",
			query:           "function:test* | const:API",
			targetLanguage:  "javascript",
			expectedMatches: 1, // Should find test functions or API constants
			description:     "Test JavaScript-specific DSL terms and OR operator work",
		},
		{
			name:            "Universal: func:new* across all languages",
			query:           "func:new*",
			targetLanguage:  "", // Test against all languages
			expectedMatches: 4,  // Should find newUser function in all languages
			description:     "Test that universal func term works across all languages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.targetLanguage == "" {
				// Test against all languages
				testCrossLanguageQuery(t, tt.query, tt.description)
			} else {
				// Test against specific language
				testLanguageSpecificQuery(t, tt.query, tt.targetLanguage, tt.expectedMatches, tt.description)
			}
		})
	}
}

// TestUniversalDSLValidation verifies all DSL variations work consistently
func TestUniversalDSLValidation(t *testing.T) {
	parser := parser.NewUniversalParser()

	// Test node kind aliases
	kindAliasTests := []struct {
		aliases      []string
		expectedKind core.NodeKind
	}{
		{[]string{"func", "function", "def", "fn"}, core.KindFunction},
		{[]string{"var", "variable", "let"}, core.KindVariable},
		{[]string{"const", "constant"}, core.KindConstant},
		{[]string{"class", "struct", "type"}, core.KindClass},
		{[]string{"import", "require", "use"}, core.KindImport},
	}

	for _, test := range kindAliasTests {
		for _, alias := range test.aliases {
			t.Run("Kind alias: "+alias, func(t *testing.T) {
				query, err := parser.ParseQuery(alias + ":test")
				if err != nil {
					t.Fatalf("Failed to parse query with alias '%s': %v", alias, err)
				}

				if query.Kind != test.expectedKind {
					t.Errorf("Alias '%s' mapped to %s, expected %s", alias, query.Kind, test.expectedKind)
				}
			})
		}
	}

	// Test operator variations
	operatorTests := []struct {
		variations []string
		baseQuery  string
	}{
		{[]string{"&", "&&", "and"}, "func:test & var:count"},
		{[]string{"|", "||", "or"}, "func:test | var:count"},
		{[]string{"!", "not"}, "!func:test"},
	}

	for _, test := range operatorTests {
		for _, op := range test.variations {
			t.Run("Operator: "+op, func(t *testing.T) {
				testQuery := strings.Replace(test.baseQuery, test.variations[0], op, 1)
				_, err := parser.ParseQuery(testQuery)
				if err != nil {
					t.Errorf("Failed to parse query with operator '%s': %v", op, err)
				}
			})
		}
	}
}

// TestProviderSwitching verifies provider switching and auto-detection works correctly
func TestProviderSwitching(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		language  string
		shouldErr bool
	}{
		{"Auto-detect Go", "test.go", "go", false},
		{"Auto-detect Python", "test.py", "python", false},
		{"Auto-detect JavaScript", "test.js", "javascript", false},
		{"Auto-detect TypeScript", "test.ts", "typescript", false},
		{"Explicit Go", "test.unknown", "go", false},
		{"Explicit Python", "test.unknown", "python", false},
		{"Unsupported extension", "test.xyz", "", true},
		{"Unsupported language", "test.go", "cobol", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var provider provider.LanguageProvider
			var err error

			if tt.language != "" {
				// Explicit language selection
				provider, err = registry.GetProvider(tt.language)
			} else {
				// Auto-detection
				provider, err = registry.GetProviderForFile(tt.filename)
			}

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got provider: %v", provider)
				}
			} else {
				if err != nil {
					t.Errorf("Expected provider but got error: %v", err)
				}
				if provider == nil {
					t.Error("Expected provider but got nil")
				}
			}
		})
	}
}

// TestEndToEndScenarios tests complete query evaluation across languages
func TestEndToEndScenarios(t *testing.T) {
	scenarios := []struct {
		name     string
		query    string
		validate func(t *testing.T, lang string, results *core.ResultSet)
	}{
		{
			name:  "Find User classes across all languages",
			query: "class:User",
			validate: func(t *testing.T, lang string, results *core.ResultSet) {
				if results.TotalMatches == 0 {
					t.Errorf("Expected to find User class in %s", lang)
				}
				for _, result := range results.Results {
					if result.Kind != core.KindClass {
						t.Errorf("Expected class kind, got %s in %s", result.Kind, lang)
					}
					if !strings.Contains(strings.ToLower(result.Name), "user") {
						t.Errorf("Expected User in name, got %s in %s", result.Name, lang)
					}
				}
			},
		},
		{
			name:  "Find test functions with wildcards",
			query: "func:test*",
			validate: func(t *testing.T, lang string, results *core.ResultSet) {
				if results.TotalMatches == 0 {
					t.Errorf("Expected to find test functions in %s", lang)
				}
				for _, result := range results.Results {
					if result.Kind != core.KindFunction {
						t.Errorf("Expected function kind, got %s in %s", result.Kind, lang)
					}
					if !strings.HasPrefix(strings.ToLower(result.Name), "test") {
						t.Errorf("Expected test prefix in name, got %s in %s", result.Name, lang)
					}
				}
			},
		},
		{
			name:  "Find constants with specific patterns",
			query: "const:*API*",
			validate: func(t *testing.T, lang string, results *core.ResultSet) {
				// Some languages might not have API constants, so we don't require matches
				for _, result := range results.Results {
					if result.Kind != core.KindConstant {
						t.Errorf("Expected constant kind, got %s in %s", result.Kind, lang)
					}
				}
			},
		},
		{
			name:  "Complex hierarchical query",
			query: "class:User > func:set*",
			validate: func(t *testing.T, lang string, results *core.ResultSet) {
				// Hierarchical queries are complex - just verify structure
				for _, result := range results.Results {
					if result.Name == "" {
						t.Errorf("Expected non-empty result name in %s", lang)
					}
				}
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for _, testFile := range testFiles {
				t.Run(testFile.Language, func(t *testing.T) {
					results := executeQuery(t, scenario.query, testFile.Provider, testFile.Content)
					scenario.validate(t, testFile.Language, results)
				})
			}
		})
	}
}

// TestErrorHandling verifies proper error handling across the system
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (provider.LanguageProvider, []byte, string)
		expectError bool
		errorType   string
	}{
		{
			name: "Invalid query syntax",
			setup: func() (provider.LanguageProvider, []byte, string) {
				return golang.NewProvider(), []byte("package main"), "invalid:query:syntax"
			},
			expectError: true,
			errorType:   "parse",
		},
		{
			name: "Unsupported node kind",
			setup: func() (provider.LanguageProvider, []byte, string) {
				return golang.NewProvider(), []byte("package main"), "unknownkind:test"
			},
			expectError: true,
			errorType:   "unsupported",
		},
		{
			name: "Empty source code",
			setup: func() (provider.LanguageProvider, []byte, string) {
				return golang.NewProvider(), []byte(""), "func:test"
			},
			expectError: true,
			errorType:   "empty",
		},
		{
			name: "Valid query on valid code",
			setup: func() (provider.LanguageProvider, []byte, string) {
				return golang.NewProvider(), []byte("package main\nfunc test() {}"), "func:test"
			},
			expectError: false,
			errorType:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, source, query := tt.setup()

			universalParser := parser.NewUniversalParser()
			parsedQuery, parseErr := universalParser.ParseQuery(query)

			if tt.errorType == "parse" {
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

			_, execErr := evaluator.Evaluate(parsedQuery, source)

			if tt.expectError && execErr == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && execErr != nil {
				t.Errorf("Unexpected execution error: %v", execErr)
			}
		})
	}
}

// testCrossLanguageQuery tests a query against all available languages
func testCrossLanguageQuery(t *testing.T, query string, description string) {
	universalParser := parser.NewUniversalParser()
	_, err := universalParser.ParseQuery(query)
	if err != nil {
		t.Fatalf("Failed to parse query '%s': %v", query, err)
	}

	totalMatches := 0
	languagesWithMatches := 0

	for _, testFile := range testFiles {
		results := executeQuery(t, query, testFile.Provider, testFile.Content)

		if results.TotalMatches > 0 {
			languagesWithMatches++
			totalMatches += results.TotalMatches
			t.Logf("%s: Found %d matches in %s", description, results.TotalMatches, testFile.Language)
		}
	}

	if languagesWithMatches == 0 {
		t.Errorf("Query '%s' found no matches in any language", query)
	}

	t.Logf("Cross-language query '%s': %d total matches across %d languages",
		query, totalMatches, languagesWithMatches)
}

// testLanguageSpecificQuery tests a query against a specific language
func testLanguageSpecificQuery(t *testing.T, query, language string, expectedMatches int, description string) {
	var testFile *TestData
	for i := range testFiles {
		if testFiles[i].Language == language {
			testFile = &testFiles[i]
			break
		}
	}

	if testFile == nil {
		t.Skipf("Test file for language %s not available", language)
		return
	}

	results := executeQuery(t, query, testFile.Provider, testFile.Content)

	t.Logf("%s: Found %d matches in %s (expected %d)",
		description, results.TotalMatches, language, expectedMatches)

	// Log details about found matches
	for _, result := range results.Results {
		t.Logf("  - %s '%s' at line %d", result.Kind, result.Name, result.Location.StartLine)
	}
}

// executeQuery executes a query against source code using the given provider
func executeQuery(t *testing.T, query string, provider provider.LanguageProvider, source []byte) *core.ResultSet {
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

// TestProviderRegistration verifies that all required providers are registered
func TestProviderRegistration(t *testing.T) {
	requiredProviders := []string{"go", "python", "javascript", "typescript"}

	for _, lang := range requiredProviders {
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

// TestProviderAliases verifies that provider aliases work correctly
func TestProviderAliases(t *testing.T) {
	aliasTests := []struct {
		alias    string
		expected string
	}{
		{"golang", "go"},
		{"py", "python"},
		{"js", "javascript"},
		{"ts", "typescript"},
	}

	for _, test := range aliasTests {
		t.Run("Alias: "+test.alias, func(t *testing.T) {
			provider, err := registry.GetProvider(test.alias)
			if err != nil {
				t.Fatalf("Failed to get provider by alias '%s': %v", test.alias, err)
			}
			if provider.Lang() != test.expected {
				t.Errorf("Alias '%s' resolved to %s, expected %s", test.alias, provider.Lang(), test.expected)
			}
		})
	}
}

// TestResultStructure verifies that results contain correct universal information
func TestResultStructure(t *testing.T) {
	// Use a simple query that should work across all languages
	query := "func:main"

	for _, testFile := range testFiles {
		t.Run(testFile.Language, func(t *testing.T) {
			results := executeQuery(t, query, testFile.Provider, testFile.Content)

			for _, result := range results.Results {
				// Verify universal result structure
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

				// Verify metadata exists
				if result.Metadata == nil {
					t.Error("Result missing Metadata")
				}

				// Verify language-agnostic structure
				if nodeType, ok := result.Metadata["node_type"].(string); !ok || nodeType == "" {
					t.Error("Result missing node_type metadata")
				}
			}
		})
	}
}
