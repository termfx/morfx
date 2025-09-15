package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/termfx/morfx/core"
)

// handleFileQueryTool executes file-based code queries
func (s *StdioServer) handleFileQueryTool(params json.RawMessage) (any, error) {
	var args struct {
		Scope core.FileScope  `json:"scope"`
		Query json.RawMessage `json:"query"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid file query parameters", err)
	}

	// Parse the query
	var query core.AgentQuery
	if err := json.Unmarshal(args.Query, &query); err != nil {
		return nil, WrapError(InvalidParams, "Invalid query structure", err)
	}

	s.debugLog("File query: %s in path %s", query.Type, args.Scope.Path)

	// Create file processor
	fileProcessor := core.NewFileProcessor(&providerRegistryAdapter{s.providers})

	// Execute file query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	matches, err := fileProcessor.QueryFiles(ctx, args.Scope, query)
	if err != nil {
		return nil, WrapError(TransformFailed, "File query failed", err)
	}

	// Format response
	responseText := s.formatFileQueryResponse(matches, args.Scope)

	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": responseText,
			},
		},
		"matches": len(matches),
		"files":   s.countUniqueFiles(matches),
	}, nil
}

// handleFileReplaceTool executes file-based replacement transformations
func (s *StdioServer) handleFileReplaceTool(params json.RawMessage) (any, error) {
	var args struct {
		Scope       core.FileScope  `json:"scope"`
		Target      json.RawMessage `json:"target"`
		Replacement string          `json:"replacement"`
		DryRun      bool            `json:"dry_run"`
		Backup      bool            `json:"backup"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid file replace parameters", err)
	}

	// Parse target
	var target core.AgentQuery
	if err := json.Unmarshal(args.Target, &target); err != nil {
		return nil, WrapError(InvalidParams, "Invalid target structure", err)
	}

	s.debugLog("File replace: %s -> %s in path %s", target.Name, args.Replacement, args.Scope.Path)

	// Create transform operation
	fileOp := core.FileTransformOp{
		TransformOp: core.TransformOp{
			Method:      "replace",
			Target:      target,
			Replacement: args.Replacement,
		},
		Scope:    args.Scope,
		DryRun:   args.DryRun,
		Backup:   args.Backup,
		Parallel: true,
	}

	return s.executeFileTransform(fileOp)
}

// handleFileDeleteTool executes file-based deletion transformations
func (s *StdioServer) handleFileDeleteTool(params json.RawMessage) (any, error) {
	var args struct {
		Scope  core.FileScope  `json:"scope"`
		Target json.RawMessage `json:"target"`
		DryRun bool            `json:"dry_run"`
		Backup bool            `json:"backup"`
	}

	if err := json.Unmarshal(params, &args); err != nil {
		return nil, WrapError(InvalidParams, "Invalid file delete parameters", err)
	}

	// Parse target
	var target core.AgentQuery
	if err := json.Unmarshal(args.Target, &target); err != nil {
		return nil, WrapError(InvalidParams, "Invalid target structure", err)
	}

	s.debugLog("File delete: %s in path %s", target.Name, args.Scope.Path)

	// Create transform operation
	fileOp := core.FileTransformOp{
		TransformOp: core.TransformOp{
			Method: "delete",
			Target: target,
		},
		Scope:    args.Scope,
		DryRun:   args.DryRun,
		Backup:   args.Backup,
		Parallel: true,
	}

	return s.executeFileTransform(fileOp)
}

// executeFileTransform is the common execution logic for file operations
func (s *StdioServer) executeFileTransform(fileOp core.FileTransformOp) (any, error) {
	// Create file processor
	fileProcessor := core.NewFileProcessor(&providerRegistryAdapter{s.providers})

	// Execute transformation with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := fileProcessor.TransformFiles(ctx, fileOp)
	if err != nil {
		return nil, WrapError(TransformFailed, "File transformation failed", err)
	}

	// Validate changes if not dry run
	if !fileOp.DryRun {
		if err := fileProcessor.ValidateChanges(result.Files); err != nil {
			return nil, WrapError(TransformFailed, "Validation failed", err)
		}
	}

	// Format response
	responseText := s.formatFileTransformResponse(fileOp, result)

	// Prepare response
	response := map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": responseText,
			},
		},
		"files_scanned":  result.FilesScanned,
		"files_modified": result.FilesModified,
		"total_matches":  result.TotalMatches,
		"confidence":     result.Confidence,
		"dry_run":        fileOp.DryRun,
	}

	// Add timing information
	if result.ScanDuration > 0 {
		response["scan_duration_ms"] = result.ScanDuration
	}
	if result.TransformDuration > 0 {
		response["transform_duration_ms"] = result.TransformDuration
	}

	// Include file details if requested
	if len(result.Files) <= 10 {
		response["file_details"] = result.Files
	}

	return response, nil
}

