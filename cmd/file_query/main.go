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
  "query": {<core.AgentQuery payload>}
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

	env, err := toolenv.NewEnvironment()
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "failed to initialise environment", err)
		os.Exit(1)
	}

	req, err := toolenv.ReadJSON[fileQueryRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	if req.Scope == nil {
		_ = toolenv.WriteError(os.Stdout, "scope is required", errors.New("missing scope"))
		os.Exit(1)
	}
	if strings.TrimSpace(req.Scope.Path) == "" {
		_ = toolenv.WriteError(os.Stdout, "scope.path is required", errors.New("missing scope.path"))
		os.Exit(1)
	}

	absPath, err := filepath.Abs(req.Scope.Path)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid scope path", err)
		os.Exit(1)
	}
	if _, err := os.Stat(absPath); err != nil {
		_ = toolenv.WriteError(os.Stdout, "scope path not accessible", err)
		os.Exit(1)
	}
	req.Scope.Path = absPath

	if len(req.Query) == 0 {
		_ = toolenv.WriteError(os.Stdout, "query is required", errors.New("missing query"))
		os.Exit(1)
	}

	var query core.AgentQuery
	if err := json.Unmarshal(req.Query, &query); err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid query structure", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	processor := env.FileProcessor()
	matches, err := processor.QueryFiles(ctx, *req.Scope, query)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "file query failed", err)
		os.Exit(1)
	}

	responseText := formatFileQueryResponse(matches, *req.Scope)

	payload := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"matches": len(matches),
		"files":   countUniqueFiles(matches),
		"results": matches,
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
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
