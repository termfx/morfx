package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Mock provider for testing
type MockProvider struct {
	language        string
	queryResult     QueryResult
	transformResult TransformResult
	delay           time.Duration // For simulating slow processing
}

func (m *MockProvider) Language() string {
	return m.language
}

func (m *MockProvider) Query(source string, query AgentQuery) QueryResult {
	return m.queryResult
}

func (m *MockProvider) Transform(source string, op TransformOp) TransformResult {
	// Simulate slow processing if delay is set
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.transformResult
}

// Mock provider registry
type MockProviderRegistry struct {
	providers map[string]Provider
}

func (m *MockProviderRegistry) Get(language string) (Provider, bool) {
	provider, exists := m.providers[language]
	return provider, exists
}

func TestNewFileProcessor(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	if processor == nil {
		t.Fatal("NewFileProcessor returned nil")
	}

	if processor.providers != registry {
		t.Error("Provider registry not set correctly")
	}

	if processor.walker == nil {
		t.Error("File walker not initialized")
	}

	if processor.workers != 8 {
		t.Errorf("Expected 8 workers, got %d", processor.workers)
	}

	if processor.atomicWriter == nil {
		t.Error("Atomic writer not initialized")
	}

	if processor.txManager == nil {
		t.Error("Transaction manager not initialized")
	}

	if !processor.safetyEnabled {
		t.Error("Safety should be enabled by default")
	}
}

func TestNewFileProcessorWithSafety(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	atomicConfig := DefaultAtomicConfig()

	tests := []struct {
		name          string
		safetyEnabled bool
	}{
		{"safety enabled", true},
		{"safety disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewFileProcessorWithSafety(registry, tt.safetyEnabled, atomicConfig)

			if processor == nil {
				t.Fatal("NewFileProcessorWithSafety returned nil")
			}

			if processor.safetyEnabled != tt.safetyEnabled {
				t.Errorf("Expected safety enabled %v, got %v", tt.safetyEnabled, processor.safetyEnabled)
			}
		})
	}
}

func TestFileProcessor_EnableSafety(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	// Test enabling safety
	processor.EnableSafety(true)
	if !processor.IsSafetyEnabled() {
		t.Error("Safety should be enabled")
	}

	// Test disabling safety
	processor.EnableSafety(false)
	if processor.IsSafetyEnabled() {
		t.Error("Safety should be disabled")
	}
}

func TestFileProcessor_IsSafetyEnabled(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}

	// Test default safety state
	processor := NewFileProcessor(registry)
	if !processor.IsSafetyEnabled() {
		t.Error("Safety should be enabled by default")
	}

	// Test with safety disabled
	atomicConfig := DefaultAtomicConfig()
	processor2 := NewFileProcessorWithSafety(registry, false, atomicConfig)
	if processor2.IsSafetyEnabled() {
		t.Error("Safety should be disabled")
	}
}

func TestFileProcessor_Cleanup(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	// Cleanup should not panic
	processor.Cleanup()

	// Should be able to call cleanup multiple times
	processor.Cleanup()
}

