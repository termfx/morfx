package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestDSLRequiredPositiveMatches tests all required positive match cases from DSL v0.1.1 specification
func TestDSLRequiredPositiveMatches(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "REQUIRED_call_util_SHA1FileHex",
			dslQuery:      "call:util.SHA1FileHex",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"util.SHA1FileHex\")) @target",
			description:   "Must match call:util.SHA1FileHex",
		},
		{
			name:          "REQUIRED_assign_fileHash_call_util_SHA1FileHex",
			dslQuery:      "assign:fileHash > call:util.SHA1FileHex",
			expectedQuery: "[(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-eq? @name \"fileHash\") . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"util.SHA1FileHex\")) @target",
			description:   "Must match assign:fileHash > call:util.SHA1FileHex",
		},
		{
			name:          "REQUIRED_assign_estaMadre_call_elpack_UnaChingadera",
			dslQuery:      "assign:estaMadre > call:elpack.UnaChingadera",
			expectedQuery: "[(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-eq? @name \"estaMadre\") . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"elpack.UnaChingadera\")) @target",
			description:   "Must match assign:estaMadre > call:elpack.UnaChingadera",
		},
		{
			name:          "REQUIRED_func_wildcard_var_wildcard_core_ModelConfig",
			dslQuery:      "func:* > var:* core.ModelConfig",
			expectedQuery: "(function_declaration) . [(var_declaration (var_spec type: (type_identifier) @type) (#eq? @type \"core.ModelConfig\")) (var_declaration (var_spec_list (var_spec type: (type_identifier) @type) (#eq? @type \"core.ModelConfig\")))] @target",
			description:   "Must match func:* > var:* core.ModelConfig",
		},
		{
			name:          "REQUIRED_struct_wildcard_field_Secret_string",
			dslQuery:      "struct:* > field:Secret string",
			expectedQuery: "(type_declaration (type_spec type: (struct_type))) . [(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"Secret\") (#eq? @type \"string\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"Secret\") (#eq? @type \"string\"))] @target",
			description:   "Must match struct:* > field:Secret string",
		},
		{
			name:          "REQUIRED_import_fmt",
			dslQuery:      "import:fmt",
			expectedQuery: "(import_spec path: (interpreted_string_literal) @path (#eq? @path \"fmt\")) @target",
			description:   "Must match import:fmt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for required positive match %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Required positive match failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestDSLRequiredNegativeMatches tests all required negative match cases from DSL v0.1.1 specification
func TestDSLRequiredNegativeMatches(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name        string
		dslQuery    string
		description string
	}{
		{
			name:        "REQUIRED_NEGATIVE_if_Nombre",
			dslQuery:    "if:Nombre",
			description: "Must fail if:Nombre (only * allowed for if/block)",
		},
		{
			name:        "REQUIRED_NEGATIVE_block_Nombre",
			dslQuery:    "block:Nombre",
			description: "Must fail block:Nombre (only * allowed for if/block)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.TranslateDSL(tt.dslQuery)

			if err == nil {
				t.Fatalf("Expected error for required negative match %q, but got none. %s", tt.dslQuery, tt.description)
			}
		})
	}
}

// TestDSLRequiredCaseSensitivity tests case-sensitive matching requirements
func TestDSLRequiredCaseSensitivity(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "CASE_SENSITIVE_func_init_vs_Init",
			dslQuery:      "func:init",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"init\")) @target",
			description:   "func:init should match 'init' but not 'Init' (case-sensitive)",
		},
		{
			name:          "CASE_SENSITIVE_func_Init_vs_init",
			dslQuery:      "func:Init",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"Init\")) @target",
			description:   "func:Init should match 'Init' but not 'init' (case-sensitive)",
		},
		{
			name:          "CASE_SENSITIVE_func_TEST_vs_Test",
			dslQuery:      "func:TEST",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"TEST\")) @target",
			description:   "func:TEST should match 'TEST' but not 'Test' (case-sensitive)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for case sensitivity test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Case sensitivity test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}
