package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/termfx/morfx/core"
	"github.com/termfx/morfx/internal/toolenv"
)

const replaceHelp = `Usage: replace [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "language": "<language id>",
  "source":   "<optional source code>",
  "path":     "<optional file path>",
  "target":   {<core.AgentQuery payload>},
  "replacement": "<replacement text>"
}
Exactly one of "source" or "path" must be provided. When "path" is set the
file will be read and modified in place.

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

type replaceRequest struct {
	Language    string          `json:"language"`
	Source      *string         `json:"source,omitempty"`
	Path        *string         `json:"path,omitempty"`
	Target      json.RawMessage `json:"target"`
	Replacement string          `json:"replacement"`
}

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Usage = func() {
		fmt.Print(replaceHelp)
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

	req, err := toolenv.ReadJSON[replaceRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	if strings.TrimSpace(req.Language) == "" {
		_ = toolenv.WriteError(os.Stdout, "language is required", errors.New("missing language"))
		os.Exit(1)
	}
	if len(req.Target) == 0 {
		_ = toolenv.WriteError(os.Stdout, "target is required", errors.New("missing target"))
		os.Exit(1)
	}
	if strings.TrimSpace(req.Replacement) == "" {
		_ = toolenv.WriteError(os.Stdout, "replacement is required", errors.New("missing replacement"))
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

	var target core.AgentQuery
	if err := json.Unmarshal(req.Target, &target); err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid target structure", err)
		os.Exit(1)
	}

	op := core.TransformOp{
		Method:      "replace",
		Target:      target,
		Replacement: req.Replacement,
	}

	result := provider.Transform(src.Code, op)
	if result.Error != nil {
		_ = toolenv.WriteError(os.Stdout, "replace operation failed", result.Error)
		os.Exit(1)
	}

	var wroteFile bool
	if src.FromFile && strings.TrimSpace(result.Modified) != "" && result.Modified != src.Code {
		if err := os.WriteFile(src.Path, []byte(result.Modified), src.Perm); err != nil {
			_ = toolenv.WriteError(os.Stdout, "failed to write modified file", err)
			os.Exit(1)
		}
		wroteFile = true
	}

	responseText := formatReplaceResponse(result, src.Path, src.FromFile, wroteFile)

	payload := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": responseText,
			},
		},
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

func formatReplaceResponse(result core.TransformResult, path string, fromFile bool, applied bool) string {
	var builder strings.Builder
	builder.WriteString("✅ Replace operation completed successfully\n\n")

	if fromFile {
		builder.WriteString(fmt.Sprintf("📄 File: %s\n", path))
		if applied {
			builder.WriteString("Changes written to disk.\n\n")
		} else {
			builder.WriteString("Preview only; file not modified.\n\n")
		}
	}

	builder.WriteString(fmt.Sprintf("Replacements made: %d\n", result.MatchCount))
	if strings.TrimSpace(result.Diff) != "" {
		builder.WriteString("\nDiff:\n")
		builder.WriteString(result.Diff)
		builder.WriteString("\n")
	}

	builder.WriteString("\nConfidence: ")
	builder.WriteString(formatConfidence(result.Confidence.Score))
	builder.WriteString(fmt.Sprintf(" (%.1f%%)", result.Confidence.Score*100))

	return builder.String()
}

func formatConfidence(score float64) string {
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	filled := int(score * 10)
	if filled > 10 {
		filled = 10
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}
