package model

// Operation defines the type of modification to perform.
type Operation string

const (
	OpReplace      Operation = "replace"
	OpInsertBefore Operation = "insert-before"
	OpInsertAfter  Operation = "insert-after"
	OpDelete       Operation = "delete"
)

// OccurrenceSpec defines how many occurrences of a pattern to modify.
type OccurrenceSpec struct {
	Max int // -1 means all occurrences.
}

// ModificationConfig holds the configuration for a single transformation rule.
type ModificationConfig struct {
	RuleID              string    `json:"rule_id"`
	Pattern             string    `json:"pattern"`
	Replacement         string    `json:"replacement"`
	Operation           Operation `json:"operation"`
	Occurrences         string    `json:"occurrences"` // "first", "all", or a number
	Multiline           bool      `json:"multiline"`
	DotAll              bool      `json:"dot_all"`
	Context             *Context  `json:"context"`
	MustMatch           int       `json:"must_match"`
	MustChangeBytes     int       `json:"must_change_bytes"`
	NormalizeWhitespace bool      `json:"normalize_whitespace"`
	LiteralPattern      bool      `json:"literal_pattern"`
	UseAST              bool      `json:"use_ast"`
	Lang                string    `json:"lang"` // "go", "python", etc.
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
	File            string    `json:"file"`
	Time            string    `json:"time"`
	SchemaVersion   int       `json:"schema_version"`
	ToolVersion     string    `json:"tool_version"`
	Success         bool      `json:"success"`
	ModifiedCount   int       `json:"modified_count"`
	ChangedBytes    int       `json:"changed_bytes"`
	Error           string    `json:"error,omitempty"`
	ErrorCode       ErrorCode `json:"error_code,omitempty"`
	OriginalSHA1    string    `json:"original_sha1,omitempty"`
	ModifiedSHA1    string    `json:"modified_sha1,omitempty"`
	OriginalContent string    `json:"-"`                 // Omitted from JSON for brevity
	ModifiedContent string    `json:"-"`                 // Omitted from JSON for brevity
	Changes         []Change  `json:"changes,omitempty"` // List of changes made
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
	SchemaVersion  int                  `json:"schema_version"`
	Files          []string             `json:"files"`
	Rules          []ModificationConfig `json:"rules"`
	FailIfNoMatch  bool                 `json:"fail_if_no_match"`
	ExitCodeNoDiff int                  `json:"exit_code_no_diff"`
}

const (
	CurrentSchemaVersion = 1
	CurrentToolVersion   = "0.3.0-test"
)
