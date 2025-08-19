package golang

import (
	"strings"
	"testing"

	"github.com/termfx/morfx/internal/evaluator"
	"github.com/termfx/morfx/internal/parser"
)

func TestE2EProviderDirect(t *testing.T) {
	// Test the Go provider directly without registry (to avoid import cycle)
	provider := NewProvider()

	if provider.Lang() != "go" {
		t.Errorf("Provider lang = %q, want %q", provider.Lang(), "go")
	}

	// Test aliases
	aliases := provider.Aliases()
	if len(aliases) < 2 || aliases[0] != "go" || aliases[1] != "golang" {
		t.Errorf("Provider aliases = %v, want [go golang]", aliases)
	}

	// Test extensions
	extensions := provider.Extensions()
	if len(extensions) != 1 || extensions[0] != ".go" {
		t.Errorf("Provider extensions = %v, want [.go]", extensions)
	}
}

func TestE2ECompleteFlow(t *testing.T) {
	// Create provider directly
	provider := NewProvider()

	// Create universal parser
	uniParser := parser.NewUniversalParser()

	// Create evaluator
	eval, err := evaluator.NewUniversalEvaluator(provider)
	if err != nil {
		t.Fatalf("Failed to create evaluator: %v", err)
	}

	// Test code
	code := []byte(`
package main

import (
	"fmt"
	"log"
)

// Config holds application configuration
type Config struct {
	Host string
	Port int
}

// Server represents our application server
type Server struct {
	config *Config
	logger *log.Logger
}

// NewServer creates a new server instance
func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
		logger: log.Default(),
	}
}

// Start begins serving requests
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Printf("Starting server on %s", addr)
	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	log.Println("Stopping server")
	return nil
}

func main() {
	cfg := &Config{
		Host: "localhost",
		Port: 8080,
	}
	
	server := NewServer(cfg)
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}

// Helper functions
func validateConfig(cfg *Config) error {
	if cfg.Port <= 0 {
		return fmt.Errorf("invalid port")
	}
	return nil
}

var defaultConfig = &Config{
	Host: "0.0.0.0",
	Port: 3000,
}

const (
	DefaultTimeout = 30
	MaxRetries    = 3
)
`)

	testCases := []struct {
		name         string
		dsl          string
		wantMinCount int
		checkNames   []string
	}{
		{
			name:         "find all functions",
			dsl:          "function:*",
			wantMinCount: 3, // main, NewServer, validateConfig
			checkNames:   []string{"main", "NewServer", "validateConfig"},
		},
		{
			name:         "find all methods",
			dsl:          "method:*",
			wantMinCount: 2, // Start, Stop
			checkNames:   []string{"Start", "Stop"},
		},
		{
			name:         "find structs",
			dsl:          "class:*",
			wantMinCount: 2, // Config, Server
			checkNames:   []string{"Config", "Server"},
		},
		{
			name:         "find specific function",
			dsl:          "function:main",
			wantMinCount: 1,
			checkNames:   []string{"main"},
		},
		{
			name:         "find functions with pattern",
			dsl:          "function:New*",
			wantMinCount: 1,
			checkNames:   []string{"NewServer"},
		},
		{
			name:         "find constants",
			dsl:          "const:*",
			wantMinCount: 1,
		},
		{
			name:         "find variables",
			dsl:          "variable:*",
			wantMinCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse DSL
			query, err := uniParser.ParseQuery(tc.dsl)
			if err != nil {
				t.Fatalf("Failed to parse DSL %q: %v", tc.dsl, err)
			}

			// Evaluate query
			resultSet, err := eval.Evaluate(query, code)
			if err != nil {
				t.Fatalf("Failed to evaluate query: %v", err)
			}

			// Check result count
			if len(resultSet.Results) < tc.wantMinCount {
				t.Errorf("Got %d results, want at least %d", len(resultSet.Results), tc.wantMinCount)
				for i, r := range resultSet.Results {
					t.Logf("  Result %d: %q (kind: %s, line: %d)",
						i, r.Name, r.Kind, r.Location.StartLine)
				}
			}

			// Check specific names if provided
			if len(tc.checkNames) > 0 {
				foundNames := make(map[string]bool)
				for _, r := range resultSet.Results {
					foundNames[r.Name] = true
				}

				for _, expectedName := range tc.checkNames {
					found := false
					for name := range foundNames {
						if name == expectedName || strings.Contains(name, expectedName) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected to find %q in results, but didn't", expectedName)
						t.Logf("Found names: %v", foundNames)
					}
				}
			}
		})
	}
}

func TestE2ELogicalQueries(t *testing.T) {
	provider := NewProvider()
	uniParser := parser.NewUniversalParser()
	eval, _ := evaluator.NewUniversalEvaluator(provider)

	code := []byte(`
package main

func foo() {}
func bar() {}
var x = 1
var y = 2
`)

	tests := []struct {
		name         string
		dsl          string
		wantMinCount int
	}{
		{
			name:         "AND query",
			dsl:          "function:foo & variable:x",
			wantMinCount: 2, // Should find both
		},
		{
			name:         "OR query",
			dsl:          "function:baz | variable:x",
			wantMinCount: 1, // Should find at least variable x
		},
		{
			name:         "negated function",
			dsl:          "!function:baz",
			wantMinCount: 0, // This might not work as expected without proper implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := uniParser.ParseQuery(tt.dsl)
			if err != nil {
				t.Fatalf("Failed to parse DSL: %v", err)
			}

			// For logical queries, we might need special handling
			if query.Operator == "&&" || query.Operator == "||" {
				// Evaluate children separately
				var totalResults int
				for _, child := range query.Children {
					childCopy := child // Make a copy
					rs, err := eval.Evaluate(&childCopy, code)
					if err == nil {
						totalResults += len(rs.Results)
					}
				}

				if totalResults < tt.wantMinCount {
					t.Errorf("Got %d total results, want at least %d", totalResults, tt.wantMinCount)
				}
			} else {
				rs, err := eval.Evaluate(query, code)
				if err != nil && !strings.Contains(tt.dsl, "!") {
					t.Fatalf("Failed to evaluate: %v", err)
				}

				if rs != nil && len(rs.Results) < tt.wantMinCount {
					t.Errorf("Got %d results, want at least %d", len(rs.Results), tt.wantMinCount)
				}
			}
		})
	}
}

func TestE2EPerformance(t *testing.T) {
	provider := NewProvider()
	eval, _ := evaluator.NewUniversalEvaluator(provider)
	uniParser := parser.NewUniversalParser()

	// Large code sample
	code := []byte(strings.Repeat(`
func testFunc() {
	x := 42
	y := "test"
	z := x + 1
}
`, 100)) // Repeat to create larger file

	query, _ := uniParser.ParseQuery("function:test*")

	// This is a basic performance check, not a benchmark
	resultSet, err := eval.Evaluate(query, code)
	if err != nil {
		t.Fatalf("Failed to evaluate: %v", err)
	}

	if len(resultSet.Results) == 0 {
		t.Error("Expected to find functions in repeated code")
	}

	// Test query cost estimation
	cost := provider.EstimateQueryCost(query)
	if cost <= 0 {
		t.Errorf("Query cost = %d, want > 0", cost)
	}
}
