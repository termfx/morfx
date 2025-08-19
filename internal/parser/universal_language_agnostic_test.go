package parser

import (
	"testing"

	"github.com/termfx/morfx/internal/core"
)

// TestLanguageAgnosticAliases verifies that ALL common programming terms
// from different languages are properly mapped to universal kinds
func TestLanguageAgnosticAliases(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name         string
		dsl          string
		expectedKind core.NodeKind
		description  string
	}{
		// Function aliases - all major languages
		{"JavaScript function", "function:test", core.KindFunction, "JavaScript/TypeScript style"},
		{"Go func", "func:Test", core.KindFunction, "Go style"},
		{"Python def", "def:test_func", core.KindFunction, "Python style"},
		{"Rust fn", "fn:calculate", core.KindFunction, "Rust style"},
		{"Perl sub", "sub:handler", core.KindFunction, "Perl style"},
		{"Pascal procedure", "procedure:init", core.KindFunction, "Pascal style"},
		{"OOP method", "method:getName", core.KindMethod, "Object-oriented style"},

		// Variable aliases - different declaration styles
		{"Generic variable", "variable:count", core.KindVariable, "Generic term"},
		{"JavaScript var", "var:data", core.KindVariable, "JavaScript/Go style"},
		{"ES6 let", "let:result", core.KindVariable, "ES6/TypeScript/Swift style"},
		{"Constant const", "const:API_URL", core.KindConstant, "Constant declaration"},
		{"Java final", "final:VERSION", core.KindConstant, "Java final"},
		{"C# readonly", "readonly:config", core.KindConstant, "C# readonly"},
		{"Scala immutable", "immutable:state", core.KindConstant, "Immutable declaration"},

		// Class and type aliases - OOP and structural
		{"OOP class", "class:User", core.KindClass, "Object-oriented class"},
		{"Go struct", "struct:Config", core.KindClass, "Go/C/Rust struct"},
		{"Type definition", "type:Handler", core.KindType, "Type alias/definition"},
		{"Java interface", "interface:Serializable", core.KindInterface, "Java/Go/C# interface"},
		{"Swift protocol", "protocol:Drawable", core.KindInterface, "Swift/Objective-C protocol"},
		{"Rust trait", "trait:Clone", core.KindInterface, "Rust trait"},
		{"Enumeration", "enum:Status", core.KindEnum, "Enumeration"},
		{"Full enumeration", "enumeration:Color", core.KindEnum, "Verbose form"},

		// Import aliases - different module systems
		{"Python import", "import:os", core.KindImport, "Python/Java/JS import"},
		{"Node.js require", "require:fs", core.KindImport, "Node.js require"},
		{"C include", "include:stdio", core.KindImport, "C/C++/PHP include"},
		{"Rust use", "use:std", core.KindImport, "Rust/PHP/C# use"},
		{"C# using", "using:System", core.KindImport, "C#/C++ using"},
		{"Python from", "from:util", core.KindImport, "Python from import"},

		// Field and property aliases
		{"Generic field", "field:name", core.KindField, "Generic field"},
		{"C# property", "property:Value", core.KindField, "C#/Python property"},
		{"Python attribute", "attribute:_private", core.KindField, "Python attribute"},
		{"C++ member", "member:data", core.KindField, "C++/C# member"},
		{"Lisp slot", "slot:value", core.KindField, "Lisp slot"},

		// Function call aliases
		{"Generic call", "call:print", core.KindCall, "Function call"},
		{"Java invoke", "invoke:method", core.KindCall, "Java/C# invoke"},
		{"Functional apply", "apply:transform", core.KindCall, "Functional apply"},
		{"Generic execute", "execute:command", core.KindCall, "Execute action"},

		// Assignment aliases
		{"Assignment", "assignment:x", core.KindAssignment, "Variable assignment"},
		{"Short assign", "assign:result", core.KindAssignment, "Shortened form"},
		{"Setter", "set:value", core.KindAssignment, "Setter context"},

		// Control flow aliases
		{"Condition", "condition:check", core.KindCondition, "Generic condition"},
		{"If statement", "if:valid", core.KindCondition, "If statement"},
		{"Switch", "switch:type", core.KindCondition, "Switch statement"},
		{"Case", "case:default", core.KindCondition, "Switch case"},
		{"Ruby when", "when:ready", core.KindCondition, "Ruby when"},
		{"Rust match", "match:pattern", core.KindCondition, "Rust pattern matching"},

		// Loop aliases
		{"Generic loop", "loop:main", core.KindLoop, "Generic loop"},
		{"For loop", "for:i", core.KindLoop, "For loop"},
		{"While loop", "while:condition", core.KindLoop, "While loop"},
		{"Do loop", "do:action", core.KindLoop, "Do-while loop"},
		{"C# foreach", "foreach:item", core.KindLoop, "C#/PHP foreach"},
		{"Pascal repeat", "repeat:until", core.KindLoop, "Pascal repeat"},

		// Block and scope aliases
		{"Code block", "block:main", core.KindBlock, "Code block"},
		{"Scope", "scope:local", core.KindBlock, "Scope block"},
		{"Pascal begin", "begin:proc", core.KindBlock, "Pascal begin"},
		{"Block end", "end:if", core.KindBlock, "Block terminator"},

		// Comment aliases
		{"Comment", "comment:TODO", core.KindComment, "Code comment"},
		{"Documentation", "doc:API", core.KindComment, "Documentation"},
		{"Full docs", "documentation:guide", core.KindComment, "Full documentation"},

		// Decorator/annotation aliases
		{"Python decorator", "decorator:property", core.KindDecorator, "Python decorator"},
		{"Java annotation", "annotation:Override", core.KindDecorator, "Java annotation"},

		// Exception handling aliases
		{"Try block", "try:operation", core.KindTryCatch, "Try block"},
		{"Catch block", "catch:error", core.KindTryCatch, "Catch block"},
		{"Python except", "except:ValueError", core.KindTryCatch, "Python except"},
		{"Ruby rescue", "rescue:StandardError", core.KindTryCatch, "Ruby rescue"},
		{"Finally block", "finally:cleanup", core.KindTryCatch, "Finally block"},

		// Return and throw aliases
		{"Return statement", "return:result", core.KindReturn, "Return statement"},
		{"Generator yield", "yield:value", core.KindReturn, "Generator yield"},
		{"JavaScript throw", "throw:error", core.KindThrow, "JavaScript throw"},
		{"Python raise", "raise:Exception", core.KindThrow, "Python raise"},
		{"Go panic", "panic:fatal", core.KindThrow, "Go panic"},

		// Parameter aliases
		{"Parameter", "parameter:input", core.KindParameter, "Function parameter"},
		{"Short param", "param:data", core.KindParameter, "Shortened parameter"},
		{"Argument", "argument:value", core.KindParameter, "Function argument"},
		{"Short arg", "arg:index", core.KindParameter, "Shortened argument"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parser.ParseQuery(tt.dsl)
			if err != nil {
				t.Fatalf("ParseQuery(%s) error = %v, want nil", tt.dsl, err)
			}

			if query.Kind != tt.expectedKind {
				t.Errorf("ParseQuery(%s).Kind = %v, want %v", tt.dsl, query.Kind, tt.expectedKind)
			}

			// Verify raw DSL is preserved
			if query.Raw != tt.dsl {
				t.Errorf("ParseQuery(%s).Raw = %v, want %v", tt.dsl, query.Raw, tt.dsl)
			}
		})
	}
}

