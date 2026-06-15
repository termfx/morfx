package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/engine"
	"github.com/oxhq/morfx/internal/toolenv"
)

const fileQueryHelp = `Usage: file_query [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "scope": {
    "path": "<root directory>",
    "include": ["<glob>", ...],
    "exclude": ["<glob>", ...],
    "language": "<optional language override>",
    "max_files": <optional limit>
  },
  "query": {<optional core.AgentQuery payload>},
  "dsl": "<optional Morfx DSL selector, such as struct:* > field:Secret type=string>"
}
"path" must reference an accessible directory. Optional include/exclude filters
follow the same semantics as the MCP tool.

Output schema:
{
  "content": [{"type": "text", "text": "<summary>"}],
  "matches": <int>,
  "files":   <int number of unique files>,
  "results": [<core.FileMatch objects>]
}`

type fileQueryRequest struct {
	Scope *core.FileScope `json:"scope"`
	Query json.RawMessage `json:"query"`
	DSL   string          `json:"dsl,omitempty"`
}

type fileQueryRunner interface {
	FileQuery(context.Context, engine.FileQueryRequest) (engine.FileQueryResult, error)
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
		fmt.Print(fileQueryHelp)
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

	req, err := toolenv.ReadJSON[fileQueryRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	payload, err := runFileQuery(ctx, e, *req)
	if err != nil {
		writeCommandError(err)
		os.Exit(1)
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func runFileQuery(ctx context.Context, runner fileQueryRunner, req fileQueryRequest) (map[string]any, error) {
	if req.Scope == nil {
		return nil, commandError{message: "scope is required", cause: errors.New("missing scope")}
	}
	if strings.TrimSpace(req.Scope.Path) == "" {
		return nil, commandError{message: "scope.path is required", cause: errors.New("missing scope.path")}
	}

	absPath, err := filepath.Abs(req.Scope.Path)
	if err != nil {
		return nil, commandError{message: "invalid scope path", cause: err}
	}
	if _, err := os.Stat(absPath); err != nil {
		return nil, commandError{message: "scope path not accessible", cause: err}
	}

	if len(req.Query) == 0 && strings.TrimSpace(req.DSL) == "" {
		return nil, commandError{message: "query is required", cause: errors.New("missing query")}
	}

	query, err := core.ParseAgentQueryPayload(req.Query, req.DSL)
	if err != nil {
		return nil, commandError{message: "invalid query structure", cause: err}
	}

	scope := *req.Scope
	scope.Path = absPath
	result, err := runner.FileQuery(ctx, engine.FileQueryRequest{
		Scope: scope,
		Query: query,
		DSL:   req.DSL,
	})
	if err != nil {
		return nil, commandError{message: "file query failed", cause: err}
	}

	responseText := formatFileQueryResponse(result.Results, scope)
	payload := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"matches": len(result.Results),
		"files":   countUniqueFiles(result.Results),
		"results": result.Results,
	}

	return payload, nil
}

func writeCommandError(err error) {
	var cmdErr commandError
	if errors.As(err, &cmdErr) {
		_ = toolenv.WriteError(os.Stdout, cmdErr.message, cmdErr.cause)
		return
	}
	_ = toolenv.WriteError(os.Stdout, "file query failed", err)
}

func formatFileQueryResponse(matches []core.FileMatch, scope core.FileScope) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found in %s", scope.Path)
	}

	fileGroups := make(map[string][]core.FileMatch)
	for _, match := range matches {
		fileGroups[match.FilePath] = append(fileGroups[match.FilePath], match)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d matches across %d files in %s:\n\n", len(matches), len(fileGroups), scope.Path))

	for filePath, fileMatches := range fileGroups {
		builder.WriteString(fmt.Sprintf("📄 %s (%d matches):\n", filePath, len(fileMatches)))
		for _, match := range fileMatches {
			builder.WriteString(fmt.Sprintf("  • %s '%s' at line %d, column %d\n", match.Type, match.Name, match.Location.Line, match.Location.Column))
			if strings.TrimSpace(match.Content) != "" {
				snippet := strings.TrimSpace(match.Content)
				if len(snippet) > 80 {
					snippet = snippet[:77] + "..."
				}
				builder.WriteString("    " + snippet + "\n")
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

func countUniqueFiles(matches []core.FileMatch) int {
	seen := make(map[string]struct{})
	for _, match := range matches {
		seen[match.FilePath] = struct{}{}
	}
	return len(seen)
}
