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

const fileDeleteHelp = `Usage: file_delete [-h]

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
  "target": {<core.AgentQuery payload>},
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
  "matches": <int total deletions>,
  "dry_run": <bool>,
  "errors": ["<issues>", ...],
  "transaction": "<optional transaction id>",
  "details": [<core.FileTransformDetail objects>]
}`

type fileDeleteRequest struct {
	Scope  *core.FileScope `json:"scope"`
	Target json.RawMessage `json:"target"`
	DryRun bool            `json:"dry_run"`
	Backup bool            `json:"backup"`
}

func main() {
	var showHelp bool
	flag.BoolVar(&showHelp, "h", false, "Show help message")
	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.Usage = func() {
		fmt.Print(fileDeleteHelp)
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

	req, err := toolenv.ReadJSON[fileDeleteRequest](os.Stdin)
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

	if len(req.Target) == 0 {
		_ = toolenv.WriteError(os.Stdout, "target is required", errors.New("missing target"))
		os.Exit(1)
	}

	var target core.AgentQuery
	if err := json.Unmarshal(req.Target, &target); err != nil {
		_ = toolenv.WriteError(os.Stdout, "invalid target structure", err)
		os.Exit(1)
	}

	op := core.FileTransformOp{
		TransformOp: core.TransformOp{
			Method: "delete",
			Target: target,
		},
		Scope:    *req.Scope,
		DryRun:   req.DryRun,
		Backup:   req.Backup,
		Parallel: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	processor := env.FileProcessor()
	result, err := processor.TransformFiles(ctx, op)
	if err != nil {
		_ = toolenv.WriteError(os.Stdout, "file delete failed", err)
		os.Exit(1)
	}

	responseText := formatFileDeleteResponse(result, req.DryRun)

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
		"details":         result.Files,
	}

	if err := toolenv.WriteJSON(os.Stdout, payload); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}
}

func formatFileDeleteResponse(result *core.FileTransformResult, dryRun bool) string {
	mode := ""
	if dryRun {
		mode = " [DRY RUN]"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("✅ File delete operation completed%s\n\n", mode))
	builder.WriteString(fmt.Sprintf("Files scanned: %d\n", result.FilesScanned))
	if dryRun {
		builder.WriteString(fmt.Sprintf("Files that would be modified: %d\n", result.FilesModified))
	} else {
		builder.WriteString(fmt.Sprintf("Files modified: %d\n", result.FilesModified))
	}
	builder.WriteString(fmt.Sprintf("Total deletions: %d\n", result.TotalMatches))

	if len(result.Files) > 0 {
		if dryRun {
			builder.WriteString("\nAffected files:\n")
		} else {
			builder.WriteString("\nModified files:\n")
		}
		for _, file := range result.Files {
			if file.MatchCount > 0 {
				builder.WriteString(fmt.Sprintf("📄 %s: %d deletions\n", file.FilePath, file.MatchCount))
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
