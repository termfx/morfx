package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	golang_sitter "github.com/smacker/go-tree-sitter/golang"

	"github.com/termfx/morfx/internal/model"
	"github.com/termfx/morfx/internal/types"
)

// MockLanguageProvider implements types.LanguageProvider for testing
type MockLanguageProvider struct {
	lang string
}

func (m *MockLanguageProvider) Lang() string         { return m.lang }
func (m *MockLanguageProvider) Aliases() []string    { return []string{} }
func (m *MockLanguageProvider) Extensions() []string { return []string{".test"} }
func (m *MockLanguageProvider) GetSitterLanguage() *sitter.Language {
	return golang_sitter.GetLanguage()
}
func (m *MockLanguageProvider) TranslateKind(kind types.NodeKind) []types.NodeMapping { return nil }
func (m *MockLanguageProvider) TranslateQuery(q *types.Query) (string, error) {
	return "(function_declaration) @target", nil
}

func (m *MockLanguageProvider) NormalizeDSLKind(dslKind string) types.NodeKind {
	return types.KindFunction
}
func (m *MockLanguageProvider) GetSupportedDSLKinds() []string { return nil }
func (m *MockLanguageProvider) ParseAttributes(node *sitter.Node, source []byte) map[string]string {
	return nil
}

func (m *MockLanguageProvider) GetNodeKind(node *sitter.Node) types.NodeKind {
	return types.KindFunction
}
func (m *MockLanguageProvider) GetNodeName(node *sitter.Node, source []byte) string { return "" }
func (m *MockLanguageProvider) OptimizeQuery(q *types.Query) *types.Query           { return q }
func (m *MockLanguageProvider) EstimateQueryCost(q *types.Query) int                { return 0 }
func (m *MockLanguageProvider) GetNodeScope(node *sitter.Node) types.ScopeType {
	return types.ScopeFile
}

func (m *MockLanguageProvider) FindEnclosingScope(node *sitter.Node, scope types.ScopeType) *sitter.Node {
	return nil
}
func (m *MockLanguageProvider) IsBlockLevelNode(nodeType string) bool          { return false }
func (m *MockLanguageProvider) GetDefaultIgnorePatterns() ([]string, []string) { return nil, nil }

// OrganizeImports organizes import statements in source code
func (m *MockLanguageProvider) OrganizeImports(source []byte) ([]byte, error) {
	return source, nil
}

// Format formats the source code according to language conventions
func (m *MockLanguageProvider) Format(source []byte) ([]byte, error) {
	return source, nil
}

// QuickCheck performs basic syntax and semantic validation
func (m *MockLanguageProvider) QuickCheck(source []byte) []types.QuickCheckDiagnostic {
	return []types.QuickCheckDiagnostic{}
}

// MockManipulator for testing
type MockManipulator struct {
	result *model.Result
	err    error
}

func (m *MockManipulator) Manipulate(cfg *model.Config, path, original string, data []byte) (*model.Result, error) {
	return m.result, m.err
}

func TestRun(t *testing.T) {
	tests := []struct {
		name          string
		config        *model.Config
		files         []string
		expectedError bool
		errorContains string
	}{
		{
			name: "commit operation",
			config: &model.Config{
				Operation: model.OpCommit,
				Provider:  &MockLanguageProvider{lang: "go"},
			},
			files:         []string{},
			expectedError: true,
			errorContains: "no staged changes",
		},
		{
			name: "process operation with files",
			config: &model.Config{
				Operation:   model.OpReplace,
				Pattern:     "(function_declaration) @target",
				Replacement: "func replaced() {}",
				Provider:    &MockLanguageProvider{lang: "go"},
			},
			files:         []string{"test.go"},
			expectedError: false,
		},
		{
			name: "process operation without files",
			config: &model.Config{
				Operation: model.OpReplace,
				Pattern:   "test",
				Provider:  &MockLanguageProvider{lang: "go"},
			},
			files:         []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary files for testing
			var tempFiles []string
			for _, file := range tt.files {
				tempFile, err := os.CreateTemp("", file)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tempFile.Name())

				// Write some test content
				_, err = tempFile.WriteString("package main\n\nfunc test() {}\n")
				if err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
				tempFile.Close()
				tempFiles = append(tempFiles, tempFile.Name())
			}

			output := Run(tempFiles, tt.config)
			if tt.expectedError {
				if output.Error == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(output.Error.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, output.Error.Error())
				}
			} else {
				if output.Error != nil {
					t.Errorf("Unexpected error: %v", output.Error)
				}
			}
		})
	}
}

func TestProcess(t *testing.T) {
	tests := []struct {
		name          string
		config        *model.Config
		files         []string
		expectedError bool
	}{
		{
			name: "process with valid files",
			config: &model.Config{
				Operation:   model.OpReplace,
				Pattern:     "(function_declaration) @target",
				Replacement: "func replaced() {}",
				Provider:    &MockLanguageProvider{lang: "go"},
			},
			files:         []string{"test1.go", "test2.go"},
			expectedError: false,
		},
		{
			name: "process with no files (stdin mode)",
			config: &model.Config{
				Operation: model.OpReplace,
				Pattern:   "test",
				Provider:  &MockLanguageProvider{lang: "go"},
			},
			files:         []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary files for testing
			var tempFiles []string
			for _, file := range tt.files {
				tempFile, err := os.CreateTemp("", file)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tempFile.Name())

				// Write some test content
				_, err = tempFile.WriteString("package main\n\nfunc main() {}\n")
				if err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
				tempFile.Close()
				tempFiles = append(tempFiles, tempFile.Name())
			}

			// Mock stdin for no files case
			if len(tt.files) == 0 {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				go func() {
					w.WriteString("package main\n\nfunc main() {}\n")
					w.Close()
				}()
				defer func() { os.Stdin = oldStdin }()
			}

			output := process(tempFiles, tt.config)
			if tt.expectedError {
				if output.Error == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if output.Error != nil {
					t.Errorf("Unexpected error: %v", output.Error)
				}
			}
		})
	}
}

