package mcp

import (
	"fmt"
	"strings"

	prompts "github.com/termfx/morfx/mcp/prompts"
	"github.com/termfx/morfx/mcp/types"
)

// PromptContent represents the content of a prompt response
type PromptContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// PromptMessage represents a prompt message
type PromptMessage struct {
	Role    string          `json:"role"`
	Content []PromptContent `json:"content"`
}

// GetPromptDefinitions returns all available prompt definitions

func GetPromptDefinitions() []types.PromptDefinition {
	definitions := []types.PromptDefinition{
		{
			Name:        "code-analysis",
			Title:       "Code Analysis",
			Description: "Analyze code structure and suggest transformations",
			Arguments: []types.PromptArgument{
				{Name: "language", Description: "Programming language of the code", Required: true},
				{Name: "code", Description: "Source code to analyze", Required: true},
				{Name: "focus", Description: "Optional focus area (functions, structs, methods, etc.)"},
			},
			Annotations: map[string]any{
				"category": "analysis",
				"audience": "developer",
			},
		},
		{
			Name:        "transformation-guide",
			Title:       "Transformation Guide",
			Description: "Generate step-by-step guide for code transformations",
			Arguments: []types.PromptArgument{
				{Name: "operation", Description: "Type of transformation (replace, delete, insert, etc.)", Required: true},
				{Name: "target", Description: "What to transform (function name, struct name, etc.)", Required: true},
				{Name: "language", Description: "Programming language"},
			},
			Annotations: map[string]any{
				"category": "guidance",
				"audience": "developer",
			},
		},
		{
			Name:        "confidence-explanation",
			Title:       "Confidence Explanation",
			Description: "Explain confidence scores and factors for transformations",
			Arguments: []types.PromptArgument{
				{Name: "score", Description: "Confidence score (0.0-1.0)", Required: true},
				{Name: "factors", Description: "JSON string of confidence factors"},
			},
			Annotations: map[string]any{
				"category": "insight",
				"audience": "developer",
			},
		},
		{
			Name:        "query-builder",
			Title:       "Query Builder",
			Description: "Help build queries for finding code elements",
			Arguments: []types.PromptArgument{
				{Name: "description", Description: "Natural language description of what to find", Required: true},
				{Name: "language", Description: "Programming language"},
			},
			Annotations: map[string]any{
				"category": "analysis",
				"audience": "developer",
			},
		},
		{
			Name:        "best-practices",
			Title:       "Best Practices",
			Description: "Provide best practices and recommendations for code transformations",
			Arguments: []types.PromptArgument{
				{Name: "language", Description: "Programming language", Required: true},
				{Name: "operation", Description: "Type of operation being performed"},
			},
			Annotations: map[string]any{
				"category": "guidance",
				"audience": "developer",
			},
		},
	}

	seen := make(map[string]struct{}, len(definitions))
	for _, def := range definitions {
		seen[def.Name] = struct{}{}
	}

	for _, prompt := range prompts.Registry.List() {
		name := prompt.Name()
		if _, exists := seen[name]; exists {
			continue
		}

		definitions = append(definitions, types.PromptDefinition{
			Name:        name,
			Title:       strings.Title(strings.ReplaceAll(name, "_", " ")),
			Description: prompt.Description(),
			Arguments:   prompt.Arguments(),
			Annotations: map[string]any{
				"category": "custom",
				"audience": "developer",
			},
		})

		seen[name] = struct{}{}
	}

	return definitions
}

// generatePromptContent creates the actual content for a prompt
func (s *StdioServer) generatePromptContent(name string, args map[string]string) ([]PromptMessage, error) {
	switch name {
	case "code-analysis":
		return s.generateCodeAnalysisPrompt(args)
	case "transformation-guide":
		return s.generateTransformationGuidePrompt(args)
	case "confidence-explanation":
		return s.generateConfidenceExplanationPrompt(args)
	case "query-builder":
		return s.generateQueryBuilderPrompt(args)
	case "best-practices":
		return s.generateBestPracticesPrompt(args)
	default:
		return nil, NewMCPError(MethodNotFound, "Prompt not found", map[string]any{
			"name": name,
		})
	}
}

// generateCodeAnalysisPrompt creates a code analysis prompt
func (s *StdioServer) generateCodeAnalysisPrompt(args map[string]string) ([]PromptMessage, error) {
	language := args["language"]
	code := args["code"]
	focus := args["focus"]

	if language == "" || code == "" {
		return nil, NewMCPError(InvalidParams, "Missing required arguments: language and code", nil)
	}

	var analysisText strings.Builder
	analysisText.WriteString(
		fmt.Sprintf("I need you to analyze this %s code and suggest potential transformations:\n\n", language),
	)
	analysisText.WriteString("```" + language + "\n")
	analysisText.WriteString(code)
	analysisText.WriteString("\n```\n\n")

	if focus != "" {
		analysisText.WriteString(fmt.Sprintf("Please focus specifically on: %s\n\n", focus))
	}

	analysisText.WriteString("Provide:\n")
	analysisText.WriteString("1. Code structure overview\n")
	analysisText.WriteString("2. Potential improvements or refactoring opportunities\n")
	analysisText.WriteString("3. Specific transformation suggestions using Morfx operations\n")
	analysisText.WriteString("4. Any code quality or best practice recommendations\n")

	return []PromptMessage{
		{
			Role: "user",
			Content: []PromptContent{
				{
					Type: "text",
					Text: analysisText.String(),
				},
			},
		},
	}, nil
}