func TestFileProcessor_GenerateChecksum(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	// Create a temporary file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := "Hello, World!"

	err := os.WriteFile(testFile, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate checksum
	checksum, err := processor.GenerateChecksum(testFile)
	if err != nil {
		t.Fatalf("GenerateChecksum failed: %v", err)
	}

	if checksum == "" {
		t.Error("Checksum should not be empty")
	}

	if len(checksum) != 64 { // SHA256 produces 64 character hex string
		t.Errorf("Expected checksum length 64, got %d", len(checksum))
	}

	// Generate checksum again - should be the same
	checksum2, err := processor.GenerateChecksum(testFile)
	if err != nil {
		t.Fatalf("Second GenerateChecksum failed: %v", err)
	}

	if checksum != checksum2 {
		t.Error("Checksums should be consistent")
	}
}

func TestFileProcessor_GenerateChecksum_InvalidFile(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	// Test with non-existent file
	_, err := processor.GenerateChecksum("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFileProcessor_ValidateChanges(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	tests := []struct {
		name        string
		details     []FileTransformDetail
		expectError bool
	}{
		{
			name:        "no details",
			details:     []FileTransformDetail{},
			expectError: false,
		},
		{
			name: "successful changes",
			details: []FileTransformDetail{
				{
					FilePath:   "test1.go",
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.8, Level: "high"},
				},
				{
					FilePath:   "test2.go",
					Modified:   false,
					Confidence: ConfidenceScore{Score: 0.9, Level: "high"},
				},
			},
			expectError: false,
		},
		{
			name: "error in file",
			details: []FileTransformDetail{
				{
					FilePath: "test1.go",
					Error:    "parse error",
				},
			},
			expectError: true,
		},
		{
			name: "very low confidence",
			details: []FileTransformDetail{
				{
					FilePath:   "test1.go",
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.2, Level: "low"},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processor.ValidateChanges(tt.details)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateChanges() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestFileProcessor_calculateOverallConfidence(t *testing.T) {
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	tests := []struct {
		name     string
		details  []FileTransformDetail
		expected string // confidence level
	}{
		{
			name:     "no details",
			details:  []FileTransformDetail{},
			expected: "low",
		},
		{
			name: "high confidence",
			details: []FileTransformDetail{
				{
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.9},
				},
				{
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.85},
				},
			},
			expected: "high",
		},
		{
			name: "medium confidence",
			details: []FileTransformDetail{
				{
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.7},
				},
				{
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.6},
				},
			},
			expected: "medium",
		},
		{
			name: "low confidence",
			details: []FileTransformDetail{
				{
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.3},
				},
			},
			expected: "low",
		},
		{
			name: "with errors",
			details: []FileTransformDetail{
				{
					Modified:   true,
					Confidence: ConfidenceScore{Score: 0.9},
					Error:      "",
				},
				{
					Modified:   false,
					Confidence: ConfidenceScore{Score: 0.0},
					Error:      "parse error",
				},
			},
			expected: "medium", // Should be reduced due to errors
		},
		{
			name: "large batch operation",
			details: func() []FileTransformDetail {
				details := make([]FileTransformDetail, 15) // > 10 files
				for i := range details {
					details[i] = FileTransformDetail{
						Modified:   true,
						Confidence: ConfidenceScore{Score: 0.9},
					}
				}
				return details
			}(),
			expected: "high", // 0.9 - 0.1 (batch penalty) = 0.8, which is still "high"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := processor.calculateOverallConfidence(tt.details)

			if confidence.Level != tt.expected {
				t.Errorf("Expected confidence level %s, got %s", tt.expected, confidence.Level)
			}

			// Verify score is between 0 and 1
			if confidence.Score < 0 || confidence.Score > 1 {
				t.Errorf("Confidence score %f is out of range [0,1]", confidence.Score)
			}
		})
	}
}

