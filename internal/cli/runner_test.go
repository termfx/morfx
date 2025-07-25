package cli_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/garaekz/fileman/internal/cli"
	"github.com/garaekz/fileman/internal/model"
)

// Helper to capture stdout/stderr
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	f()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	out, _ := io.ReadAll(rOut)
	err, _ := io.ReadAll(rErr)
	return string(out) + string(err)
}

func TestRunWithConfig_InvalidConfigPath(t *testing.T) {
	r := &cli.Runner{}
	output := captureOutput(func() {
		exitCode := r.RunWithConfig("non_existent_config.json")
		if exitCode != 1 {
			t.Errorf("Expected exit code 1 for invalid config path, got %d", exitCode)
		}
	})

	if !strings.Contains(output, "Error: error reading config file:") {
		t.Errorf("Expected 'Error: error reading config file:' in output, got %q", output)
	}
}

func TestRunWithConfig_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	invalidJSONPath := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(invalidJSONPath, []byte("{invalid json"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON file: %v", err)
	}

	r := &cli.Runner{}
	output := captureOutput(func() {
		exitCode := r.RunWithConfig(invalidJSONPath)
		if exitCode != 1 {
			t.Errorf("Expected exit code 1 for invalid JSON, got %d", exitCode)
		}
	})

	if !strings.Contains(output, "Error: error parsing config file:") {
		t.Errorf("Expected 'Error: error parsing config file:' in output, got %q", output)
	}
}

func TestRunWithConfig_UnsupportedSchemaVersion(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "unsupported_schema.json")
	content := fmt.Sprintf(
		`{"schema_version": %d, "files": ["a.go"], "rules": [{"rule_id": "test", "pattern": "a", "replacement": "b", "operation": "replace"}]}`,
		model.CurrentSchemaVersion+1,
	)
	err := os.WriteFile(configPath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	r := &cli.Runner{}
	output := captureOutput(func() {
		exitCode := r.RunWithConfig(configPath)
		if exitCode != 1 {
			t.Errorf("Expected exit code 1 for unsupported schema version, got %d", exitCode)
		}
	})

	if !strings.Contains(output, "is not supported by tool version") {
		t.Errorf("Expected 'is not supported by tool version' in output, got %q", output)
	}
}

func TestRunWithConfig_NoFilesOrRules(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "no_files_rules.json")

	// Test case: No files
	err := os.WriteFile(
		configPath,
		[]byte(
			`{"schema_version": 1, "rules": [{"rule_id": "test", "pattern": "a", "replacement": "b", "operation": "replace"}]}`,
		),
		0o644,
	)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	r := &cli.Runner{}
	output := captureOutput(func() {
		exitCode := r.RunWithConfig(configPath)
		if exitCode != 2 {
			t.Errorf("Expected exit code 2 for no files, got %d", exitCode)
		}
	})
	if !strings.Contains(output, "Error: config must specify at least one file and one rule.") {
		t.Errorf(
			"Expected 'Error: config must specify at least one file and one rule.' in output, got %q",
			output,
		)
	}

	// Test case: No rules
	err = os.WriteFile(configPath, []byte(`{"schema_version": 1, "files": ["a.go"]}`), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	output = captureOutput(func() {
		exitCode := r.RunWithConfig(configPath)
		if exitCode != 2 {
			t.Errorf("Expected exit code 2 for no rules, got %d", exitCode)
		}
	})
	if !strings.Contains(output, "Error: config must specify at least one file and one rule.") {
		t.Errorf(
			"Expected 'Error: config must specify at least one file and one rule.' in output, got %q",
			output,
		)
	}
}

func TestRunWithFlags_LiteralPattern(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "literal_test.txt")
	err := os.WriteFile(
		testFilePath,
		[]byte("func main() {\n\tfmt.Println(\"Hello, World!\")\n}"),
		0o644,
	)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test with a pattern that contains regex metacharacters, used as literal
	cfg := model.ModificationConfig{
		RuleID:         "literal-test",
		Pattern:        "fmt.Println(\"Hello, World!\")",
		Replacement:    "fmt.Println(\"Goodbye, World!\")",
		Operation:      model.OpReplace,
		Occurrences:    "all",
		LiteralPattern: true,
	}

	r := &cli.Runner{Verbose: true}
	output := captureOutput(func() {
		exitCode := r.RunWithFlags([]string{testFilePath}, cfg, false)
		if exitCode != 0 {
			t.Errorf("Expected exit code 0 for successful literal pattern match, got %d", exitCode)
		}
	})

	if !strings.Contains(output, "✓") || !strings.Contains(output, "1 changes") {
		t.Errorf("Expected success message for literal pattern, got %q", output)
	}

	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	expected := "func main() {\n\tfmt.Println(\"Goodbye, World!\")\n}"
	if string(modifiedContent) != expected {
		t.Errorf("Expected file content:\n%q\nGot:\n%q", expected, string(modifiedContent))
	}
}

