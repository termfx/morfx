package core

import (
	"testing"
)

func TestNodeKindConstants(t *testing.T) {
	tests := []struct {
		name     string
		kind     NodeKind
		expected string
	}{
		{"KindFunction", KindFunction, "function"},
		{"KindVariable", KindVariable, "variable"},
		{"KindClass", KindClass, "class"},
		{"KindMethod", KindMethod, "method"},
		{"KindImport", KindImport, "import"},
		{"KindConstant", KindConstant, "constant"},
		{"KindField", KindField, "field"},
		{"KindCall", KindCall, "call"},
		{"KindAssignment", KindAssignment, "assignment"},
		{"KindCondition", KindCondition, "condition"},
		{"KindLoop", KindLoop, "loop"},
		{"KindBlock", KindBlock, "block"},
		{"KindComment", KindComment, "comment"},
		{"KindDecorator", KindDecorator, "decorator"},
		{"KindType", KindType, "type"},
		{"KindInterface", KindInterface, "interface"},
		{"KindEnum", KindEnum, "enum"},
		{"KindParameter", KindParameter, "parameter"},
		{"KindReturn", KindReturn, "return"},
		{"KindThrow", KindThrow, "throw"},
		{"KindTryCatch", KindTryCatch, "try_catch"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.kind) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.kind))
			}
		})
	}
}

func TestScopeTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		scope    ScopeType
		expected string
	}{
		{"ScopeFile", ScopeFile, "file"},
		{"ScopeClass", ScopeClass, "class"},
		{"ScopeFunction", ScopeFunction, "function"},
		{"ScopeBlock", ScopeBlock, "block"},
		{"ScopeNamespace", ScopeNamespace, "namespace"},
		{"ScopePackage", ScopePackage, "package"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.scope) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.scope))
			}
		})
	}
}

func TestQueryConstruction(t *testing.T) {
	tests := []struct {
		name     string
		query    Query
		validate func(*testing.T, Query)
	}{
		{
			name: "simple query",
			query: Query{
				Kind:       KindFunction,
				Pattern:    "test*",
				Attributes: map[string]string{"type": "public"},
				Operator:   "",
				Children:   []Query{},
				Scope:      ScopeFile,
				Raw:        "function:test* public",
			},
			validate: func(t *testing.T, q Query) {
				if q.Kind != KindFunction {
					t.Errorf("Expected Kind %s, got %s", KindFunction, q.Kind)
				}
				if q.Pattern != "test*" {
					t.Errorf("Expected Pattern 'test*', got '%s'", q.Pattern)
				}
				if q.Attributes["type"] != "public" {
					t.Errorf("Expected type attribute 'public', got '%s'", q.Attributes["type"])
				}
				if q.Scope != ScopeFile {
					t.Errorf("Expected Scope %s, got %s", ScopeFile, q.Scope)
				}
			},
		},
		{
			name: "logical query with children",
			query: Query{
				Kind:     "logical",
				Pattern:  "",
				Operator: "AND",
				Children: []Query{
					{Kind: KindFunction, Pattern: "test*", Attributes: map[string]string{}, Raw: "function:test*"},
					{Kind: KindVariable, Pattern: "var*", Attributes: map[string]string{}, Raw: "variable:var*"},
				},
				Attributes: map[string]string{},
				Raw:        "function:test* & variable:var*",
			},
			validate: func(t *testing.T, q Query) {
				if q.Operator != "AND" {
					t.Errorf("Expected Operator 'AND', got '%s'", q.Operator)
				}
				if len(q.Children) != 2 {
					t.Errorf("Expected 2 children, got %d", len(q.Children))
				}
				if q.Children[0].Kind != KindFunction {
					t.Errorf("Expected first child Kind %s, got %s", KindFunction, q.Children[0].Kind)
				}
				if q.Children[1].Kind != KindVariable {
					t.Errorf("Expected second child Kind %s, got %s", KindVariable, q.Children[1].Kind)
				}
			},
		},
		{
			name: "hierarchical query",
			query: Query{
				Kind:     KindMethod,
				Pattern:  "getName",
				Operator: "HIERARCHY",
				Children: []Query{
					{Kind: KindClass, Pattern: "User", Attributes: map[string]string{}, Raw: "class:User"},
				},
				Attributes: map[string]string{},
				Raw:        "class:User > method:getName",
			},
			validate: func(t *testing.T, q Query) {
				if q.Operator != "HIERARCHY" {
					t.Errorf("Expected Operator 'HIERARCHY', got '%s'", q.Operator)
				}
				if len(q.Children) != 1 {
					t.Errorf("Expected 1 child, got %d", len(q.Children))
				}
				if q.Children[0].Kind != KindClass {
					t.Errorf("Expected child Kind %s, got %s", KindClass, q.Children[0].Kind)
				}
			},
		},
		{
			name: "negated query",
			query: Query{
				Kind:       KindFunction,
				Pattern:    "test*",
				Operator:   "NOT",
				Attributes: map[string]string{},
				Children:   []Query{},
				Raw:        "function:test*",
			},
			validate: func(t *testing.T, q Query) {
				if q.Operator != "NOT" {
					t.Errorf("Expected Operator 'NOT', got '%s'", q.Operator)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.query)
		})
	}
}

