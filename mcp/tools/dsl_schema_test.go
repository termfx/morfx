package tools

import (
	"strings"
	"testing"

	"github.com/oxhq/morfx/mcp/types"
)

func TestDSLFieldsAreAgentReadableInMCPSchemas(t *testing.T) {
	server := newMockServer()
	cases := []struct {
		name      string
		tool      types.Tool
		field     string
		fragments []string
	}{
		{
			name:  "query",
			tool:  NewQueryTool(server),
			field: "dsl",
			fragments: []string{
				"kind:name",
				"func:* > call:os.Getenv",
				"Operators",
				"attributes",
			},
		},
		{
			name:  "file_query",
			tool:  NewFileQueryTool(server),
			field: "dsl",
			fragments: []string{
				"kind:name",
				"struct:* > field:Secret type=string",
				"Operators",
				"attributes",
			},
		},
		{
			name:  "replace",
			tool:  NewReplaceTool(server),
			field: "target_dsl",
			fragments: []string{
				"target_dsl",
				"func:* > call:os.Getenv",
				"Use this instead of target",
			},
		},
		{
			name:  "recipe",
			tool:  NewRecipeTool(server),
			field: "target_dsl",
			fragments: []string{
				"target_dsl",
				"func:Legacy*",
				"kind:name",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			description := schemaDescription(t, tt.tool.InputSchema(), tt.field)
			for _, fragment := range tt.fragments {
				if !strings.Contains(description, fragment) {
					t.Fatalf("%s.%s description missing %q:\n%s", tt.name, tt.field, fragment, description)
				}
			}
		})
	}
}

func schemaDescription(t *testing.T, schema map[string]any, field string) string {
	t.Helper()

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema has no properties")
	}

	if field == "target_dsl" {
		if steps, ok := properties["steps"].(map[string]any); ok {
			items, ok := steps["items"].(map[string]any)
			if !ok {
				t.Fatalf("steps has no item schema")
			}
			properties, ok = items["properties"].(map[string]any)
			if !ok {
				t.Fatalf("step schema has no properties")
			}
		}
	}

	property, ok := properties[field].(map[string]any)
	if !ok {
		t.Fatalf("schema has no %s property", field)
	}
	description, ok := property["description"].(string)
	if !ok || description == "" {
		t.Fatalf("%s property has no description", field)
	}
	return description
}