// TestOperatorNormalization verifies that all operator variations are normalized correctly
func TestOperatorNormalization(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name             string
		dsl              string
		expectedOperator string
		description      string
	}{
		// AND operator variations
		{"Single ampersand (primary)", "func:test & var:data", "AND", "Primary single-character operator"},
		{"Double ampersand", "func:test && var:data", "AND", "C-family style"},
		{"Word and", "func:test and var:data", "AND", "English-like style"},
		{"Upper AND", "func:test AND var:data", "AND", "Already normalized"},

		// OR operator variations
		{"Single pipe (primary)", "func:test | var:data", "OR", "Primary single-character operator"},
		{"Double pipe", "func:test || var:data", "OR", "C-family style"},
		{"Word or", "func:test or var:data", "OR", "English-like style"},
		{"Upper OR", "func:test OR var:data", "OR", "Already normalized"},

		// NOT operator variations
		{"Exclamation (primary)", "!func:test", "NOT", "Primary single-character operator"},
		{"Word not", "not func:test", "NOT", "English-like style"},

		// HIERARCHY operator
		{"Greater than", "class:User > method:getName", "HIERARCHY", "Parent > child relationship"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parser.ParseQuery(tt.dsl)
			if err != nil {
				t.Fatalf("ParseQuery(%s) error = %v, want nil", tt.dsl, err)
			}

			if query.Operator != tt.expectedOperator {
				t.Errorf("ParseQuery(%s).Operator = %v, want %v", tt.dsl, query.Operator, tt.expectedOperator)
			}
		})
	}
}