// formatFileQueryResponse creates human-readable response for file queries
func (s *StdioServer) formatFileQueryResponse(matches []core.FileMatch, scope core.FileScope) string {
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found in %s", scope.Path)
	}

	// Group matches by file
	fileGroups := make(map[string][]core.FileMatch)
	for _, match := range matches {
		fileGroups[match.FilePath] = append(fileGroups[match.FilePath], match)
	}

	responseText := fmt.Sprintf("Found %d matches across %d files in %s:\n\n",
		len(matches), len(fileGroups), scope.Path)

	fileCount := 0
	for filePath, fileMatches := range fileGroups {
		fileCount++
		if fileCount > 5 { // Limit output
			responseText += fmt.Sprintf("... and %d more files\n", len(fileGroups)-5)
			break
		}

		responseText += fmt.Sprintf("📁 %s (%s, %d matches):\n",
			filePath, fileMatches[0].Language, len(fileMatches))

		for i, match := range fileMatches {
			if i >= 3 { // Limit matches per file
				responseText += fmt.Sprintf("   ... and %d more matches\n", len(fileMatches)-3)
				break
			}
			responseText += fmt.Sprintf("   • %s '%s' at line %d\n",
				match.Type, match.Name, match.Location.Line)
		}
		responseText += "\n"
	}

	return responseText
}

// formatFileTransformResponse creates human-readable response for file transforms
func (s *StdioServer) formatFileTransformResponse(op core.FileTransformOp, result *core.FileTransformResult) string {
	action := op.Method
	switch action {
	case "replace":
		action = "Replaced"
	case "delete":
		action = "Deleted"
	case "insert_before":
		action = "Inserted before"
	case "insert_after":
		action = "Inserted after"
	}

	var text string
	if op.DryRun {
		text = fmt.Sprintf("🔍 DRY RUN: %s %d targets across %d files\n",
			action, result.TotalMatches, result.FilesModified)
		text += "No files were actually modified.\n\n"
	} else {
		text = fmt.Sprintf("✅ %s %d targets across %d files\n",
			action, result.TotalMatches, result.FilesModified)
	}

	// Add confidence info
	text += fmt.Sprintf("Overall Confidence: %s (%.2f)\n",
		result.Confidence.Level, result.Confidence.Score)

	// Add timing info
	totalTime := result.ScanDuration + result.TransformDuration
	text += fmt.Sprintf("Performance: %dms scan + %dms transform = %dms total\n\n",
		result.ScanDuration, result.TransformDuration, totalTime)

	// Add confidence factors
	if len(result.Confidence.Factors) > 0 {
		text += "Confidence Factors:\n"
		for _, factor := range result.Confidence.Factors {
			sign := "+"
			if factor.Impact < 0 {
				sign = ""
			}
			text += fmt.Sprintf("• %s (%s%.2f): %s\n",
				factor.Name, sign, factor.Impact, factor.Reason)
		}
		text += "\n"
	}

	// Show sample of modified files
	modifiedFiles := 0
	for _, file := range result.Files {
		if file.Modified {
			modifiedFiles++
			if modifiedFiles <= 3 {
				text += fmt.Sprintf("📝 %s (%d matches", file.FilePath, file.MatchCount)
				if file.BackupPath != "" {
					text += fmt.Sprintf(", backup: %s", file.BackupPath)
				}
				text += ")\n"
			}
		}
	}

	if modifiedFiles > 3 {
		text += fmt.Sprintf("... and %d more files\n", modifiedFiles-3)
	}

	// Add warnings for errors
	errorCount := 0
	for _, file := range result.Files {
		if file.Error != "" {
			errorCount++
		}
	}

	if errorCount > 0 {
		text += fmt.Sprintf("\n⚠️ %d files had errors and were skipped\n", errorCount)
	}

	return text
}

// countUniqueFiles counts unique file paths in matches
func (s *StdioServer) countUniqueFiles(matches []core.FileMatch) int {
	files := make(map[string]bool)
	for _, match := range matches {
		files[match.FilePath] = true
	}
	return len(files)
}