func TestRunWithFlags_NormalizeWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "whitespace_test.txt")
	err := os.WriteFile(
		testFilePath,
		[]byte("func  main()   {\n\tfmt.Println(\"Hello, World!\")\n}"),
		0o644,
	) // Varied whitespace
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test with a pattern that has different whitespace than the content
	cfg := model.ModificationConfig{
		RuleID:              "normalize-test",
		Pattern:             "func main() { fmt.Println(\"Hello, World!\") }", // Normalized pattern
		Replacement:         "func main() { fmt.Println(\"Goodbye, World!\") }",
		Operation:           model.OpReplace,
		Occurrences:         "all",
		NormalizeWhitespace: true, // This is the key
	}

	r := &cli.Runner{Verbose: true}
	output := captureOutput(func() {
		exitCode := r.RunWithFlags([]string{testFilePath}, cfg, false)
		if exitCode != 0 {
			t.Errorf(
				"Expected exit code 0 for successful whitespace normalization match, got %d",
				exitCode,
			)
		}
	})

	if !strings.Contains(output, "✓") || !strings.Contains(output, "1 changes") {
		t.Errorf("Expected success message for whitespace normalization, got %q", output)
	}

	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	// Note: The replacement string itself is not normalized by this flag, only the pattern and content for matching.
	// The original content's whitespace will be preserved around the replacement.
	// This test will verify if the match happened despite whitespace differences.
	expected := "func  main()   {\n\tfmt.Println(\"Goodbye, World!\")\n}"
	if string(modifiedContent) != expected {
		t.Errorf("Expected file content:\n%q\nGot:\n%q", expected, string(modifiedContent))
	}
}

func TestRunWithFlags_LiteralPatternAndNormalizeWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "literal_norm_test.txt")
	err := os.WriteFile(
		testFilePath,
		[]byte("var myMap = map[string]int{\n\t\"key1\": 1,\n\t\"key2\": 2,\n}"),
		0o644,
	) // Varied whitespace, regex metacharacters
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Pattern with regex metacharacters and varied whitespace
	cfg := model.ModificationConfig{
		RuleID:              "literal-norm-test",
		Pattern:             "var myMap = map[string]int{ \"key1\": 1, \"key2\": 2, }",
		Replacement:         "var myMap = map[string]int{ \"newKey\": 100, }",
		Operation:           model.OpReplace,
		Occurrences:         "all",
		LiteralPattern:      true,
		NormalizeWhitespace: true,
	}

	r := &cli.Runner{Verbose: true}
	output := captureOutput(func() {
		exitCode := r.RunWithFlags([]string{testFilePath}, cfg, false)
		if exitCode != 0 {
			t.Errorf(
				"Expected exit code 0 for successful literal and normalized match, got %d",
				exitCode,
			)
		}
	})

	if !strings.Contains(output, "✓") || !strings.Contains(output, "1 changes") {
		t.Errorf("Expected success message for literal and normalized match, got %q", output)
	}

	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	expected := "var myMap = map[string]int{ \"newKey\": 100, }" // The original whitespace of the matched block is replaced by the replacement's whitespace.
	if string(modifiedContent) != expected {
		t.Errorf("Expected file content:\n%q\nGot:\n%q", expected, string(modifiedContent))
	}
}

func TestRunWithConfig_Success(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	configPath := filepath.Join(tempDir, "config.json")

	// Create a dummy file
	err := os.WriteFile(testFilePath, []byte("hello world"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a config file that replaces "world" with "Go"
	configContent := fmt.Sprintf(`{
		"schema_version": 1,
		"files": ["%s"],
		"rules": [
			{
				"rule_id": "test-replace",
				"pattern": "world",
				"replacement": "Go",
				"operation": "replace",
				"occurrences": "all"
			}
		]
	}`, strings.ReplaceAll(testFilePath, "\\", "\\")) // Escape backslashes for JSON

	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	r := &cli.Runner{Verbose: true}
	output := captureOutput(func() {
		exitCode := r.RunWithConfig(configPath)
		if exitCode != 0 {
			t.Errorf("Expected exit code 0 for successful run, got %d", exitCode)
		}
	})

	if !strings.Contains(output, "✓") || !strings.Contains(output, "1 changes") {
		t.Errorf("Expected success message in output, got %q", output)
	}

	// Verify file content
	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	if string(modifiedContent) != "hello Go" {
		t.Errorf("Expected file content 'hello Go', got '%s'", string(modifiedContent))
	}
}

func TestRunWithFlags_Success(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_file_flags.txt")

	// Create a dummy file
	err := os.WriteFile(testFilePath, []byte("foo bar"), 0o644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	cfg := model.ModificationConfig{
		RuleID:      "flag-test",
		Pattern:     "bar",
		Replacement: "baz",
		Operation:   model.OpReplace,
		Occurrences: "all",
	}

	r := &cli.Runner{Verbose: true}
	output := captureOutput(func() {
		exitCode := r.RunWithFlags([]string{testFilePath}, cfg, false)
		if exitCode != 0 {
			t.Errorf("Expected exit code 0 for successful run with flags, got %d", exitCode)
		}
	})

	if !strings.Contains(output, "✓") || !strings.Contains(output, "1 changes") {
		t.Errorf("Expected success message in output, got %q", output)
	}

	// Verify file content
	modifiedContent, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("Failed to read modified file: %v", err)
	}
	if string(modifiedContent) != "foo baz" {
		t.Errorf("Expected file content 'foo baz', got '%s'", string(modifiedContent))
	}
}