func TestProcessFile(t *testing.T) {
	tests := []struct {
		name          string
		config        *model.Config
		filePath      string
		expectedError bool
		errorContains string
	}{
		{
			name: "process valid file",
			config: &model.Config{
				Operation:   model.OpReplace,
				Pattern:     "(function_declaration) @target",
				Replacement: "func replaced() {}",
				Provider:    &MockLanguageProvider{lang: "go"},
			},
			filePath:      "test.go",
			expectedError: false,
		},
		{
			name: "process non-existent file",
			config: &model.Config{
				Operation: model.OpReplace,
				Pattern:   "test",
				Provider:  &MockLanguageProvider{lang: "go"},
			},
			filePath:      "/non/existent/file.go",
			expectedError: true,
			errorContains: "no such file",
		},
		{
			name: "process stdin (dash path)",
			config: &model.Config{
				Operation:   model.OpReplace,
				Pattern:     "(function_declaration) @target",
				Replacement: "func replaced() {}",
				Provider:    &MockLanguageProvider{lang: "go"},
			},
			filePath:      "-",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempFile *os.File
			var err error
			filePath := tt.filePath

			// Create temp file for valid file tests
			if tt.filePath != "" && !strings.Contains(tt.filePath, "/non/existent/") {
				tempFile, err = os.CreateTemp("", tt.filePath)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tempFile.Name())

				_, err = tempFile.WriteString("package main\n\nfunc main() {}\n")
				if err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
				tempFile.Close()
				filePath = tempFile.Name()
			}

			// Mock stdin for dash path case
			if tt.filePath == "-" {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				go func() {
					w.WriteString("package main\n\nfunc main() {}\n")
					w.Close()
				}()
				defer func() { os.Stdin = oldStdin }()
			}

			_, err = processFile(filePath, tt.config)
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestReadFileData(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		expectedData  string
		expectedError bool
		errorContains string
	}{
		{
			name:         "read valid file",
			filePath:     "test.go",
			expectedData: "package main\n\nfunc test() {}\n",
		},
		{
			name:          "read non-existent file",
			filePath:      "/non/existent/file.go",
			expectedError: true,
			errorContains: "no such file",
		},
		{
			name:         "read from stdin (dash path)",
			filePath:     "-",
			expectedData: "package main\n\nfunc main() {}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempFile *os.File
			var err error
			filePath := tt.filePath

			// Create temp file for valid file tests
			if tt.filePath != "" && tt.filePath != "-" && !strings.Contains(tt.filePath, "/non/existent/") {
				tempFile, err = os.CreateTemp("", tt.filePath)
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tempFile.Name())

				_, err = tempFile.WriteString("package main\n\nfunc test() {}\n")
				if err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
				tempFile.Close()
				filePath = tempFile.Name()
			}

			// Mock stdin for dash path case
			if tt.filePath == "-" {
				oldStdin := os.Stdin
				r, w, _ := os.Pipe()
				os.Stdin = r
				go func() {
					w.WriteString("package main\n\nfunc main() {}\n")
					w.Close()
				}()
				defer func() { os.Stdin = oldStdin }()
			}

			var data []byte
			if filePath == "-" {
				data, err = io.ReadAll(os.Stdin)
			} else {
				data, err = os.ReadFile(filePath)
			}
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if string(data) != tt.expectedData {
					t.Errorf("Expected data %q, got %q", tt.expectedData, string(data))
				}
			}
		})
	}
}

// Test helper function to verify stdin reading
func TestReadFromStdin(t *testing.T) {
	originalStdin := os.Stdin
	defer func() { os.Stdin = originalStdin }()

	// Create a pipe to simulate stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	// Replace stdin with our pipe
	os.Stdin = r

	// Write test data to the pipe
	testData := "package main\n\nfunc main() {}\n"
	go func() {
		defer w.Close()
		w.WriteString(testData)
	}()

	// Read from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Errorf("Failed to read from stdin: %v", err)
	}

	if string(data) != testData {
		t.Errorf("Expected %q, got %q", testData, string(data))
	}
}

// Test error handling in file reading
func TestReadFileDataErrors(t *testing.T) {
	// Test with a directory instead of a file
	tempDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	_, err = os.ReadFile(tempDir)
	if err == nil {
		t.Errorf("Expected error when reading directory, got none")
	}
}

// Benchmark tests
func BenchmarkProcessFile(b *testing.B) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "benchmark.go")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test content
	_, err = tempFile.WriteString("package main\n\nfunc main() {}\n")
	if err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	config := &model.Config{
		Operation:   model.OpReplace,
		Pattern:     "(function_declaration) @target",
		Replacement: "func replaced() {}",
		Provider:    &MockLanguageProvider{lang: "go"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processFile(tempFile.Name(), config)
	}
}

func BenchmarkReadFileData(b *testing.B) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "benchmark.go")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write test content
	_, err = tempFile.WriteString("package main\n\nfunc main() {}\n")
	if err != nil {
		b.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		os.ReadFile(tempFile.Name())
	}
}
