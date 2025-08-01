package golang_test

import (
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/lang/golang"
)

// TestTemplateBasicFunctionality tests basic template functionality for all node types
func TestTemplateBasicFunctionality(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "function_template",
			dslQuery:      "func:main",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) @target",
			description:   "Function template should work correctly",
		},
		{
			name:          "function_template_wildcard",
			dslQuery:      "func:*",
			expectedQuery: "(function_declaration) @target",
			description:   "Function wildcard template should work correctly",
		},
		{
			name:          "const_template",
			dslQuery:      "const:PI",
			expectedQuery: "(const_declaration (const_spec name: (identifier_list (identifier) @name)) (#any-eq? @name \"PI\")) @target",
			description:   "Const template should work correctly",
		},
		{
			name:          "const_template_wildcard",
			dslQuery:      "const:*",
			expectedQuery: "(const_declaration) @target",
			description:   "Const wildcard template should work correctly",
		},
		{
			name:          "var_template",
			dslQuery:      "var:config",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")))] @target",
			description:   "Variable template should work correctly",
		},
		{
			name:          "var_template_wildcard",
			dslQuery:      "var:*",
			expectedQuery: "[(var_declaration (var_spec type: (type_identifier) @type) ) (var_declaration (var_spec_list (var_spec type: (type_identifier) @type) ))] @target",
			description:   "Variable wildcard template should work correctly",
		},
		{
			name:          "struct_template",
			dslQuery:      "struct:User",
			expectedQuery: "(type_declaration (type_spec name: (type_identifier) @name type: (struct_type)) (#eq? @name \"User\")) @target",
			description:   "Struct template should work correctly",
		},
		{
			name:          "struct_template_wildcard",
			dslQuery:      "struct:*",
			expectedQuery: "(type_declaration (type_spec type: (struct_type))) @target",
			description:   "Struct wildcard template should work correctly",
		},
		{
			name:          "field_template",
			dslQuery:      "field:Name",
			expectedQuery: "[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"Name\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"Name\"))] @target",
			description:   "Field template should work correctly",
		},
		{
			name:          "field_template_wildcard",
			dslQuery:      "field:*",
			expectedQuery: "(field_declaration type: (type_identifier) @type ) @target",
			description:   "Field wildcard template should work correctly",
		},
		{
			name:          "call_template",
			dslQuery:      "call:fmt.Println",
			expectedQuery: "(call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"fmt.Println\")) @target",
			description:   "Call template should work correctly",
		},
		{
			name:          "call_template_wildcard",
			dslQuery:      "call:*",
			expectedQuery: "(call_expression) @target",
			description:   "Call wildcard template should work correctly",
		},
		{
			name:          "assign_template",
			dslQuery:      "assign:result",
			expectedQuery: "[(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-eq? @name \"result\") @target",
			description:   "Assignment template should work correctly",
		},
		{
			name:          "assign_template_wildcard",
			dslQuery:      "assign:*",
			expectedQuery: "[(assignment_statement) (short_var_declaration)] @target",
			description:   "Assignment wildcard template should work correctly",
		},
		{
			name:          "import_template",
			dslQuery:      "import:fmt",
			expectedQuery: "(import_spec path: (interpreted_string_literal) @path (#eq? @path \"fmt\")) @target",
			description:   "Import template should work correctly",
		},
		{
			name:          "import_template_wildcard",
			dslQuery:      "import:*",
			expectedQuery: "(import_spec) @target",
			description:   "Import wildcard template should work correctly",
		},
		{
			name:          "if_template_wildcard",
			dslQuery:      "if:*",
			expectedQuery: "(if_statement) @target",
			description:   "If wildcard template should work correctly",
		},
		{
			name:          "block_template_wildcard",
			dslQuery:      "block:*",
			expectedQuery: "(block) @target",
			description:   "Block wildcard template should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for template test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Template test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestTemplateWithTypes tests template functionality with type specifications
func TestTemplateWithTypes(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "var_with_type",
			dslQuery:      "var:config Config",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\") (#eq? @type \"Config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\") (#eq? @type \"Config\")))] @target",
			description:   "Variable with type should work correctly",
		},
		{
			name:          "var_wildcard_with_type",
			dslQuery:      "var:* Config",
			expectedQuery: "[(var_declaration (var_spec type: (type_identifier) @type) (#eq? @type \"Config\")) (var_declaration (var_spec_list (var_spec type: (type_identifier) @type) (#eq? @type \"Config\")))] @target",
			description:   "Variable wildcard with type should work correctly",
		},
		{
			name:          "field_with_type",
			dslQuery:      "field:Name string",
			expectedQuery: "[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"Name\") (#eq? @type \"string\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"Name\") (#eq? @type \"string\"))] @target",
			description:   "Field with type should work correctly",
		},
		{
			name:          "field_wildcard_with_type",
			dslQuery:      "field:* string",
			expectedQuery: "(field_declaration type: (type_identifier) @type (#eq? @type \"string\")) @target",
			description:   "Field wildcard with type should work correctly",
		},
		{
			name:          "var_with_complex_type",
			dslQuery:      "var:client http.Client",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"client\") (#eq? @type \"http.Client\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"client\") (#eq? @type \"http.Client\")))] @target",
			description:   "Variable with complex type should work correctly",
		},
		{
			name:          "field_with_pointer_type",
			dslQuery:      "field:user *User",
			expectedQuery: "[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"user\") (#match? @type \".*User$\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"user\") (#match? @type \".*User$\"))] @target",
			description:   "Field with pointer type should work correctly",
		},
		{
			name:          "var_with_slice_type",
			dslQuery:      "var:items []string",
			expectedQuery: "[(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"items\") (#eq? @type \"[]string\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"items\") (#eq? @type \"[]string\")))] @target",
			description:   "Variable with slice type should work correctly",
		},
		{
			name:          "field_with_map_type",
			dslQuery:      "field:data map[string]interface{}",
			expectedQuery: "[(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"data\") (#eq? @type \"map[string]interface{}\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"data\") (#eq? @type \"map[string]interface{}\"))] @target",
			description:   "Field with map type should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for template with type test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Template with type test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}

// TestTemplateHierarchy tests template functionality in hierarchical queries
func TestTemplateHierarchy(t *testing.T) {
	p := golang.New()

	tests := []struct {
		name          string
		dslQuery      string
		expectedQuery string
		description   string
	}{
		{
			name:          "func_with_var_child",
			dslQuery:      "func:main > var:config",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\")))] @target",
			description:   "Function with variable child should work correctly",
		},
		{
			name:          "struct_with_field_child",
			dslQuery:      "struct:User > field:Name",
			expectedQuery: "(type_declaration (type_spec name: (type_identifier) @name type: (struct_type)) (#eq? @name \"User\")) . [(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"Name\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"Name\"))] @target",
			description:   "Struct with field child should work correctly",
		},
		{
			name:          "func_with_if_with_call",
			dslQuery:      "func:handler > if:* > call:log.Error",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"handler\")) . (if_statement) . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"log.Error\")) @target",
			description:   "Function with if and call hierarchy should work correctly",
		},
		{
			name:          "func_with_assign_with_call",
			dslQuery:      "func:process > assign:result > call:util.Process",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"process\")) . [(assignment_statement left: (expression_list [(identifier) @name])) (short_var_declaration left: (expression_list [(identifier) @name]))] (#any-eq? @name \"result\") . (call_expression function: [(identifier) (selector_expression)] @name (#eq? @name \"util.Process\")) @target",
			description:   "Function with assignment and call hierarchy should work correctly",
		},
		{
			name:          "wildcard_hierarchy",
			dslQuery:      "func:* > var:* > field:*",
			expectedQuery: "(function_declaration) . [(var_declaration (var_spec type: (type_identifier) @type) ) (var_declaration (var_spec_list (var_spec type: (type_identifier) @type) ))] . (field_declaration type: (type_identifier) @type ) @target",
			description:   "Wildcard hierarchy should work correctly",
		},
		{
			name:          "mixed_hierarchy_with_types",
			dslQuery:      "func:main > var:config Config > field:Host string",
			expectedQuery: "(function_declaration name: (identifier) @name (#eq? @name \"main\")) . [(var_declaration (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\") (#eq? @type \"Config\")) (var_declaration (var_spec_list (var_spec name: (identifier_list (identifier) @name) type: (type_identifier) @type) (#any-eq? @name \"config\") (#eq? @type \"Config\")))] . [(field_declaration name: (field_identifier_list (field_identifier) @name) type: (type_identifier) @type (#any-eq? @name \"Host\") (#eq? @type \"string\")) (field_declaration name: (field_identifier) @name type: (type_identifier) @type (#any-eq? @name \"Host\") (#eq? @type \"string\"))] @target",
			description:   "Mixed hierarchy with types should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, err := p.TranslateDSL(tt.dslQuery)
			if err != nil {
				t.Fatalf("Unexpected error for template hierarchy test %q: %v. %s", tt.dslQuery, err, tt.description)
			}

			// Normalize whitespace for comparison
			actualQuery = strings.Join(strings.Fields(actualQuery), " ")
			expectedQuery := strings.Join(strings.Fields(tt.expectedQuery), " ")

			if actualQuery != expectedQuery {
				t.Errorf("Template hierarchy test failed for query %q:\nExpected: %s\nActual:   %s\nDescription: %s",
					tt.dslQuery, expectedQuery, actualQuery, tt.description)
			}
		})
	}
}