func TestLocationConstruction(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		validate func(*testing.T, Location)
	}{
		{
			name: "basic location",
			location: Location{
				File:      "test.go",
				StartLine: 10,
				EndLine:   15,
				StartCol:  5,
				EndCol:    20,
				StartByte: 100,
				EndByte:   150,
			},
			validate: func(t *testing.T, loc Location) {
				if loc.File != "test.go" {
					t.Errorf("Expected File 'test.go', got '%s'", loc.File)
				}
				if loc.StartLine != 10 {
					t.Errorf("Expected StartLine 10, got %d", loc.StartLine)
				}
				if loc.EndLine != 15 {
					t.Errorf("Expected EndLine 15, got %d", loc.EndLine)
				}
				if loc.StartCol != 5 {
					t.Errorf("Expected StartCol 5, got %d", loc.StartCol)
				}
				if loc.EndCol != 20 {
					t.Errorf("Expected EndCol 20, got %d", loc.EndCol)
				}
				if loc.StartByte != 100 {
					t.Errorf("Expected StartByte 100, got %d", loc.StartByte)
				}
				if loc.EndByte != 150 {
					t.Errorf("Expected EndByte 150, got %d", loc.EndByte)
				}
			},
		},
		{
			name: "single line location",
			location: Location{
				File:      "single.go",
				StartLine: 5,
				EndLine:   5,
				StartCol:  1,
				EndCol:    10,
				StartByte: 50,
				EndByte:   59,
			},
			validate: func(t *testing.T, loc Location) {
				if loc.StartLine != loc.EndLine {
					t.Errorf("Expected single line location, got StartLine=%d, EndLine=%d", loc.StartLine, loc.EndLine)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.location)
		})
	}
}