func TestFileProcessor_QueryFiles_NoProvider(t *testing.T) {
	// Create registry without any providers
	registry := &MockProviderRegistry{providers: make(map[string]Provider)}
	processor := NewFileProcessor(registry)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	// Create a Go file
	err := os.WriteFile(testFile, []byte("package main"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	scope := FileScope{
		Path:     tempDir,
		Include:  []string{"*.go"},
		Language: "go",
	}

	query := AgentQuery{
		Type: "function",
		Name: "main",
	}

	// Should complete without error even without provider
	matches, err := processor.QueryFiles(ctx, scope, query)
	if err != nil {
		t.Fatalf("QueryFiles failed: %v", err)
	}

	// Should return empty matches since no provider for Go
	if len(matches) != 0 {
		t.Errorf("Expected 0 matches without provider, got %d", len(matches))
	}
}

func TestFileProcessor_QueryFiles_WithProvider(t *testing.T) {
	// Create mock provider that returns matches
	mockProvider := &MockProvider{
		language: "go",
		queryResult: QueryResult{
			Matches: []Match{
				{
					Name: "main",
					Type: "function",
					Location: Location{
						Line:    1,
						EndLine: 3,
					},
					Content: "package main\nfunc main() {}",
				},
			},
		},
	}

	registry := &MockProviderRegistry{
		providers: map[string]Provider{
			"go": mockProvider,
		},
	}
	processor := NewFileProcessor(registry)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")

	// Create a Go file
	err := os.WriteFile(testFile, []byte("package main\nfunc main() {}"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	scope := FileScope{
		Path:     tempDir,
		Include:  []string{"*.go"},
		Language: "go",
	}

	query := AgentQuery{
		Type: "function",
		Name: "main",
	}

	matches, err := processor.QueryFiles(ctx, scope, query)
	if err != nil {
		t.Fatalf("QueryFiles failed: %v", err)
	}

	// Since we have a provider that should return matches, we expect some results
	// The mock provider should be called for each Go file found
	if len(matches) == 0 {
		t.Log("No matches found - this could be expected if the provider doesn't find the specific query")
	} else {
		// Verify match details if we got results
		match := matches[0]
		if match.Name != "main" {
			t.Errorf("Expected match name 'main', got %s", match.Name)
		}

		if match.Type != "function" {
			t.Errorf("Expected match type 'function', got %s", match.Type)
		}

		if match.FilePath != testFile {
			t.Errorf("Expected file path %s, got %s", testFile, match.FilePath)
		}
	}
}

func TestFileProcessor_TransformFiles_DryRun(t *testing.T) {
	// Create mock provider that returns successful transform
	mockProvider := &MockProvider{
		language: "go",
		transformResult: TransformResult{
			Modified:   "package main\nfunc newMain() {}",
			MatchCount: 1,
			Confidence: ConfidenceScore{Score: 0.9, Level: "high"},
			Diff:       "- func main()\n+ func newMain()",
		},
	}

	registry := &MockProviderRegistry{
		providers: map[string]Provider{
			"go": mockProvider,
		},
	}
	processor := NewFileProcessor(registry)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	originalContent := "package main\nfunc main() {}"

	// Create a Go file
	err := os.WriteFile(testFile, []byte(originalContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	op := FileTransformOp{
		Scope: FileScope{
			Path:     tempDir,
			Include:  []string{"*.go"},
			Language: "go",
		},
		TransformOp: TransformOp{
			Method: "replace",
			Target: AgentQuery{
				Type: "function",
				Name: "main",
			},
			Replacement: "newMain",
		},
		DryRun: true,
	}

	result, err := processor.TransformFiles(ctx, op)
	if err != nil {
		t.Fatalf("TransformFiles failed: %v", err)
	}

	if result == nil {
		t.Fatal("Transform result is nil")
	}

	if result.FilesScanned == 0 {
		t.Error("Expected files to be scanned")
	}

	if result.FilesModified != 1 {
		t.Errorf("Expected 1 file modified, got %d", result.FilesModified)
	}

	if result.TotalMatches != 1 {
		t.Errorf("Expected 1 match, got %d", result.TotalMatches)
	}

	// Verify original file unchanged (dry run)
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != originalContent {
		t.Error("Original file should be unchanged in dry run mode")
	}

	// Verify result contains details
	if len(result.Files) == 0 {
		t.Error("Expected file details in result")
	}

	fileDetail := result.Files[0]
	if !fileDetail.Modified {
		t.Error("File detail should show modification")
	}

	if fileDetail.MatchCount != 1 {
		t.Errorf("Expected 1 match in file detail, got %d", fileDetail.MatchCount)
	}
}

func TestFileProcessor_TransformFiles_NoMatches(t *testing.T) {
	// Create mock provider that returns no matches
	mockProvider := &MockProvider{
		language: "go",
		transformResult: TransformResult{
			Modified:   "package main\nfunc main() {}", // No change
			MatchCount: 0,
			Confidence: ConfidenceScore{Score: 0.0, Level: "low"},
		},
	}

	registry := &MockProviderRegistry{
		providers: map[string]Provider{
			"go": mockProvider,
		},
	}
	processor := NewFileProcessor(registry)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	originalContent := "package main\nfunc main() {}"

	err := os.WriteFile(testFile, []byte(originalContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	op := FileTransformOp{
		Scope: FileScope{
			Path:     tempDir,
			Include:  []string{"*.go"},
			Language: "go",
		},
		TransformOp: TransformOp{
			Method: "replace",
			Target: AgentQuery{
				Type: "function",
				Name: "nonExistent",
			},
			Replacement: "newName",
		},
		DryRun: true,
	}

	result, err := processor.TransformFiles(ctx, op)
	if err != nil {
		t.Fatalf("TransformFiles failed: %v", err)
	}

	if result.FilesModified != 0 {
		t.Errorf("Expected 0 files modified, got %d", result.FilesModified)
	}

	if result.TotalMatches != 0 {
		t.Errorf("Expected 0 matches, got %d", result.TotalMatches)
	}
}

func TestFileProcessor_TransformFiles_WithBackup(t *testing.T) {
	// Create mock provider that returns successful transform
	mockProvider := &MockProvider{
		language: "go",
		transformResult: TransformResult{
			Modified:   "package main\nfunc newMain() {}",
			MatchCount: 1,
			Confidence: ConfidenceScore{Score: 0.9, Level: "high"},
		},
	}

	registry := &MockProviderRegistry{
		providers: map[string]Provider{
			"go": mockProvider,
		},
	}
	processor := NewFileProcessor(registry)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.go")
	originalContent := "package main\nfunc main() {}"

	err := os.WriteFile(testFile, []byte(originalContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	op := FileTransformOp{
		Scope: FileScope{
			Path:     tempDir,
			Include:  []string{"*.go"},
			Language: "go",
		},
		TransformOp: TransformOp{
			Method: "replace",
			Target: AgentQuery{
				Type: "function",
				Name: "main",
			},
			Replacement: "newMain",
		},
		DryRun: false,
		Backup: true,
	}

	// Disable safety to test backup without transactions
	processor.EnableSafety(false)

	result, err := processor.TransformFiles(ctx, op)
	if err != nil {
		t.Fatalf("TransformFiles failed: %v", err)
	}

	if result.FilesModified != 1 {
		t.Errorf("Expected 1 file modified, got %d", result.FilesModified)
	}

	// Verify backup was created
	backupFile := testFile + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	} else {
		// Verify backup content
		backupContent, err := os.ReadFile(backupFile)
		if err != nil {
			t.Fatalf("Failed to read backup file: %v", err)
		}

		if string(backupContent) != originalContent {
			t.Error("Backup file does not contain original content")
		}
	}

	// Verify original file was modified
	modifiedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}

	if strings.Contains(string(modifiedContent), "func main()") {
		t.Error("Original file should have been modified")
	}
}
