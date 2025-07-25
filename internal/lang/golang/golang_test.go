package golang

import (
	"strings"
	"testing"
)

func TestGoProvider_GetQuery(t *testing.T) {
	p := &goProvider{}

	testCases := []struct {
		nodeType      string
		nodeName      string
		shouldContain string
		shouldExist   bool
	}{
		// Basic cases
		{"function", "MyFunction", `(#eq? @name "MyFunction")`, true},
		{"method", "MyMethod", `(#eq? @name "MyMethod")`, true},
		{"import", "fmt", `(#eq? @path "\"fmt\"")`, true},
		{"struct", "MyStruct", `(#eq? @name "MyStruct")`, true},
		{"interface", "MyInterface", `(#eq? @name "MyInterface")`, true},
		{"type", "MyType", `(#eq? @name "MyType")`, true},
		{"const", "MyConst", `(#eq? @name "MyConst")`, true},
		{"var", "myVar", `(#eq? @name "myVar")`, true},
		{"package", "main", `(#eq? @name "main")`, true},

		// Edge case: node name with underscores
		{"function", "my_private_func", `(#eq? @name "my_private_func")`, true},

		// Edge case: import path with slashes
		{"import", "github.com/user/repo", `(#eq? @path "\"github.com/user/repo\"")`, true},

		// Failure case
		{"nonexistent", "SomeName", "", false},
		{"", "SomeName", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.nodeType, func(t *testing.T) {
			query, ok := p.GetQuery(tc.nodeType, tc.nodeName)

			if ok != tc.shouldExist {
				t.Fatalf("Expected existence to be %v, but got %v", tc.shouldExist, ok)
			}

			if tc.shouldExist {
				if query == "" {
					t.Fatal("Expected a query string, but got empty string")
				}
				if !strings.Contains(query, tc.shouldContain) {
					t.Errorf("Query for %s '%s' did not contain expected substring '%s'.\nGot: %s",
						tc.nodeType, tc.nodeName, tc.shouldContain, query)
				}
			}
		})
	}
}
