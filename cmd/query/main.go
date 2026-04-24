package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/internal/toolenv"
)

const queryHelp = `Usage: query [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "language": "<language id>",
  "source":   "<optional source code>",
  "path":     "<optional file path>",
  "query":    {<optional core.AgentQuery payload>},
  "dsl":      "<optional Morfx DSL selector, such as func:* > call:os.Getenv>"
}
Exactly one of "source" or "path" must be provided. When "path" is used the
file is read from disk.

Output schema:
{
  "content": [{"type": "text", "text": "<human readable summary>"}],
  "matches": <int>,
  "results": [<core.Match objects>],
  "path": "<optional original path>"
}`

type queryRequest struct {
	Language string          `json:"language"`
	Source   *string         `json:"source,omitempty"`
	Path     *string         `json:"path,omitempty"`
	Query    json.RawMessage `json:"query"`
	DSL      string          `json:"dsl,omitempty"`
}

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Usage = func() {
		fmt.Print(queryHelp)
	}
	flag.Parse()
	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	env, err := toolenv.NewEnvironment()
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "failed to initialise environment", err)
		os.Exit(1)
	}

	req, err := toolenv.ReadJSON[queryRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	if strings.TrimSpace(req.Language) == "" {
		_ = toolenv.WriteError(os.Stdout, "language is required", errors.New("missing language"))
		os.Exit(1)
	}
	if len(req.Query) == 0 && strings.TrimSpace(req.DSL) == "" {
		_ = toolenv.WriteError(os.Stdout, "query is required", errors.New("missing query"))
		os.Exit(1)
	}

	src, err := toolenv.LoadSource(req.Source, req.Path)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "failed to resolve source", err)
		os.Exit(1)
	}

	provider, err := env.Provider(req.Language)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "language provider not available", err)
		os.Exit(1)
	}

	query, err := core.ParseAgentQueryPayload(req.Query, req.DSL)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid query structure", err)
		os.Exit(1)
	}

	result := provider.Query(src.Code, query)
	if result.Error != nil {
		_ = toolenv.WriteError(os.Stdout, "query execution failed", result.Error)
		os.Exit(1)
	}

	responseText := formatQueryResponse(result, src.Path)

	payload := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": responseText,
			},
		},
		"matches": result.Total,
		"results": result.Matches,
	}

	if src.FromFile {
		payload["path"] = src.Path
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func formatQueryResponse(result core.QueryResult, path string) string {
	if result.Total == 0 {
		if strings.TrimSpace(path) != "" {
			return fmt.Sprintf("File: %s\n\nNo matches found", path)
		}
		return "No matches found"
	}

	var builder strings.Builder
	if strings.TrimSpace(path) != "" {
		builder.WriteString(fmt.Sprintf("File: %s\n\n", path))
	}

	builder.WriteString(fmt.Sprintf("Found %d match", result.Total))
	if result.Total != 1 {
		builder.WriteString("es")
	}
	builder.WriteString(":\n\n")

	for _, match := range result.Matches {
		builder.WriteString(fmt.Sprintf("• %s '%s' at line %d, column %d\n", match.Type, match.Name, match.Location.Line, match.Location.Column))
		if strings.TrimSpace(match.Content) != "" {
			builder.WriteString(fmt.Sprintf("  Content: %s\n", strings.TrimSpace(match.Content)))
		}
	}

	return builder.String()
}
