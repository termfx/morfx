package mcp

import "github.com/termfx/morfx/mcp/types"

func registerDefaultResourceTemplates(reg *ResourceTemplateRegistry) {
	if reg == nil {
		return
	}

	reg.Register("workspace-file", types.ResourceTemplateDefinition{
		Name:        "workspace-file",
		Title:       "Workspace File",
		Description: "Read a file from the active workspace by relative path.",
		URITemplate: "file://{path}",
		InputSchema: types.NormalizeSchema(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file, relative to the workspace root.",
				},
			},
			"required": []string{"path"},
		}),
		Annotations: map[string]any{
			"workspaceRelative": true,
		},
	})

	reg.Register("workspace-directory", types.ResourceTemplateDefinition{
		Name:        "workspace-directory",
		Title:       "Workspace Directory",
		Description: "List files within a workspace directory.",
		URITemplate: "morfx://workspace/{path}",
		InputSchema: types.NormalizeSchema(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Directory path relative to the workspace root.",
				},
				"recursive": map[string]any{
					"type":        "boolean",
					"description": "Include nested files when true.",
					"default":     false,
				},
				"maxEntries": map[string]any{
					"type":        "integer",
					"description": "Maximum number of entries to include.",
					"minimum":     1,
					"maximum":     500,
				},
			},
			"required": []string{"path"},
		}),
		Annotations: map[string]any{
			"workspaceRelative": true,
		},
	})
}