// TestCrossLanguageCompatibility verifies that the same logical query works
// across different programming language syntaxes
func TestCrossLanguageCompatibility(t *testing.T) {
	parser := NewUniversalParser()

	// Same logical intent expressed in different language styles
	tests := []struct {
		name         string
		queries      []string
		expectedKind core.NodeKind
		description  string
	}{
		{
			name: "Function definitions across languages",
			queries: []string{
				"function:test*",  // JavaScript/TypeScript
				"func:Test*",      // Go
				"def:test_*",      // Python
				"fn:test_*",       // Rust
				"sub:test*",       // Perl
				"procedure:test*", // Pascal
			},
			expectedKind: core.KindFunction,
			description:  "Function definitions should map to same universal kind",
		},
		{
			name: "Variable declarations across languages",
			queries: []string{
				"variable:data", // Generic
				"var:data",      // JavaScript/Go
				"let:data",      // ES6/TypeScript
			},
			expectedKind: core.KindVariable,
			description:  "Variable declarations should map to same universal kind",
		},
		{
			name: "Class/struct definitions across languages",
			queries: []string{
				"class:User",  // OOP languages
				"struct:User", // Go/C/Rust
			},
			expectedKind: core.KindClass,
			description:  "Class/struct definitions should map to same universal kind",
		},
		{
			name: "Import statements across languages",
			queries: []string{
				"import:module",   // Python/Java/JavaScript
				"require:module",  // Node.js
				"include:header",  // C/C++
				"use:crate",       // Rust
				"using:namespace", // C#
			},
			expectedKind: core.KindImport,
			description:  "Import statements should map to same universal kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, queryDSL := range tt.queries {
				query, err := parser.ParseQuery(queryDSL)
				if err != nil {
					t.Fatalf("ParseQuery(%s) error = %v, want nil", queryDSL, err)
				}

				if query.Kind != tt.expectedKind {
					t.Errorf("ParseQuery(%s).Kind = %v, want %v (description: %s)",
						queryDSL, query.Kind, tt.expectedKind, tt.description)
				}
			}
		})
	}
}

// TestComplexQueries verifies that complex multi-language queries work correctly
func TestComplexQueries(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name        string
		dsl         string
		expectError bool
		description string
	}{
		{
			name:        "Python style with Go operators",
			dsl:         "def:test* & !struct:mock",
			expectError: false,
			description: "Python function syntax with Go struct and operators",
		},
		{
			name:        "JavaScript with English operators",
			dsl:         "function:test* and not const:DEBUG",
			expectError: false,
			description: "JavaScript syntax with English-like operators",
		},
		{
			name:        "Mixed language hierarchical query",
			dsl:         "class:Controller > def:handle*",
			expectError: false,
			description: "OOP class containing Python-style method",
		},
		{
			name:        "Rust with double operators",
			dsl:         "fn:process* && !struct:Config",
			expectError: false,
			description: "Rust syntax with C-style operators",
		},
		{
			name:        "Complex mixed query",
			dsl:         "(def:parse* || fn:compile*) & !import:deprecated",
			expectError: true, // Parentheses not yet supported
			description: "Complex query with mixed language terms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := parser.ParseQuery(tt.dsl)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseQuery(%s) expected error but got none", tt.dsl)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseQuery(%s) error = %v, want nil", tt.dsl, err)
			}

			if query == nil {
				t.Errorf("ParseQuery(%s) returned nil query", tt.dsl)
			}
		})
	}
}

