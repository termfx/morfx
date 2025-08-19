package types

import (
	"testing"
	
	"github.com/termfx/morfx/internal/core"
)

// Test DBConfig struct
func TestDBConfig(t *testing.T) {
	t.Run("create_db_config", func(t *testing.T) {
		config := &DBConfig{
			DBPath:            "/tmp/test.db",
			ActiveKeyVersion:  1,
			EncryptionKeys:    map[int][]byte{1: []byte("test-key")},
			KeyDerivationSalt: []byte("salt"),
			EncryptionMode:    "enabled",
			MasterKey:         "master-key",
			EncryptionAlgo:    "AES-256-GCM",
			RetentionRuns:     10,
		}

		if config.DBPath != "/tmp/test.db" {
			t.Errorf("Expected DBPath '/tmp/test.db', got '%s'", config.DBPath)
		}
		if config.ActiveKeyVersion != 1 {
			t.Errorf("Expected ActiveKeyVersion 1, got %d", config.ActiveKeyVersion)
		}
		if len(config.EncryptionKeys) != 1 {
			t.Errorf("Expected 1 encryption key, got %d", len(config.EncryptionKeys))
		}
		if string(config.EncryptionKeys[1]) != "test-key" {
			t.Errorf("Expected encryption key 'test-key', got '%s'", string(config.EncryptionKeys[1]))
		}
		if string(config.KeyDerivationSalt) != "salt" {
			t.Errorf("Expected salt 'salt', got '%s'", string(config.KeyDerivationSalt))
		}
		if config.EncryptionMode != "enabled" {
			t.Errorf("Expected EncryptionMode 'enabled', got '%s'", config.EncryptionMode)
		}
		if config.MasterKey != "master-key" {
			t.Errorf("Expected MasterKey 'master-key', got '%s'", config.MasterKey)
		}
		if config.EncryptionAlgo != "AES-256-GCM" {
			t.Errorf("Expected EncryptionAlgo 'AES-256-GCM', got '%s'", config.EncryptionAlgo)
		}
		if config.RetentionRuns != 10 {
			t.Errorf("Expected RetentionRuns 10, got %d", config.RetentionRuns)
		}
	})

	t.Run("empty_db_config", func(t *testing.T) {
		config := &DBConfig{}

		if config.DBPath != "" {
			t.Errorf("Expected empty DBPath, got '%s'", config.DBPath)
		}
		if config.ActiveKeyVersion != 0 {
			t.Errorf("Expected ActiveKeyVersion 0, got %d", config.ActiveKeyVersion)
		}
		if config.EncryptionKeys != nil {
			t.Errorf("Expected nil EncryptionKeys, got %v", config.EncryptionKeys)
		}
		if config.RetentionRuns != 0 {
			t.Errorf("Expected RetentionRuns 0, got %d", config.RetentionRuns)
		}
	})
}

// Test NodeKind constants
func TestNodeKind(t *testing.T) {
	t.Run("node_kind_constants", func(t *testing.T) {
		tests := []struct {
			name     string
			kind     NodeKind
			expected string
		}{
			{"function", KindFunction, "function"},
			{"variable", KindVariable, "variable"},
			{"class", KindClass, "class"},
			{"method", KindMethod, "method"},
			{"import", KindImport, "import"},
			{"constant", KindConstant, "constant"},
			{"field", KindField, "field"},
			{"call", KindCall, "call"},
			{"assignment", KindAssignment, "assignment"},
			{"condition", KindCondition, "condition"},
			{"loop", KindLoop, "loop"},
			{"block", KindBlock, "block"},
			{"comment", KindComment, "comment"},
			{"decorator", KindDecorator, "decorator"},
			{"type", KindType, "type"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if string(tt.kind) != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, string(tt.kind))
				}
			})
		}
	})
}

// Test ScopeType constants
func TestScopeType(t *testing.T) {
	t.Run("scope_type_constants", func(t *testing.T) {
		tests := []struct {
			name     string
			scope    ScopeType
			expected string
		}{
			{"file", ScopeFile, "file"},
			{"class", ScopeClass, "class"},
			{"function", ScopeFunction, "function"},
			{"block", ScopeBlock, "block"},
			{"namespace", ScopeNamespace, "namespace"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				if string(tt.scope) != tt.expected {
					t.Errorf("Expected %s, got %s", tt.expected, string(tt.scope))
				}
			})
		}
	})
}

