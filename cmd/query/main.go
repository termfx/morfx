package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
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

type queryRunner interface {
	Query(context.Context, engine.QueryRequest) (engine.QueryResult, error)
}

type commandError struct {
	message string
	cause   error
}

func (e commandError) Error() string {
	if e.cause == nil {
		return e.message
	}
	return e.message + ": " + e.cause.Error()
}

func (e commandError) Unwrap() error {
	return e.cause
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

	e, err := engine.New(engine.Config{})
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "failed to initialise engine", err)
		os.Exit(1)
	}

	req, err := toolenv.ReadJSON[queryRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	payload, err := runQuery(context.Background(), e, *req)
	if err != nil {
		writeCommandError(err)
		os.Exit(1)
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func runQuery(ctx context.Context, runner queryRunner, req queryRequest) (map[string]any, error) {
	if strings.TrimSpace(req.Language) == "" {
		return nil, commandError{message: "language is required", cause: errors.New("missing language")}
	}
	if len(req.Query) == 0 && strings.TrimSpace(req.DSL) == "" {
		return nil, commandError{message: "query is required", cause: errors.New("missing query")}
	}

	src, err := toolenv.LoadSource(req.Source, req.Path)
	if err != nil {
		return nil, commandError{message: "failed to resolve source", cause: err}
	}

	query, err := core.ParseAgentQueryPayload(req.Query, req.DSL)
	if err != nil {
		return nil, commandError{message: "invalid query structure", cause: err}
	}

	result, err := runner.Query(ctx, engine.QueryRequest{
		Language: req.Language,
		Source:   src.Code,
		Query:    query,
		DSL:      req.DSL,
	})
	if err != nil {
		return nil, commandError{message: "query execution failed", cause: err}
	}

	formatted := core.QueryResult{
		Total:   result.Matches,
		Matches: result.Results,
	}
	responseText := formatQueryResponse(formatted, src.Path)

	payload := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": responseText,
			},
		},
		"matches": result.Matches,
		"results": result.Results,
	}

	if src.FromFile {
		payload["path"] = src.Path
	}

	return payload, nil
}

func writeCommandError(err error) {
	var cmdErr commandError
	if errors.As(err, &cmdErr) {
		_ = toolenv.WriteError(os.Stdout, cmdErr.message, cmdErr.cause)
		return
	}
	_ = toolenv.WriteError(os.Stdout, "query execution failed", err)
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
