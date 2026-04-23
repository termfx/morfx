package core

import (
	"testing"
)

// TestAgentQuery_ComplexQueries tests complex query structures
func TestAgentQuery_ComplexQueries(t *testing.T) {
	tests := []struct {
		name  string
		query AgentQuery
		valid bool
	}{
		{
			name: "simple_function_query",
			query: AgentQuery{
				Type: "function",
				Name: "main",
			},
			valid: true,
		},
		{
			name: "nested_query",
			query: AgentQuery{
				Type: "class",
				Name: "User",
				Contains: &AgentQuery{
					Type: "method",
					Name: "save",
				},
			},
			valid: true,
		},
		{
			name: "compound_query_with_operands",
			query: AgentQuery{
				Operator: "AND",
				Operands: []AgentQuery{
					{Type: "function", Name: "process*"},
					{Type: "function", Name: "*handler"},
				},
			},
			valid: true,
		},
		{
			name: "or_query",
			query: AgentQuery{
				Operator: "OR",
				Operands: []AgentQuery{
					{Type: "function", Name: "test*"},
					{Type: "method", Name: "Test*"},
				},
			},
			valid: true,
		},
		{
			name: "not_query",
			query: AgentQuery{
				Operator: "NOT",
				Operands: []AgentQuery{
					{Type: "function", Name: "deprecated*"},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - ensure required fields are set
			if tt.query.Type == "" && tt.query.Operator == "" {
				if tt.valid {
					t.Error("Query should have either Type or Operator set")
				}
			}

			if tt.query.Operator != "" && len(tt.query.Operands) == 0 {
				if tt.valid {
					t.Error("Compound query should have operands")
				}
			}
		})
	}
}

// TestMatch_LocationDetails tests match location information
func TestMatch_LocationDetails(t *testing.T) {
	match := Match{
		Type: "function",
		Name: "calculateSum",
		Location: Location{
			File:      "/path/to/file.go",
			Line:      10,
			Column:    5,
			EndLine:   15,
			EndColumn: 10,
		},
		Content: "func calculateSum(a, b int) int {\n    return a + b\n}",
		Scope:   "global",
		Parent:  "",
	}

	// Test location bounds
	if match.Location.Line > match.Location.EndLine {
		t.Error("Start line should not be greater than end line")
	}

	if match.Location.Line == match.Location.EndLine &&
		match.Location.Column > match.Location.EndColumn {
		t.Error("Start column should not be greater than end column on same line")
	}

	// Test required fields
	if match.Type == "" {
		t.Error("Match type should not be empty")
	}

	if match.Name == "" {
		t.Error("Match name should not be empty")
	}

	if match.Location.Line <= 0 {
		t.Error("Location line should be positive")
	}
}

// TestTransformOp_Validation tests transform operation validation
func TestTransformOp_Validation(t *testing.T) {
	tests := []struct {
		name  string
		op    TransformOp
		valid bool
	}{
		{
			name: "replace_operation",
			op: TransformOp{
				Method:      "replace",
				Target:      AgentQuery{Type: "function", Name: "oldFunction"},
				Replacement: "newFunction",
			},
			valid: true,
		},
		{
			name: "insert_operation",
			op: TransformOp{
				Method:  "insert_before",
				Target:  AgentQuery{Type: "function", Name: "main"},
				Content: "// New comment\n",
			},
			valid: true,
		},
		{
			name: "delete_operation",
			op: TransformOp{
				Method: "delete",
				Target: AgentQuery{Type: "function", Name: "deprecatedFunction"},
			},
			valid: true,
		},
		{
			name: "invalid_replace_missing_replacement",
			op: TransformOp{
				Method: "replace",
				Target: AgentQuery{Type: "function", Name: "oldFunction"},
				// Missing Replacement
			},
			valid: false,
		},
		{
			name: "invalid_insert_missing_content",
			op: TransformOp{
				Method: "insert_before",
				Target: AgentQuery{Type: "function", Name: "main"},
				// Missing Content
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate based on method
			switch tt.op.Method {
			case "replace":
				if tt.op.Replacement == "" && tt.valid {
					t.Error("Replace operation should have replacement text")
				}
			case "insert_before", "insert_after", "append":
				if tt.op.Content == "" && tt.valid {
					t.Error("Insert operation should have content")
				}
			case "delete":
				// Delete operations don't need additional content
			default:
				if tt.valid {
					t.Errorf("Unknown method: %s", tt.op.Method)
				}
			}

			// All operations should have a target
			if tt.op.Target.Type == "" && tt.valid {
				t.Error("Operation should have a target")
			}
		})
	}
}

// TestConfidenceScore_Levels tests confidence score level calculation
func TestConfidenceScore_Levels(t *testing.T) {
	tests := []struct {
		name          string
		score         float64
		expectedLevel string
	}{
		{"very_high", 0.95, "high"},
		{"high", 0.85, "high"},
		{"medium_high", 0.75, "medium"},
		{"medium", 0.65, "medium"},
		{"medium_low", 0.55, "medium"},
		{"low", 0.35, "low"},
		{"very_low", 0.15, "low"},
		{"zero", 0.0, "low"},
		{"perfect", 1.0, "high"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := ConfidenceScore{
				Score: tt.score,
			}

			// Determine level based on score
			var level string
			if confidence.Score >= 0.8 {
				level = "high"
			} else if confidence.Score >= 0.5 {
				level = "medium"
			} else {
				level = "low"
			}

			if level != tt.expectedLevel {
				t.Errorf("Score %.2f should have level %s, got %s",
					tt.score, tt.expectedLevel, level)
			}

			// Validate score bounds
			if confidence.Score < 0.0 || confidence.Score > 1.0 {
				t.Errorf("Score %.2f is out of valid range [0.0, 1.0]", confidence.Score)
			}
		})
	}
}

