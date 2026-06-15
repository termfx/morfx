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
	"github.com/oxhq/morfx/internal/toolcmd"
	"github.com/oxhq/morfx/internal/toolenv"
)

const replaceHelp = `Usage: replace [-h]

Reads a JSON request from stdin and emits a JSON response to stdout.

Input schema:
{
  "language": "<language id>",
  "source":   "<optional source code>",
  "path":     "<optional file path>",
  "target":   {<optional core.AgentQuery payload>},
  "target_dsl": "<optional Morfx DSL selector, such as func:Legacy*>",
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
	TargetDSL   string          `json:"target_dsl,omitempty"`
	Replacement string          `json:"replacement"`
}

type replaceRunner interface {
	Transform(context.Context, engine.TransformRequest) (engine.TransformResult, error)
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
		fmt.Print(replaceHelp)
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

	req, err := toolenv.ReadJSON[replaceRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	payload, err := runReplace(context.Background(), e, *req)
	if err != nil {
		writeCommandError(err)
		os.Exit(1)
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func runReplace(ctx context.Context, runner replaceRunner, req replaceRequest) (map[string]any, error) {
	if strings.TrimSpace(req.Language) == "" {
		return nil, commandError{message: "language is required", cause: errors.New("missing language")}
	}
	if len(req.Target) == 0 && strings.TrimSpace(req.TargetDSL) == "" {
		return nil, commandError{message: "target is required", cause: errors.New("missing target")}
	}
	if strings.TrimSpace(req.Replacement) == "" {
		return nil, commandError{message: "replacement is required", cause: errors.New("missing replacement")}
	}

	src, err := toolenv.LoadSource(req.Source, req.Path)
	if err != nil {
		return nil, commandError{message: "failed to resolve source", cause: err}
	}

	target, err := core.ParseAgentQueryPayload(req.Target, req.TargetDSL)
	if err != nil {
		return nil, commandError{message: "invalid target structure", cause: err}
	}

	result, err := runner.Transform(ctx, engine.TransformRequest{
		Language: req.Language,
		Source:   src.Code,
		Op: core.TransformOp{
			Method:      "replace",
			Target:      target,
			Replacement: req.Replacement,
		},
	})
	if err != nil {
		return nil, commandError{message: "replace operation failed", cause: err}
	}

	wroteFile, err := toolcmd.WriteModifiedSource(src.Path, src.FromFile, src.Code, result.Modified, src.Perm)
	if err != nil {
		return nil, commandError{message: "failed to write modified file", cause: err}
	}

	formatted := core.TransformResult{
		MatchCount: result.MatchCount,
		Modified:   result.Modified,
		Diff:       result.Diff,
		Confidence: result.Confidence,
	}
	responseText := formatReplaceResponse(formatted, src.Path, src.FromFile, wroteFile)

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

	return payload, nil
}

func writeCommandError(err error) {
	var cmdErr commandError
	if errors.As(err, &cmdErr) {
		_ = toolenv.WriteError(os.Stdout, cmdErr.message, cmdErr.cause)
		return
	}
	_ = toolenv.WriteError(os.Stdout, "replace operation failed", err)
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
	builder.WriteString(toolcmd.FormatConfidence(result.Confidence.Score))
	builder.WriteString(fmt.Sprintf(" (%.1f%%)", result.Confidence.Score*100))

	return builder.String()
}
