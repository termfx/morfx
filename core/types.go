package core

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