func TestResultConstruction(t *testing.T) {
	tests := []struct {
		name     string
		result   Result
		validate func(*testing.T, Result)
	}{
		{
			name: "function result",
			result: Result{
				Kind: KindFunction,
				Name: "testFunction",
				Location: Location{
					File:      "test.go",
					StartLine: 10,
					EndLine:   20,
					StartCol:  1,
					EndCol:    2,
					StartByte: 100,
					EndByte:   200,
				},
				Content: "func testFunction() {\n\t// implementation\n}",
				Metadata: map[string]any{
					"visibility": "public",
					"parameters": 0,
					"returns":    1,
				},
				ParentKind: KindClass,
				ParentName: "TestClass",
				Scope:      ScopeClass,
			},
			validate: func(t *testing.T, r Result) {
				if r.Kind != KindFunction {
					t.Errorf("Expected Kind %s, got %s", KindFunction, r.Kind)
				}
				if r.Name != "testFunction" {
					t.Errorf("Expected Name 'testFunction', got '%s'", r.Name)
				}
				if r.ParentKind != KindClass {
					t.Errorf("Expected ParentKind %s, got %s", KindClass, r.ParentKind)
				}
				if r.ParentName != "TestClass" {
					t.Errorf("Expected ParentName 'TestClass', got '%s'", r.ParentName)
				}
				if r.Scope != ScopeClass {
					t.Errorf("Expected Scope %s, got %s", ScopeClass, r.Scope)
				}
				if visibility, ok := r.Metadata["visibility"]; !ok || visibility != "public" {
					t.Errorf("Expected visibility metadata 'public', got %v", visibility)
				}
			},
		},
		{
			name: "variable result",
			result: Result{
				Kind: KindVariable,
				Name: "testVar",
				Location: Location{
					File:      "test.go",
					StartLine: 5,
					EndLine:   5,
					StartCol:  5,
					EndCol:    12,
					StartByte: 50,
					EndByte:   57,
				},
				Content: "var testVar string",
				Metadata: map[string]any{
					"type":        "string",
					"mutable":     true,
					"initialized": false,
				},
				ParentKind: KindFunction,
				ParentName: "main",
				Scope:      ScopeFunction,
			},
			validate: func(t *testing.T, r Result) {
				if r.Kind != KindVariable {
					t.Errorf("Expected Kind %s, got %s", KindVariable, r.Kind)
				}
				if r.Name != "testVar" {
					t.Errorf("Expected Name 'testVar', got '%s'", r.Name)
				}
				if typeInfo, ok := r.Metadata["type"]; !ok || typeInfo != "string" {
					t.Errorf("Expected type metadata 'string', got %v", typeInfo)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.result)
		})
	}
}

func TestResultSetOperations(t *testing.T) {
	// Create test results
	results := []*Result{
		{
			Kind:     KindFunction,
			Name:     "func1",
			Location: Location{File: "test.go", StartLine: 1, EndLine: 5},
		},
		{
			Kind:     KindFunction,
			Name:     "func2",
			Location: Location{File: "test.go", StartLine: 10, EndLine: 15},
		},
		{
			Kind:     KindVariable,
			Name:     "var1",
			Location: Location{File: "test.go", StartLine: 20, EndLine: 20},
		},
	}

	tests := []struct {
		name      string
		resultSet ResultSet
		validate  func(*testing.T, ResultSet)
	}{
		{
			name: "basic result set",
			resultSet: ResultSet{
				Results:         results,
				QueryHash:       "abc123",
				TotalMatches:    3,
				ExecutionTimeMs: 150,
			},
			validate: func(t *testing.T, rs ResultSet) {
				if len(rs.Results) != 3 {
					t.Errorf("Expected 3 results, got %d", len(rs.Results))
				}
				if rs.TotalMatches != 3 {
					t.Errorf("Expected TotalMatches 3, got %d", rs.TotalMatches)
				}
				if rs.QueryHash != "abc123" {
					t.Errorf("Expected QueryHash 'abc123', got '%s'", rs.QueryHash)
				}
				if rs.ExecutionTimeMs != 150 {
					t.Errorf("Expected ExecutionTimeMs 150, got %d", rs.ExecutionTimeMs)
				}
			},
		},
		{
			name: "empty result set",
			resultSet: ResultSet{
				Results:         []*Result{},
				QueryHash:       "empty",
				TotalMatches:    0,
				ExecutionTimeMs: 10,
			},
			validate: func(t *testing.T, rs ResultSet) {
				if len(rs.Results) != 0 {
					t.Errorf("Expected 0 results, got %d", len(rs.Results))
				}
				if rs.TotalMatches != 0 {
					t.Errorf("Expected TotalMatches 0, got %d", rs.TotalMatches)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.resultSet)
		})
	}
}

