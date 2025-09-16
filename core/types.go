package core

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// AgentQuery represents a natural language query for code elements
type AgentQuery struct {
	Type     string       `json:"type"`               // function, struct, class, etc
	Name     string       `json:"name,omitempty"`     // name pattern with wildcards
	Contains *AgentQuery  `json:"contains,omitempty"` // nested queries
	Operator string       `json:"operator,omitempty"` // AND, OR, NOT
	Operands []AgentQuery `json:"operands,omitempty"` // for compound queries
}

// Match represents a found code element
type Match struct {
	Type     string   `json:"type"`
	Name     string   `json:"name"`
	Location Location `json:"location"`
	Content  string   `json:"content,omitempty"`
	Scope    string   `json:"scope,omitempty"`  // file, function, class
	Parent   string   `json:"parent,omitempty"` // parent element name
}

// Location in source code
type Location struct {
	File      string `json:"file,omitempty"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"end_line,omitempty"`
	EndColumn int    `json:"end_column,omitempty"`
}

// QueryResult from provider
type QueryResult struct {
	Matches []Match `json:"matches"`
	Total   int     `json:"total"`
	Error   error   `json:"-"`
}

// TransformOp represents a transformation operation
type TransformOp struct {
	Method      string     `json:"method"`                // replace, delete, insert_before, etc
	Target      AgentQuery `json:"target"`                // what to find
	Content     string     `json:"content,omitempty"`     // for insert/append
	Replacement string     `json:"replacement,omitempty"` // for replace
}

// TransformResult from provider
type TransformResult struct {
	Modified   string          `json:"modified"`
	Diff       string          `json:"diff"`
	Confidence ConfidenceScore `json:"confidence"`
	MatchCount int             `json:"match_count"`        // Number of elements matched/transformed
	Metadata   map[string]any  `json:"metadata,omitempty"` // Additional info (strategy, etc)
	Error      error           `json:"-"`
}

// ConfidenceScore for transformations
type ConfidenceScore struct {
	Score   float64            `json:"score"` // 0.0 to 1.0
	Level   string             `json:"level"` // high, medium, low
	Factors []ConfidenceFactor `json:"factors"`
}

// ConfidenceFactor explains score calculation
type ConfidenceFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"` // -1.0 to 1.0
	Reason string  `json:"reason"`
}

// FileScope defines which files to process in filesystem operations
type FileScope struct {
	Path           string   `json:"path"`                // Root path to scan
	Include        []string `json:"include,omitempty"`   // File patterns to include (*.go, **/*.ts)
	Exclude        []string `json:"exclude,omitempty"`   // File patterns to exclude
	MaxDepth       int      `json:"max_depth,omitempty"` // Max directory depth (0 = unlimited)
	MaxFiles       int      `json:"max_files,omitempty"` // Max files to process (0 = unlimited)
	FollowSymlinks bool     `json:"follow_symlinks"`     // Follow symbolic links
	Language       string   `json:"language,omitempty"`  // Auto-detect by extension if empty
}

// FileTransformOp represents a file-based transformation operation
type FileTransformOp struct {
	TransformOp           // Embedded base operation
	Scope       FileScope `json:"scope"`    // Files to operate on
	DryRun      bool      `json:"dry_run"`  // Preview only, don't modify files
	Backup      bool      `json:"backup"`   // Create .bak files before modifying
	Parallel    bool      `json:"parallel"` // Use parallel processing
}

// CodeMatch represents a specific code element match with precise location
type CodeMatch struct {
	Node      *sitter.Node `json:"-"`         // AST node (not serialized)
	Name      string       `json:"name"`      // Extracted name/identifier
	Type      string       `json:"type"`      // Query type that matched
	NodeType  string       `json:"node_type"` // AST node type
	StartByte uint32       `json:"start_byte"`
	EndByte   uint32       `json:"end_byte"`
	Line      uint32       `json:"line"`
	Column    uint32       `json:"column"`
}

// FileMatch represents a code match with file information
type FileMatch struct {
	Match           // Embedded base match
	FilePath string `json:"file_path"` // Absolute file path
	FileSize int64  `json:"file_size"` // File size in bytes
	ModTime  int64  `json:"mod_time"`  // Last modification time (Unix timestamp)
	Language string `json:"language"`  // Detected language
}

// FileTransformResult represents the result of file-based transformations
type FileTransformResult struct {
	FilesScanned      int                   `json:"files_scanned"`            // Total files processed
	FilesModified     int                   `json:"files_modified"`           // Files actually changed
	TotalMatches      int                   `json:"total_matches"`            // Total matches across all files
	ScanDuration      int64                 `json:"scan_duration_ms"`         // Time spent scanning (ms)
	TransformDuration int64                 `json:"transform_duration_ms"`    // Time spent transforming (ms)
	Files             []FileTransformDetail `json:"files"`                    // Per-file results
	Confidence        ConfidenceScore       `json:"confidence"`               // Overall confidence
	TransactionID     string                `json:"transaction_id,omitempty"` // Transaction ID for rollback
	Error             error                 `json:"-"`
}

// FileTransformDetail represents the transformation result for a single file
type FileTransformDetail struct {
	FilePath     string          `json:"file_path"`
	Language     string          `json:"language"`
	MatchCount   int             `json:"match_count"`
	Modified     bool            `json:"modified"`
	Diff         string          `json:"diff,omitempty"`
	Confidence   ConfidenceScore `json:"confidence"`
	Error        string          `json:"error,omitempty"`
	BackupPath   string          `json:"backup_path,omitempty"`
	OriginalSize int64           `json:"original_size"`
	ModifiedSize int64           `json:"modified_size"`
}
