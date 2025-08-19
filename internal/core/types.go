package core

import (
	"time"
)

// Operation represents the type of code transformation to perform
type Operation string

const (
	// InsertBefore inserts code before the matched location
	InsertBefore Operation = "insert_before"
	// InsertAfter inserts code after the matched location
	InsertAfter Operation = "insert_after"
	// Replace replaces the matched code with new code
	Replace Operation = "replace"
	// Delete removes the matched code
	Delete Operation = "delete"
	// AppendToBody appends code to the body of a matched construct
	AppendToBody Operation = "append_to_body"
)

// Status represents the execution status of an operation
type Status string

const (
	// StatusSuccess indicates successful completion
	StatusSuccess Status = "success"
	// StatusError indicates an error occurred
	StatusError Status = "error"
	// StatusPartial indicates partial success with warnings
	StatusPartial Status = "partial"
	// StatusSkipped indicates the operation was skipped
	StatusSkipped Status = "skipped"
)

// Input represents the core input structure for Morfx operations
type Input struct {
	// Language specifies the programming language (e.g., "go", "python")
	Language string `json:"language"`

	// CodeIn contains the source code to transform
	CodeIn string `json:"code_in"`

	// Query is the DSL query to find target locations
	Query string `json:"query"`

	// Op specifies the operation to perform
	Op Operation `json:"op"`

	// Repl contains the replacement/insertion code (if applicable)
	Repl string `json:"repl,omitempty"`

	// Options contains additional configuration
	Options *InputOptions `json:"options,omitempty"`
}

// InputOptions contains optional configuration for operations
type InputOptions struct {
	// DryRun performs validation without applying changes
	DryRun bool `json:"dry_run,omitempty"`

	// Interactive enables interactive mode for confirmations
	Interactive bool `json:"interactive,omitempty"`

	// Fuzz enables fuzzy matching when exact matches fail
	Fuzz bool `json:"fuzz,omitempty"`

	// MaxFuzzDistance sets the maximum edit distance for fuzzy matching
	MaxFuzzDistance int `json:"max_fuzz_distance,omitempty"`

	// SkipValidation bypasses snippet validation
	SkipValidation bool `json:"skip_validation,omitempty"`

	// SkipFormat bypasses code formatting
	SkipFormat bool `json:"skip_format,omitempty"`

	// SkipImports bypasses import organization
	SkipImports bool `json:"skip_imports,omitempty"`

	// Context provides additional context for operations
	Context map[string]any `json:"context,omitempty"`
}

// Stats contains execution statistics
type Stats struct {
	// Duration is the total execution time
	Duration time.Duration `json:"duration"`

	// BytesProcessed is the number of bytes processed
	BytesProcessed int64 `json:"bytes_processed"`

	// LinesProcessed is the number of lines processed
	LinesProcessed int `json:"lines_processed"`

	// MatchesFound is the number of query matches found
	MatchesFound int `json:"matches_found"`

	// EditsApplied is the number of edits successfully applied
	EditsApplied int `json:"edits_applied"`

	// OverlapsDetected is the number of edit conflicts detected
	OverlapsDetected int `json:"overlaps_detected"`
}

// Diagnostic represents a warning or error message
type Diagnostic struct {
	// Severity indicates the diagnostic level
	Severity string `json:"severity"`

	// Code is an optional diagnostic code
	Code string `json:"code,omitempty"`

	// Message is the human-readable diagnostic message
	Message string `json:"message"`

	// File is the file path where the diagnostic occurred
	File string `json:"file,omitempty"`

	// Line is the line number (1-based)
	Line int `json:"line,omitempty"`

	// Column is the column number (1-based)
	Column int `json:"column,omitempty"`

	// Range specifies the affected text range
	Range *Location `json:"range,omitempty"`
}

// Engine contains metadata about the execution engine
type Engine struct {
	// Version is the Morfx version
	Version string `json:"version"`

	// Provider is the language provider used
	Provider string `json:"provider"`

	// TreeSitterVersion is the Tree-sitter version
	TreeSitterVersion string `json:"tree_sitter_version,omitempty"`

	// Timestamp is when the operation was executed
	Timestamp time.Time `json:"timestamp"`
}

// FuzzyMatch contains information about fuzzy matching results
type FuzzyMatch struct {
	// Used indicates whether fuzzy matching was used
	Used bool `json:"used"`

	// OriginalQuery is the original query that failed
	OriginalQuery string `json:"original_query,omitempty"`

	// ResolvedQuery is the fuzzy-matched query that succeeded
	ResolvedQuery string `json:"resolved_query,omitempty"`

	// Confidence is the confidence score (0.0 to 1.0)
	Confidence float64 `json:"confidence,omitempty"`

	// Score is the fuzzy match score (0.0 to 1.0)
	Score float64 `json:"score,omitempty"`

	// Distance is the edit distance from the original query
	Distance int `json:"distance,omitempty"`

	// Heuristics contains the heuristics that were applied
	Heuristics []string `json:"heuristics,omitempty"`
}

// PipelineResult represents the core output structure for Morfx pipeline operations
// This is different from core.Result which represents query match results
type PipelineResult struct {
	// Status indicates the overall execution status
	Status Status `json:"status"`

	// ResolvedOp is the actual operation that was performed
	ResolvedOp Operation `json:"resolved_op"`

	// CodeOut contains the transformed code
	CodeOut string `json:"code_out"`

	// Diff contains the unified diff of changes
	Diff string `json:"diff,omitempty"`

	// Stats contains execution statistics
	Stats Stats `json:"stats"`

	// Diagnostics contains warnings and errors
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`

	// Engine contains execution metadata
	Engine Engine `json:"engine"`

	// Hash is the SHA-256 hash of the output code
	Hash string `json:"hash"`

	// Fuzzy contains fuzzy matching information
	Fuzzy FuzzyMatch `json:"fuzzy"`
}