// Test Query struct
func TestQuery(t *testing.T) {
	t.Run("create_query", func(t *testing.T) {
		query := &Query{
			Kind:       KindFunction,
			Pattern:    "test*",
			Attributes: map[string]string{"visibility": "public"},
			Operator:   "&&",
			Children:   []Query{{Kind: KindVariable, Pattern: "var*"}},
			Scope:      ScopeClass,
			Raw:        "function:test* && variable:var*",
		}

		if query.Kind != KindFunction {
			t.Errorf("Expected Kind %s, got %s", KindFunction, query.Kind)
		}
		if query.Pattern != "test*" {
			t.Errorf("Expected Pattern 'test*', got '%s'", query.Pattern)
		}
		if query.Attributes["visibility"] != "public" {
			t.Errorf("Expected visibility 'public', got '%s'", query.Attributes["visibility"])
		}
		if query.Operator != "&&" {
			t.Errorf("Expected Operator '&&', got '%s'", query.Operator)
		}
		if len(query.Children) != 1 {
			t.Errorf("Expected 1 child, got %d", len(query.Children))
		}
		if query.Children[0].Kind != KindVariable {
			t.Errorf("Expected child Kind %s, got %s", KindVariable, query.Children[0].Kind)
		}
		if query.Scope != ScopeClass {
			t.Errorf("Expected Scope %s, got %s", ScopeClass, query.Scope)
		}
		if query.Raw != "function:test* && variable:var*" {
			t.Errorf("Expected Raw 'function:test* && variable:var*', got '%s'", query.Raw)
		}
	})

	t.Run("empty_query", func(t *testing.T) {
		query := &Query{}

		if query.Kind != "" {
			t.Errorf("Expected empty Kind, got '%s'", query.Kind)
		}
		if query.Pattern != "" {
			t.Errorf("Expected empty Pattern, got '%s'", query.Pattern)
		}
		if query.Attributes != nil {
			t.Errorf("Expected nil Attributes, got %v", query.Attributes)
		}
		if len(query.Children) != 0 {
			t.Errorf("Expected 0 children, got %d", len(query.Children))
		}
	})
}

// Test Location struct
func TestLocation(t *testing.T) {
	t.Run("create_location", func(t *testing.T) {
		location := Location{
			File:      "test.go",
			StartLine: 10,
			EndLine:   20,
			StartCol:  5,
			EndCol:    15,
		}

		if location.File != "test.go" {
			t.Errorf("Expected File 'test.go', got '%s'", location.File)
		}
		if location.StartLine != 10 {
			t.Errorf("Expected StartLine 10, got %d", location.StartLine)
		}
		if location.EndLine != 20 {
			t.Errorf("Expected EndLine 20, got %d", location.EndLine)
		}
		if location.StartCol != 5 {
			t.Errorf("Expected StartCol 5, got %d", location.StartCol)
		}
		if location.EndCol != 15 {
			t.Errorf("Expected EndCol 15, got %d", location.EndCol)
		}
	})

	t.Run("zero_location", func(t *testing.T) {
		location := Location{}

		if location.File != "" {
			t.Errorf("Expected empty File, got '%s'", location.File)
		}
		if location.StartLine != 0 {
			t.Errorf("Expected StartLine 0, got %d", location.StartLine)
		}
		if location.EndLine != 0 {
			t.Errorf("Expected EndLine 0, got %d", location.EndLine)
		}
	})
}

// Test Result struct
func TestResult(t *testing.T) {
	t.Run("create_result", func(t *testing.T) {
		location := Location{
			File:      "test.go",
			StartLine: 1,
			EndLine:   5,
			StartCol:  0,
			EndCol:    10,
		}

		metadata := map[string]any{
			"visibility": "public",
			"type":       "string",
		}

		result := &Result{
			Result: &core.Result{
				Kind:     KindFunction,
				Name:     "testFunc",
				Location: location,
				Metadata: metadata,
			},
			Node: nil, // We'll test with nil since we don't have a real sitter.Node
		}

		if result.Node != nil {
			t.Errorf("Expected nil Node, got %v", result.Node)
		}
		if result.Result.Kind != KindFunction {
			t.Errorf("Expected Kind %s, got %s", KindFunction, result.Result.Kind)
		}
		if result.Result.Name != "testFunc" {
			t.Errorf("Expected Name 'testFunc', got '%s'", result.Result.Name)
		}
		if result.Result.Location.File != "test.go" {
			t.Errorf("Expected Location.File 'test.go', got '%s'", result.Result.Location.File)
		}
		if result.Result.Metadata["visibility"] != "public" {
			t.Errorf("Expected visibility 'public', got '%v'", result.Result.Metadata["visibility"])
		}
		if result.Result.Metadata["type"] != "string" {
			t.Errorf("Expected type 'string', got '%v'", result.Result.Metadata["type"])
		}
	})

	t.Run("empty_result", func(t *testing.T) {
		result := &Result{}

		if result.Result != nil {
			t.Errorf("Expected nil Result, got %v", result.Result)
		}
	})
}

