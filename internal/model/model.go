package model

import (
	"github.com/garaekz/fileman/internal/types"
)

// Operation defines the type of modification to perform.
type (
	Operation string
	RuleID    string
)

const (
	OpReplace      Operation = "replace"
	OpInsertBefore Operation = "insert-before"
	OpInsertAfter  Operation = "insert-after"
	OpDelete       Operation = "delete"
	OpGet          Operation = "get"
	OpCommit       Operation = "commit"
)

// OccurrenceSpec defines how many occurrences of a pattern to modify.
type OccurrenceSpec struct {
	Max int // -1 means all occurrences.
}

// Config holds the configuration for a single transformation rule.
type Config struct {
	RuleID         string                 `json:"rule_id"`
	Pattern        string                 `json:"pattern"`
	Replacement    string                 `json:"replacement"`
	Operation      Operation              `json:"operation"`
	Occurrence     OccurrenceSpec         `json:"occurrence,omitzero"` // Optional, defaults to all
	Provider       types.LanguageProvider // Target language (e.g., "go", "python")
	DryRun         bool                   `json:"dry_run,omitempty"`           // If true, no files are written
	Interactive    bool                   `json:"interactive,omitempty"`       // If true, ask for confirmation before writing
	ShowDiff       bool                   `json:"show_diff,omitempty"`         // If true, show a unified diff of changes
	DiffContext    int                    `json:"diff_context,omitempty"`      // Lines of context for the diff
	Verbose        bool                   `json:"verbose,omitempty"`           // Enable verbose output
	JSONOutput     bool                   `json:"json_output,omitempty"`       // If true, output results in JSON format
	StdoutMode     bool                   `json:"stdout_mode,omitempty"`       // If true, output modified content to stdout
	ExitCodeNoDiff int                    `json:"exit_code_no_diff,omitempty"` // Exit code
	FailIfNoMatch  bool                   `json:"fail_if_no_match,omitempty"`  // If true, fail if no matches found
	Workers        int                    `json:"workers,omitempty"`           // Number of concurrent workers (default: runtime.NumCPU())
}

// Context defines constraints on the text surrounding a match.
type Context struct {
	Before       string `json:"before"`
	After        string `json:"after"`
	WindowBefore int    `json:"window_before"` // in lines (0 = unlimited)
	WindowAfter  int    `json:"window_after"`
}

// Result holds the outcome of processing a single file.
type Result struct {
	File            string   `json:"file"`
	Time            string   `json:"time"`
	SchemaVersion   int      `json:"schema_version"`
	ToolVersion     string   `json:"tool_version"`
	Success         bool     `json:"success"`
	ModifiedCount   int      `json:"modified_count"`
	ChangedBytes    int      `json:"changed_bytes"`
	Error           error    `json:"error,omitzero"`
	OriginalSHA1    string   `json:"original_sha1,omitempty"`
	ModifiedSHA1    string   `json:"modified_sha1,omitempty"`
	OriginalContent string   `json:"-"`                 // Omitted from JSON for brevity
	ModifiedContent string   `json:"-"`                 // Omitted from JSON for brevity
	Changes         []Change `json:"changes,omitempty"` // List of changes made
}

// Change represents a single modification within a file.
type Change struct {
	RuleID    string `json:"rule_id"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Start     int    `json:"start"` // byte offsets
	End       int    `json:"end"`
	Original  string `json:"original"`
	New       string `json:"new"`
}

// ToolConfig is the top-level structure for running the tool with a config file.
type ToolConfig struct {
	SchemaVersion  int      `json:"schema_version"`
	Files          []string `json:"files"`
	Rules          []Config `json:"rules"`
	FailIfNoMatch  bool     `json:"fail_if_no_match"`
	ExitCodeNoDiff int      `json:"exit_code_no_diff"`
}

const (
	CurrentSchemaVersion = 1
	CurrentToolVersion   = "0.3.0-test"
)

func (r *Result) NewErroringResult(err error) Result {
	return Result{
		File:          r.File,
		Time:          r.Time,
		SchemaVersion: r.SchemaVersion,
		ToolVersion:   r.ToolVersion,
		Success:       false,
		Error:         err,
	}
}

func (c *Config) CacheKey() string {
	return c.RuleID + "|" + c.Pattern + "|" + string(c.Operation) + "|" +
		c.Provider.Lang() + "|" + boolToStr(c.DryRun) + "|" +
		boolToStr(c.ShowDiff) + "|" + boolToStr(c.JSONOutput) +
		boolToStr(c.FailIfNoMatch) + "|" + boolToStr(c.Verbose) + "|" +
		boolToStr(c.DiffContext > 0) + "|" +
		boolToStr(c.Occurrence.Max != -1) + "|" +
		boolToStr(c.Workers > 0)
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
