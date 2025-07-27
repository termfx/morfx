package golang

import (
	"fmt"
	"strings"
	"testing"
)

func TestGoProvider_TranslateDSL(t *testing.T) {
	p := &goProvider{}

	testCases := []struct {
		name          string
		dslQuery      string
		shouldContain []string
		shouldError   bool
	}{
		// Basic DSL queries
		{
			name:          "simple function query",
			dslQuery:      "func:Init",
			shouldContain: []string{"function_declaration", `(#eq? @name "Init")`, "@target"},
			shouldError:   false,
		},
		{
			name:          "simple var query",
			dslQuery:      "var:myVar",
			shouldContain: []string{"var_declaration", `(#eq? @name "myVar")`, "@target"},
			shouldError:   false,
		},
		{
			name:          "simple struct query",
			dslQuery:      "struct:User",
			shouldContain: []string{"type_declaration", `(#eq? @name "User")`, "struct_type", "@target"},
			shouldError:   false,
		},

		// Wildcard queries
		{
			name:          "wildcard prefix",
			dslQuery:      "func:Handle*",
			shouldContain: []string{"function_declaration", `(#match? @name "^Handle.*")`, "@target"},
			shouldError:   false,
		},
		{
			name:          "wildcard suffix",
			dslQuery:      "func:*Handler",
			shouldContain: []string{"function_declaration", `(#match? @name ".*Handler$")`, "@target"},
			shouldError:   false,
		},
		{
			name:          "wildcard contains",
			dslQuery:      "func:*Test*",
			shouldContain: []string{"function_declaration", `(#match? @name ".*Test.*")`, "@target"},
			shouldError:   false,
		},
		{
			name:          "wildcard all",
			dslQuery:      "func:*",
			shouldContain: []string{"function_declaration", "@target"},
			shouldError:   false,
		},

		// Negation queries
		{
			name:          "negated function",
			dslQuery:      "!func:Test*",
			shouldContain: []string{"function_declaration", `(#is-not? @name "^Test.*")`, "@target"},
			shouldError:   false,
		},

		// Parent/child relationships
		{
			name:          "function with var child",
			dslQuery:      "func:Init > var:config",
			shouldContain: []string{"function_declaration", `(#eq? @name "Init")`, "var_declaration", `(#eq? @name "config")`},
			shouldError:   false,
		},
		{
			name:          "function with wildcard child",
			dslQuery:      "func:* > call:os.Getenv",
			shouldContain: []string{"function_declaration", "call_expression", `(#eq? @name "os.Getenv")`},
			shouldError:   false,
		},

		// Error cases
		{
			name:        "empty query",
			dslQuery:    "",
			shouldError: true,
		},
		{
			name:        "invalid format",
			dslQuery:    "func",
			shouldError: true,
		},
		{
			name:        "unknown node type",
			dslQuery:    "unknown:test",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.TranslateDSL(tc.dslQuery)

			if tc.shouldError {
				if err == nil {
					t.Fatalf("Expected error for query '%s', but got none", tc.dslQuery)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for query '%s': %v", tc.dslQuery, err)
			}

			if result == "" {
				t.Fatalf("Expected non-empty result for query '%s'", tc.dslQuery)
			}

			for _, expected := range tc.shouldContain {
				if !strings.Contains(result, expected) {
					t.Errorf("Query result for '%s' should contain '%s'.\nGot: %s", tc.dslQuery, expected, result)
				}
			}

			t.Logf("DSL Query: %s\nTree-sitter Query: %s", tc.dslQuery, result)
		})
	}
}

func TestMatchesWildcard(t *testing.T) {
	testCases := []struct {
		pattern  string
		text     string
		expected bool
	}{
		// Exact matches
		{"foo", "foo", true},
		{"foo", "bar", false},

		// Wildcard all
		{"*", "anything", true},
		{"*", "", true},

		// Prefix wildcards (ends with)
		{"*foo", "foo", true},
		{"*foo", "barfoo", true},
		{"*foo", "foobar", false},

		// Suffix wildcards (starts with)
		{"foo*", "foo", true},
		{"foo*", "foobar", true},
		{"foo*", "barfoo", false},

		// Contains wildcards
		{"*foo*", "foo", true},
		{"*foo*", "barfoo", true},
		{"*foo*", "foobar", true},
		{"*foo*", "barfoobar", true},
		{"*foo*", "bar", false},

		// Complex wildcards (starts with X and ends with Y)
		{"foo*bar", "foobar", true},
		{"foo*bar", "fooxbar", true},
		{"foo*bar", "fooxyzbar", true},
		{"foo*bar", "foo", false},
		{"foo*bar", "bar", false},
		{"foo*bar", "foobarx", false},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_matches_%s", tc.pattern, tc.text), func(t *testing.T) {
			result := MatchesWildcard(tc.pattern, tc.text)
			if result != tc.expected {
				t.Errorf("MatchesWildcard(%q, %q) = %v, want %v", tc.pattern, tc.text, result, tc.expected)
			}
		})
	}
}