// generateTransformationGuidePrompt creates a transformation guide prompt
func (s *StdioServer) generateTransformationGuidePrompt(args map[string]string) ([]PromptMessage, error) {
	operation := args["operation"]
	target := args["target"]
	language := args["language"]

	if operation == "" || target == "" {
		return nil, NewMCPError(InvalidParams, "Missing required arguments: operation and target", nil)
	}

	var guideText strings.Builder
	guideText.WriteString(
		fmt.Sprintf("I need a step-by-step guide for performing a '%s' operation on '%s'", operation, target),
	)

	if language != "" {
		guideText.WriteString(fmt.Sprintf(" in %s", language))
	}

	guideText.WriteString(".\n\n")
	guideText.WriteString("Please provide:\n")
	guideText.WriteString("1. Overview of the transformation\n")
	guideText.WriteString("2. Step-by-step instructions\n")
	guideText.WriteString("3. Morfx tool commands to use\n")
	guideText.WriteString("4. Example queries and parameters\n")
	guideText.WriteString("5. Potential risks and considerations\n")
	guideText.WriteString("6. How to verify the transformation was successful\n")

	return []PromptMessage{
		{
			Role: "user",
			Content: []PromptContent{
				{
					Type: "text",
					Text: guideText.String(),
				},
			},
		},
	}, nil
}

// generateConfidenceExplanationPrompt creates a confidence explanation prompt
func (s *StdioServer) generateConfidenceExplanationPrompt(args map[string]string) ([]PromptMessage, error) {
	score := args["score"]
	factors := args["factors"]

	if score == "" {
		return nil, NewMCPError(InvalidParams, "Missing required argument: score", nil)
	}

	var explanationText strings.Builder
	explanationText.WriteString(fmt.Sprintf("Please explain this confidence score: %s\n\n", score))

	if factors != "" {
		explanationText.WriteString("Confidence factors:\n")
		explanationText.WriteString(factors)
		explanationText.WriteString("\n\n")
	}

	explanationText.WriteString("Please provide:\n")
	explanationText.WriteString("1. What this confidence score means\n")
	explanationText.WriteString("2. Interpretation of the confidence level (high/medium/low)\n")
	explanationText.WriteString("3. Explanation of each factor and its impact\n")
	explanationText.WriteString("4. Recommendations for how to proceed\n")
	explanationText.WriteString("5. Ways to potentially improve the confidence score\n")

	return []PromptMessage{
		{
			Role: "user",
			Content: []PromptContent{
				{
					Type: "text",
					Text: explanationText.String(),
				},
			},
		},
	}, nil
}

// generateQueryBuilderPrompt creates a query builder prompt
func (s *StdioServer) generateQueryBuilderPrompt(args map[string]string) ([]PromptMessage, error) {
	description := args["description"]
	language := args["language"]

	if description == "" {
		return nil, NewMCPError(InvalidParams, "Missing required argument: description", nil)
	}

	var queryText strings.Builder
	queryText.WriteString("I need help building a Morfx query for this request:\n\n")
	queryText.WriteString(fmt.Sprintf("Description: %s\n\n", description))

	if language != "" {
		queryText.WriteString(fmt.Sprintf("Language: %s\n\n", language))
	}

	queryText.WriteString("Please provide:\n")
	queryText.WriteString("1. The appropriate Morfx query structure\n")
	queryText.WriteString("2. JSON query object to use\n")
	queryText.WriteString("3. Alternative query approaches if applicable\n")
	queryText.WriteString("4. Expected results and what to look for\n")
	queryText.WriteString("5. Tips for refining the query if needed\n\n")

	queryText.WriteString("Format the query as a JSON object that can be used with Morfx tools.")

	return []PromptMessage{
		{
			Role: "user",
			Content: []PromptContent{
				{
					Type: "text",
					Text: queryText.String(),
				},
			},
		},
	}, nil
}

// generateBestPracticesPrompt creates a best practices prompt
func (s *StdioServer) generateBestPracticesPrompt(args map[string]string) ([]PromptMessage, error) {
	language := args["language"]
	operation := args["operation"]

	if language == "" {
		return nil, NewMCPError(InvalidParams, "Missing required argument: language", nil)
	}

	var practicesText strings.Builder
	practicesText.WriteString(
		fmt.Sprintf("Please provide best practices and recommendations for code transformations in %s", language),
	)

	if operation != "" {
		practicesText.WriteString(fmt.Sprintf(", specifically for '%s' operations", operation))
	}

	practicesText.WriteString(".\n\n")
	practicesText.WriteString("Please cover:\n")
	practicesText.WriteString("1. General transformation best practices\n")
	practicesText.WriteString("2. Language-specific considerations\n")
	practicesText.WriteString("3. Common pitfalls to avoid\n")
	practicesText.WriteString("4. Testing and validation strategies\n")
	practicesText.WriteString("5. When to use staging vs auto-apply\n")
	practicesText.WriteString("6. Backup and rollback strategies\n")
	practicesText.WriteString("7. Performance considerations for large codebases\n")

	return []PromptMessage{
		{
			Role: "user",
			Content: []PromptContent{
				{
					Type: "text",
					Text: practicesText.String(),
				},
			},
		},
	}, nil
}
