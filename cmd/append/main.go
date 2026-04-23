package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/internal/toolcmd"
	"github.com/oxhq/morfx/internal/toolenv"
)

const appendHelp = `Usage: append [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "language": "<language id>",
  "source":   "<optional source code>",
  "path":     "<optional file path>",
  "target":   {<optional core.AgentQuery payload>},
  "content":  "<text to append>"
}
Exactly one of "source" or "path" must be provided. When "path" is set the
file will be updated in place if the operation succeeds. "target" is optional;
when omitted the provider chooses a sensible append location.

Output schema:
{
  "content":   [{"type": "text", "text": "<summary>"}],
  "matches":   <int>,
  "diff":      "<unified diff>",
  "confidence": {<core.ConfidenceScore>},
  "modified":  "<modified source>",
  "path":      "<optional original path>",
  "applied":   <bool indicating file write>
}`

type appendRequest struct {
	Language string          `json:"language"`
	Source   *string         `json:"source,omitempty"`
	Path     *string         `json:"path,omitempty"`
	Target   json.RawMessage `json:"target,omitempty"`
	Content  *string         `json:"content"`
}

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Usage = func() {
		fmt.Print(appendHelp)
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

	req, err := toolenv.ReadJSON[appendRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	if strings.TrimSpace(req.Language) == "" {
		_ = toolenv.WriteError(os.Stdout, "language is required", errors.New("missing language"))
		os.Exit(1)
	}
	if req.Content == nil {
		_ = toolenv.WriteError(os.Stdout, "content is required", errors.New("missing content"))
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

	op := core.TransformOp{
		Method:  "append",
		Content: *req.Content,
	}

	if len(req.Target) > 0 && strings.TrimSpace(string(req.Target)) != "null" {
		var target core.AgentQuery
		if err := json.Unmarshal(req.Target, &target); err != nil {
			_ = toolenv.WriteError(os.Stdout, "invalid target structure", err)
			os.Exit(1)
		}
		op.Target = target
	}

	result := provider.Transform(src.Code, op)
	if result.Error != nil {
		_ = toolenv.WriteError(os.Stdout, "append operation failed", result.Error)
		os.Exit(1)
	}

	wroteFile, err := toolcmd.WriteModifiedSource(src.Path, src.FromFile, src.Code, result.Modified, src.Perm)
	if err != nil {
		if writeErr := toolenv.WriteError(os.Stdout, "failed to write modified file", err); writeErr != nil {
			fmt.Fprintf(os.Stderr, "failed to write error output: %v\n", writeErr)
		}
		os.Exit(1)
	}

	responseText := formatAppendResponse(result, src.Path, src.FromFile, wroteFile)

	payload := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"matches":    result.MatchCount,
		"diff":       result.Diff,
		"confidence": result.Confidence,
		"modified":   result.Modified,
	}

	if src.FromFile {
		payload["path"] = src.Path
		payload["applied"] = wroteFile
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func formatAppendResponse(result core.TransformResult, path string, fromFile bool, applied bool) string {
	var builder strings.Builder
	builder.WriteString("✅ Append operation completed successfully\n\n")

	if fromFile {
		builder.WriteString(fmt.Sprintf("📄 File: %s\n", path))
		if applied {
			builder.WriteString("Changes written to disk.\n\n")
		} else {
			builder.WriteString("Preview only; file not modified.\n\n")
		}
	}

	builder.WriteString(fmt.Sprintf("Content appended to %d location(s)\n", result.MatchCount))
	builder.WriteString("\nConfidence: ")
	builder.WriteString(toolcmd.FormatConfidence(result.Confidence.Score))
	builder.WriteString(fmt.Sprintf(" (%.1f%%)", result.Confidence.Score*100))

	return builder.String()
}