func TestNodeMappingConstruction(t *testing.T) {
	tests := []struct {
		name     string
		mapping  NodeMapping
		validate func(*testing.T, NodeMapping)
	}{
		{
			name: "function mapping",
			mapping: NodeMapping{
				Kind:        KindFunction,
				NodeTypes:   []string{"function_declaration", "method_declaration"},
				NameCapture: "@name",
				TypeCapture: "@type",
				Template:    "(function_declaration name: (identifier) @name)",
				Attributes:  map[string]string{"visibility": "@visibility"},
				Priority:    100,
			},
			validate: func(t *testing.T, m NodeMapping) {
				if m.Kind != KindFunction {
					t.Errorf("Expected Kind %s, got %s", KindFunction, m.Kind)
				}
				if len(m.NodeTypes) != 2 {
					t.Errorf("Expected 2 NodeTypes, got %d", len(m.NodeTypes))
				}
				if m.NodeTypes[0] != "function_declaration" {
					t.Errorf("Expected first NodeType 'function_declaration', got '%s'", m.NodeTypes[0])
				}
				if m.NameCapture != "@name" {
					t.Errorf("Expected NameCapture '@name', got '%s'", m.NameCapture)
				}
				if m.Priority != 100 {
					t.Errorf("Expected Priority 100, got %d", m.Priority)
				}
			},
		},
		{
			name: "variable mapping",
			mapping: NodeMapping{
				Kind:        KindVariable,
				NodeTypes:   []string{"var_declaration", "short_var_declaration"},
				NameCapture: "@name",
				TypeCapture: "@type",
				Template:    "(var_declaration (var_spec name: (identifier) @name))",
				Attributes:  map[string]string{"mutable": "@mutable"},
				Priority:    50,
			},
			validate: func(t *testing.T, m NodeMapping) {
				if m.Kind != KindVariable {
					t.Errorf("Expected Kind %s, got %s", KindVariable, m.Kind)
				}
				if len(m.NodeTypes) != 2 {
					t.Errorf("Expected 2 NodeTypes, got %d", len(m.NodeTypes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.mapping)
		})
	}
}

func TestQueryOptionsConstruction(t *testing.T) {
	tests := []struct {
		name     string
		options  QueryOptions
		validate func(*testing.T, QueryOptions)
	}{
		{
			name: "default options",
			options: QueryOptions{
				CaseSensitive:   true,
				UseRegex:        false,
				MaxDepth:        10,
				Timeout:         5000,
				IncludeContent:  true,
				IncludeMetadata: true,
			},
			validate: func(t *testing.T, opts QueryOptions) {
				if !opts.CaseSensitive {
					t.Error("Expected CaseSensitive to be true")
				}
				if opts.UseRegex {
					t.Error("Expected UseRegex to be false")
				}
				if opts.MaxDepth != 10 {
					t.Errorf("Expected MaxDepth 10, got %d", opts.MaxDepth)
				}
				if opts.Timeout != 5000 {
					t.Errorf("Expected Timeout 5000, got %d", opts.Timeout)
				}
			},
		},
		{
			name: "regex options",
			options: QueryOptions{
				CaseSensitive:   false,
				UseRegex:        true,
				MaxDepth:        5,
				Timeout:         10000,
				IncludeContent:  false,
				IncludeMetadata: false,
			},
			validate: func(t *testing.T, opts QueryOptions) {
				if opts.CaseSensitive {
					t.Error("Expected CaseSensitive to be false")
				}
				if !opts.UseRegex {
					t.Error("Expected UseRegex to be true")
				}
				if opts.IncludeContent {
					t.Error("Expected IncludeContent to be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.options)
		})
	}
}

func TestValidationError(t *testing.T) {
	tests := []struct {
		name     string
		error    ValidationError
		validate func(*testing.T, ValidationError)
	}{
		{
			name: "field validation error",
			error: ValidationError{
				Code:    "INVALID_FIELD",
				Message: "Field 'pattern' cannot be empty",
				Field:   "pattern",
				Value:   "",
			},
			validate: func(t *testing.T, err ValidationError) {
				if err.Code != "INVALID_FIELD" {
					t.Errorf("Expected Code 'INVALID_FIELD', got '%s'", err.Code)
				}
				if err.Field != "pattern" {
					t.Errorf("Expected Field 'pattern', got '%s'", err.Field)
				}
				if err.Value != "" {
					t.Errorf("Expected empty Value, got %v", err.Value)
				}
			},
		},
		{
			name: "type validation error",
			error: ValidationError{
				Code:    "INVALID_TYPE",
				Message: "Expected string, got int",
				Field:   "type",
				Value:   42,
			},
			validate: func(t *testing.T, err ValidationError) {
				if err.Code != "INVALID_TYPE" {
					t.Errorf("Expected Code 'INVALID_TYPE', got '%s'", err.Code)
				}
				if err.Value != 42 {
					t.Errorf("Expected Value 42, got %v", err.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.error)
		})
	}
}

func TestProviderCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities ProviderCapabilities
		validate     func(*testing.T, ProviderCapabilities)
	}{
		{
			name: "full capabilities",
			capabilities: ProviderCapabilities{
				SupportedKinds:             []NodeKind{KindFunction, KindVariable, KindClass},
				SupportedScopes:            []ScopeType{ScopeFile, ScopeFunction, ScopeClass},
				SupportsRegex:              true,
				SupportsNesting:            true,
				MaxQueryDepth:              10,
				SupportsValidation:         true,
				SupportsFormatting:         true,
				SupportsImportOrganization: true,
			},
			validate: func(t *testing.T, caps ProviderCapabilities) {
				if len(caps.SupportedKinds) != 3 {
					t.Errorf("Expected 3 supported kinds, got %d", len(caps.SupportedKinds))
				}
				if !caps.SupportsRegex {
					t.Error("Expected SupportsRegex to be true")
				}
				if caps.MaxQueryDepth != 10 {
					t.Errorf("Expected MaxQueryDepth 10, got %d", caps.MaxQueryDepth)
				}
			},
		},
		{
			name: "limited capabilities",
			capabilities: ProviderCapabilities{
				SupportedKinds:             []NodeKind{KindFunction},
				SupportedScopes:            []ScopeType{ScopeFile},
				SupportsRegex:              false,
				SupportsNesting:            false,
				MaxQueryDepth:              1,
				SupportsValidation:         false,
				SupportsFormatting:         false,
				SupportsImportOrganization: false,
			},
			validate: func(t *testing.T, caps ProviderCapabilities) {
				if len(caps.SupportedKinds) != 1 {
					t.Errorf("Expected 1 supported kind, got %d", len(caps.SupportedKinds))
				}
				if caps.SupportsRegex {
					t.Error("Expected SupportsRegex to be false")
				}
				if caps.SupportsNesting {
					t.Error("Expected SupportsNesting to be false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.capabilities)
		})
	}
}

func TestQuickCheckDiagnostic(t *testing.T) {
	tests := []struct {
		name       string
		diagnostic QuickCheckDiagnostic
		validate   func(*testing.T, QuickCheckDiagnostic)
	}{
		{
			name: "error diagnostic",
			diagnostic: QuickCheckDiagnostic{
				Severity: "error",
				Message:  "Syntax error: unexpected token",
				Line:     10,
				Column:   5,
				Code:     "E001",
			},
			validate: func(t *testing.T, diag QuickCheckDiagnostic) {
				if diag.Severity != "error" {
					t.Errorf("Expected Severity 'error', got '%s'", diag.Severity)
				}
				if diag.Line != 10 {
					t.Errorf("Expected Line 10, got %d", diag.Line)
				}
				if diag.Column != 5 {
					t.Errorf("Expected Column 5, got %d", diag.Column)
				}
			},
		},
		{
			name: "warning diagnostic",
			diagnostic: QuickCheckDiagnostic{
				Severity: "warning",
				Message:  "Unused variable 'x'",
				Line:     5,
				Column:   10,
				Code:     "W001",
			},
			validate: func(t *testing.T, diag QuickCheckDiagnostic) {
				if diag.Severity != "warning" {
					t.Errorf("Expected Severity 'warning', got '%s'", diag.Severity)
				}
				if diag.Code != "W001" {
					t.Errorf("Expected Code 'W001', got '%s'", diag.Code)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.diagnostic)
		})
	}
}

// Test data structure integrity and relationships
func TestDataStructureIntegrity(t *testing.T) {
	t.Run("query with result consistency", func(t *testing.T) {
		query := Query{
			Kind:    KindFunction,
			Pattern: "test*",
			Raw:     "function:test*",
		}

		result := Result{
			Kind: query.Kind,
			Name: "testFunction", // Used to verify query-result relationship
			Metadata: map[string]any{
				"query_kind":    string(query.Kind),
				"query_pattern": query.Pattern,
			},
		}

		// Verify consistency between query and result
		if result.Kind != query.Kind {
			t.Error("Query and result kinds should match")
		}

		if queryKind, ok := result.Metadata["query_kind"]; !ok || queryKind != string(query.Kind) {
			t.Error("Result metadata should reference original query kind")
		}

		// Verify Raw field preserves original query string
		if query.Raw != "function:test*" {
			t.Errorf("Expected Raw field to be 'function:test*', got '%s'", query.Raw)
		}

		// Verify Result.Name matches Query.Name
		if result.Name != "testFunction" {
			t.Errorf("Expected Result.Name to be 'testFunction', got '%s'", result.Name)
		}
	})

	t.Run("location byte ranges", func(t *testing.T) {
		location := Location{
			StartLine: 10,
			EndLine:   15,
			StartCol:  5,
			EndCol:    20,
			StartByte: 100,
			EndByte:   150,
		}

		// Basic validation
		if location.EndLine < location.StartLine {
			t.Error("EndLine should be >= StartLine")
		}
		if location.EndByte < location.StartByte {
			t.Error("EndByte should be >= StartByte")
		}
		if location.StartLine == location.EndLine && location.EndCol < location.StartCol {
			t.Error("For same line, EndCol should be >= StartCol")
		}
	})
}

// Benchmark tests for performance-critical operations
func BenchmarkQueryCreation(b *testing.B) {
	for b.Loop() {
		_ = Query{
			Kind:       KindFunction,
			Pattern:    "test*",
			Attributes: map[string]string{"type": "public"},
			Children:   []Query{},
			Raw:        "function:test* public",
		}
	}
}

func BenchmarkResultCreation(b *testing.B) {
	location := Location{
		File:      "test.go",
		StartLine: 10,
		EndLine:   20,
		StartCol:  1,
		EndCol:    2,
		StartByte: 100,
		EndByte:   200,
	}

	for b.Loop() {
		_ = Result{
			Kind:     KindFunction,
			Name:     "testFunction",
			Location: location,
			Content:  "func testFunction() {}",
			Metadata: map[string]any{
				"visibility": "public",
				"parameters": 0,
			},
			ParentKind: KindClass,
			Scope:      ScopeClass,
		}
	}
}

func BenchmarkResultSetCreation(b *testing.B) {
	results := make([]*Result, 100)
	for i := range results {
		results[i] = &Result{
			Kind: KindFunction,
			Name: "func" + string(rune(i)),
		}
	}

	for b.Loop() {
		_ = ResultSet{
			Results:         results,
			QueryHash:       "benchmark_hash",
			TotalMatches:    len(results),
			ExecutionTimeMs: 100,
		}
	}
}

func BenchmarkQueryAttributeAccess(b *testing.B) {
	query := Query{
		Kind:    KindFunction,
		Pattern: "test*",
		Attributes: map[string]string{
			"type":       "public",
			"visibility": "exported",
			"params":     "0",
			"returns":    "1",
		},
	}

	for b.Loop() {
		_ = query.Kind
		_ = query.Pattern
		_ = query.Attributes["type"]
		_ = query.Attributes["visibility"]
		_ = query.Attributes["params"]
		_ = query.Attributes["returns"]
	}
}