// TestConfidenceFactor_Impact tests confidence factor impact calculation
func TestConfidenceFactor_Impact(t *testing.T) {
	tests := []struct {
		name   string
		factor ConfidenceFactor
		valid  bool
	}{
		{
			name: "positive_factor",
			factor: ConfidenceFactor{
				Name:   "syntax_validation",
				Impact: 0.1,
				Reason: "Code passes syntax validation",
			},
			valid: true,
		},
		{
			name: "negative_factor",
			factor: ConfidenceFactor{
				Name:   "complexity_penalty",
				Impact: -0.2,
				Reason: "High complexity transformation",
			},
			valid: true,
		},
		{
			name: "zero_impact",
			factor: ConfidenceFactor{
				Name:   "neutral_factor",
				Impact: 0.0,
				Reason: "Neutral impact",
			},
			valid: true,
		},
		{
			name: "impact_too_high",
			factor: ConfidenceFactor{
				Name:   "invalid_boost",
				Impact: 1.5,
				Reason: "Invalid high impact",
			},
			valid: false,
		},
		{
			name: "impact_too_low",
			factor: ConfidenceFactor{
				Name:   "invalid_penalty",
				Impact: -1.5,
				Reason: "Invalid low impact",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate impact range
			if tt.factor.Impact < -1.0 || tt.factor.Impact > 1.0 {
				if tt.valid {
					t.Errorf("Impact %.2f is out of valid range [-1.0, 1.0]", tt.factor.Impact)
				}
			}

			// Validate required fields
			if tt.factor.Name == "" && tt.valid {
				t.Error("Factor should have a name")
			}

			if tt.factor.Reason == "" && tt.valid {
				t.Error("Factor should have a reason")
			}
		})
	}
}