// Test ResultSet functionality
func TestResultSet(t *testing.T) {
	t.Run("add_and_get_results", func(t *testing.T) {
		rs := NewResultSet()

		result1 := &Result{
			Result: &core.Result{
				Kind:     KindFunction,
				Name:     "func1",
				Location: Location{File: "test1.go", StartLine: 1},
			},
		}
		result2 := &Result{
			Result: &core.Result{
				Kind:     KindVariable,
				Name:     "var1",
				Location: Location{File: "test2.go", StartLine: 5},
			},
		}

		rs.Add(result1)
		rs.Add(result2)

		if rs.Count() != 2 {
			t.Errorf("Expected count 2, got %d", rs.Count())
		}

		results := rs.All()
		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		if results[0].Result.Name != "func1" {
			t.Errorf("Expected first result name 'func1', got '%s'", results[0].Result.Name)
		}
		if results[1].Result.Name != "var1" {
			t.Errorf("Expected second result name 'var1', got '%s'", results[1].Result.Name)
		}
	})

	t.Run("empty_result_set", func(t *testing.T) {
		rs := NewResultSet()

		if rs.Count() != 0 {
			t.Errorf("Expected count 0, got %d", rs.Count())
		}

		results := rs.All()
		if len(results) != 0 {
			t.Errorf("Expected 0 results, got %d", len(results))
		}
	})

	t.Run("filter_results", func(t *testing.T) {
		rs := NewResultSet()

		result1 := &Result{Result: &core.Result{Kind: KindFunction, Name: "func1"}}
		result2 := &Result{Result: &core.Result{Kind: KindVariable, Name: "var1"}}
		result3 := &Result{Result: &core.Result{Kind: KindFunction, Name: "func2"}}

		rs.Add(result1)
		rs.Add(result2)
		rs.Add(result3)

		// Filter for functions only
		filteredRs := rs.Filter(func(r *Result) bool {
			return r.Result.Kind == KindFunction
		})

		if filteredRs.Count() != 2 {
			t.Errorf("Expected filtered count 2, got %d", filteredRs.Count())
		}

		filteredResults := filteredRs.All()
		for _, result := range filteredResults {
			if result.Result.Kind != KindFunction {
				t.Errorf("Expected all filtered results to be functions, got %s", result.Result.Kind)
			}
		}
	})

	t.Run("merge_result_sets", func(t *testing.T) {
		rs1 := NewResultSet()
		rs2 := NewResultSet()

		result1 := &Result{Result: &core.Result{Kind: KindFunction, Name: "func1"}}
		result2 := &Result{Result: &core.Result{Kind: KindVariable, Name: "var1"}}
		result3 := &Result{Result: &core.Result{Kind: KindClass, Name: "class1"}}

		rs1.Add(result1)
		rs1.Add(result2)
		rs2.Add(result3)

		mergedRs := rs1.Merge(rs2)

		if mergedRs.Count() != 3 {
			t.Errorf("Expected merged count 3, got %d", mergedRs.Count())
		}

		mergedResults := mergedRs.All()
		expectedNames := []string{"func1", "var1", "class1"}
		for i, result := range mergedResults {
			if result.Result.Name != expectedNames[i] {
				t.Errorf("Expected result %d name '%s', got '%s'", i, expectedNames[i], result.Result.Name)
			}
		}
	})

	t.Run("merge_empty_result_sets", func(t *testing.T) {
		rs1 := NewResultSet()
		rs2 := NewResultSet()

		mergedRs := rs1.Merge(rs2)

		if mergedRs.Count() != 0 {
			t.Errorf("Expected merged count 0, got %d", mergedRs.Count())
		}
	})

	t.Run("filter_with_no_matches", func(t *testing.T) {
		rs := NewResultSet()

		result1 := &Result{Result: &core.Result{Kind: KindFunction, Name: "func1"}}
		result2 := &Result{Result: &core.Result{Kind: KindVariable, Name: "var1"}}

		rs.Add(result1)
		rs.Add(result2)

		// Filter for classes (none exist)
		filteredRs := rs.Filter(func(r *Result) bool {
			return r.Result.Kind == KindClass
		})

		if filteredRs.Count() != 0 {
			t.Errorf("Expected filtered count 0, got %d", filteredRs.Count())
		}
	})
}

// Mock implementation of GlobalConfig for testing
type mockGlobalConfig struct {
	dbConfig *DBConfig
}

func (m *mockGlobalConfig) GetDBConfig() *DBConfig {
	return m.dbConfig
}

func TestGlobalConfig(t *testing.T) {
	t.Run("mock_global_config", func(t *testing.T) {
		dbConfig := &DBConfig{
			DBPath:         "/test/path",
			EncryptionMode: "enabled",
		}

		mock := &mockGlobalConfig{dbConfig: dbConfig}
		retrieved := mock.GetDBConfig()

		if retrieved != dbConfig {
			t.Error("Expected same DBConfig instance")
		}
		if retrieved.DBPath != "/test/path" {
			t.Errorf("Expected DBPath '/test/path', got '%s'", retrieved.DBPath)
		}
		if retrieved.EncryptionMode != "enabled" {
			t.Errorf("Expected EncryptionMode 'enabled', got '%s'", retrieved.EncryptionMode)
		}
	})

	t.Run("nil_db_config", func(t *testing.T) {
		mock := &mockGlobalConfig{dbConfig: nil}
		retrieved := mock.GetDBConfig()

		if retrieved != nil {
			t.Error("Expected nil DBConfig")
		}
	})
}
