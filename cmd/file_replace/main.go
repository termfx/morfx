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

const fileReplaceHelp = `Usage: file_replace [-h]

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
  "target": {<optional core.AgentQuery payload>},
  "target_dsl": "<optional Morfx DSL selector, such as func:* > call:os.Getenv>",
  "replacement": "<text to insert>",
  "dry_run": <bool>,
  "backup": <bool>
}
"path" must reference an accessible directory. When "dry_run" is true the
filesystem is not modified.

Output schema:
{
  "content": [{"type": "text", "text": "<summary>"}],
  "files_processed": <int>,
  "files_modified": <int>,
  "matches": <int total matches>,
  "dry_run": <bool>,
  "errors": ["<issues>", ...],
  "transaction": "<optional transaction id>",
  "details": [<core.FileTransformDetail objects>]
}`

type fileReplaceRequest struct {
	Scope       *core.FileScope `json:"scope"`
	Target      json.RawMessage `json:"target"`
	TargetDSL   string          `json:"target_dsl,omitempty"`
	Replacement string          `json:"replacement"`
	DryRun      bool            `json:"dry_run"`
	Backup      bool            `json:"backup"`
}

type fileReplaceRunner interface {
	FileReplace(context.Context, engine.FileReplaceRequest) (engine.FileReplaceResult, error)
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
		fmt.Print(fileReplaceHelp)
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

	req, err := toolenv.ReadJSON[fileReplaceRequest](os.Stdin)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid input", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payload, err := runFileReplace(ctx, e, *req)
	if err != nil {
		writeCommandError(err)
		os.Exit(1)
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func runFileReplace(ctx context.Context, runner fileReplaceRunner, req fileReplaceRequest) (map[string]any, error) {
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

	if len(req.Target) == 0 && strings.TrimSpace(req.TargetDSL) == "" {
		return nil, commandError{message: "target is required", cause: errors.New("missing target")}
	}
	if strings.TrimSpace(req.Replacement) == "" {
		return nil, commandError{message: "replacement is required", cause: errors.New("missing replacement")}
	}

	target, err := core.ParseAgentQueryPayload(req.Target, req.TargetDSL)
	if err != nil {
		return nil, commandError{message: "invalid target structure", cause: err}
	}

	scope := *req.Scope
	scope.Path = absPath
	result, err := runner.FileReplace(ctx, engine.FileReplaceRequest{
		Scope: scope,
		Op: core.TransformOp{
			Method:      "replace",
			Target:      target,
			Replacement: req.Replacement,
		},
		DryRun: req.DryRun,
		Backup: req.Backup,
	})
	if err != nil {
		return nil, commandError{message: "file replace failed", cause: err}
	}

	formatted := &core.FileTransformResult{
		FilesScanned:  result.FilesScanned,
		FilesModified: result.FilesModified,
		TotalMatches:  result.TotalMatches,
		Errors:        result.Errors,
		TransactionID: result.TransactionID,
		Files:         result.Details,
	}
	responseText := formatFileReplaceResponse(formatted, req.DryRun)
	payload := map[string]any{
		"content": []map[string]any{{
			"type": "text",
			"text": responseText,
		}},
		"files_processed": result.FilesScanned,
		"files_modified":  result.FilesModified,
		"matches":         result.TotalMatches,
		"dry_run":         req.DryRun,
		"errors":          result.Errors,
		"transaction":     result.TransactionID,
		"details":         result.Details,
	}

	return payload, nil
}

func writeCommandError(err error) {
	var cmdErr commandError
	if errors.As(err, &cmdErr) {
		_ = toolenv.WriteError(os.Stdout, cmdErr.message, cmdErr.cause)
		return
	}
	_ = toolenv.WriteError(os.Stdout, "file replace failed", err)
}

func formatFileReplaceResponse(result *core.FileTransformResult, dryRun bool) string {
	mode := ""
	if dryRun {
		mode = " [DRY RUN]"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("✅ File replace operation completed%s\n\n", mode))
	builder.WriteString(fmt.Sprintf("Files scanned: %d\n", result.FilesScanned))
	if dryRun {
		builder.WriteString(fmt.Sprintf("Files that would be modified: %d\n", result.FilesModified))
	} else {
		builder.WriteString(fmt.Sprintf("Files modified: %d\n", result.FilesModified))
	}
	builder.WriteString(fmt.Sprintf("Total matches: %d\n", result.TotalMatches))

	if len(result.Files) > 0 {
		if dryRun {
			builder.WriteString("\nAffected files:\n")
		} else {
			builder.WriteString("\nModified files:\n")
		}
		for _, file := range result.Files {
			if file.MatchCount > 0 {
				builder.WriteString(fmt.Sprintf("📄 %s: %d changes\n", file.FilePath, file.MatchCount))
			}
		}
	}

	if len(result.Errors) > 0 {
		builder.WriteString("\n⚠️  Issues encountered:\n")
		for _, issue := range result.Errors {
			builder.WriteString("- " + issue + "\n")
		}
	}

	if dryRun {
		builder.WriteString("\n⚠️  This was a dry run. No files were modified.\n")
	}

	return builder.String()
}