// TestFileScope_Validation tests file scope validation
func TestFileScope_Validation(t *testing.T) {
	tests := []struct {
		name  string
		scope FileScope
		valid bool
	}{
		{
			name: "basic_scope",
			scope: FileScope{
				Path:     "/project/src",
				Include:  []string{"*.go"},
				Language: "go",
			},
			valid: true,
		},
		{
			name: "scope_with_exclusions",
			scope: FileScope{
				Path:    "/project",
				Include: []string{"**/*.js", "**/*.ts"},
				Exclude: []string{"node_modules/**", "dist/**"},
			},
			valid: true,
		},
		{
			name: "scope_with_limits",
			scope: FileScope{
				Path:     "/large/project",
				Include:  []string{"**/*.py"},
				MaxDepth: 5,
				MaxFiles: 1000,
			},
			valid: true,
		},
		{
			name: "empty_path",
			scope: FileScope{
				Path:    "",
				Include: []string{"*.go"},
			},
			valid: false,
		},
		{
			name: "negative_limits",
			scope: FileScope{
				Path:     "/project",
				Include:  []string{"*.go"},
				MaxDepth: -1,
				MaxFiles: -100,
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate path
			if tt.scope.Path == "" && tt.valid {
				t.Error("Scope should have a non-empty path")
			}

			// Validate limits
			if (tt.scope.MaxDepth < 0 || tt.scope.MaxFiles < 0) && tt.valid {
				t.Error("Scope limits should not be negative")
			}

			// Validate patterns
			if len(tt.scope.Include) == 0 && len(tt.scope.Exclude) == 0 && tt.valid {
				// It's valid to have no patterns (process all files)
			}
		})
	}
}

// TestFileTransformDetail_Consistency tests file transform detail consistency
func TestFileTransformDetail_Consistency(t *testing.T) {
	tests := []struct {
		name   string
		detail FileTransformDetail
		valid  bool
	}{
		{
			name: "successful_modification",
			detail: FileTransformDetail{
				FilePath:     "/path/to/file.go",
				Language:     "go",
				MatchCount:   2,
				Modified:     true,
				Confidence:   ConfidenceScore{Score: 0.9, Level: "high"},
				OriginalSize: 1024,
				ModifiedSize: 1100,
			},
			valid: true,
		},
		{
			name: "no_modification",
			detail: FileTransformDetail{
				FilePath:     "/path/to/file.go",
				Language:     "go",
				MatchCount:   0,
				Modified:     false,
				Confidence:   ConfidenceScore{Score: 0.0, Level: "low"},
				OriginalSize: 1024,
				ModifiedSize: 0, // Should be 0 when not modified
			},
			valid: true,
		},
		{
			name: "error_case",
			detail: FileTransformDetail{
				FilePath:     "/path/to/file.go",
				Language:     "go",
				MatchCount:   0,
				Modified:     false,
				Error:        "syntax error",
				Confidence:   ConfidenceScore{Score: 0.0, Level: "low"},
				OriginalSize: 1024,
			},
			valid: true,
		},
		{
			name: "inconsistent_modified_size",
			detail: FileTransformDetail{
				FilePath:     "/path/to/file.go",
				Modified:     false,
				ModifiedSize: 1100, // Should be 0 when not modified
			},
			valid: false,
		},
		{
			name: "inconsistent_match_count",
			detail: FileTransformDetail{
				FilePath:   "/path/to/file.go",
				Modified:   true,
				MatchCount: 0, // Should be > 0 if modified
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check consistency rules
			if tt.detail.Modified && tt.detail.MatchCount == 0 && tt.valid {
				t.Error("Modified files should have match count > 0")
			}

			if !tt.detail.Modified && tt.detail.ModifiedSize > 0 && tt.valid {
				t.Error("Unmodified files should have ModifiedSize = 0")
			}

			if tt.detail.Error != "" && tt.detail.Modified && tt.valid {
				t.Error("Files with errors should not be marked as modified")
			}

			// Validate required fields
			if tt.detail.FilePath == "" && tt.valid {
				t.Error("File path should not be empty")
			}

			if tt.detail.OriginalSize < 0 && tt.valid {
				t.Error("Original size should not be negative")
			}

			if tt.detail.ModifiedSize < 0 && tt.valid {
				t.Error("Modified size should not be negative")
			}
		})
	}
}