// TestErrorHandling verifies proper error messages for unsupported constructs
func TestErrorHandling(t *testing.T) {
	parser := NewUniversalParser()

	tests := []struct {
		name          string
		dsl           string
		expectedError string
		description   string
	}{
		{
			name:          "Unsupported kind",
			dsl:           "nonsense:test",
			expectedError: "unsupported node kind",
			description:   "Should reject unknown kind aliases",
		},
		{
			name:          "Invalid format",
			dsl:           "function_without_colon",
			expectedError: "invalid query format",
			description:   "Should require colon separator",
		},
		{
			name:          "Empty query",
			dsl:           "",
			expectedError: "empty query string",
			description:   "Should reject empty queries",
		},
		{
			name:          "Invalid hierarchical format",
			dsl:           "func:test > > method:handle",
			expectedError: "invalid hierarchical query format",
			description:   "Should reject malformed hierarchical queries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseQuery(tt.dsl)
			if err == nil {
				t.Errorf("ParseQuery(%s) expected error containing '%s' but got nil",
					tt.dsl, tt.expectedError)
				return
			}

			if !contains(err.Error(), tt.expectedError) {
				t.Errorf("ParseQuery(%s) error = %v, want error containing '%s'",
					tt.dsl, err, tt.expectedError)
			}
		})
	}
}

// TestLanguageAgnosticOutput verifies that parser output contains no language-specific references
func TestLanguageAgnosticOutput(t *testing.T) {
	parser := NewUniversalParser()

	// Test various language-specific inputs
	languageSpecificInputs := []string{
		"def:python_function",   // Python
		"func:GoFunction",       // Go
		"function:jsFunction",   // JavaScript
		"fn:rust_function",      // Rust
		"sub:perl_sub",          // Perl
		"procedure:pascal_proc", // Pascal
	}

	for _, input := range languageSpecificInputs {
		t.Run(input, func(t *testing.T) {
			query, err := parser.ParseQuery(input)
			if err != nil {
				t.Fatalf("ParseQuery(%s) error = %v, want nil", input, err)
			}

			// Verify output only contains universal kinds
			if !isUniversalKind(query.Kind) {
				t.Errorf("ParseQuery(%s) produced non-universal kind: %v", input, query.Kind)
			}

			// Verify no language-specific terms in the query structure
			// (The pattern can contain language-specific names, that's fine)
			if query.Operator != "" && !isUniversalOperator(query.Operator) {
				t.Errorf("ParseQuery(%s) produced non-universal operator: %v", input, query.Operator)
			}
		})
	}
}

// TestOperatorEfficiency verifies that single-character operators are recognized as primary
func TestOperatorEfficiency(t *testing.T) {
	parser := NewUniversalParser()

	// Single-character operators should be the primary/fastest path
	singleCharTests := []struct {
		operator string
		dsl      string
		expected string
	}{
		{"&", "func:test & var:data", "AND"},
		{"|", "func:test | var:data", "OR"},
		{"!", "!func:test", "NOT"},
		{">", "class:User > method:get", "HIERARCHY"},
	}

	for _, tt := range singleCharTests {
		t.Run("Single char "+tt.operator, func(t *testing.T) {
			query, err := parser.ParseQuery(tt.dsl)
			if err != nil {
				t.Fatalf("ParseQuery(%s) error = %v, want nil", tt.dsl, err)
			}

			if query.Operator != tt.expected {
				t.Errorf("ParseQuery(%s).Operator = %v, want %v",
					tt.dsl, query.Operator, tt.expected)
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 || s[len(s)-len(substr):] == substr ||
			s[:len(substr)] == substr ||
			len(s) > len(substr) && s[1:len(substr)+1] == substr)
}

func isUniversalKind(kind core.NodeKind) bool {
	universalKinds := map[core.NodeKind]bool{
		core.KindFunction: true, core.KindVariable: true, core.KindClass: true,
		core.KindMethod: true, core.KindImport: true, core.KindConstant: true,
		core.KindField: true, core.KindCall: true, core.KindAssignment: true,
		core.KindCondition: true, core.KindLoop: true, core.KindBlock: true,
		core.KindComment: true, core.KindDecorator: true, core.KindType: true,
		core.KindInterface: true, core.KindEnum: true, core.KindParameter: true,
		core.KindReturn: true, core.KindThrow: true, core.KindTryCatch: true,
		"logical": true, // Special case for logical operations
	}
	return universalKinds[kind]
}

func isUniversalOperator(op string) bool {
	universalOps := map[string]bool{
		"AND": true, "OR": true, "NOT": true, "HIERARCHY": true,
	}
	return universalOps[op]
}
