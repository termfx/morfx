package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var entryPath string

// TestMain locates the main entry point for the morfx CLI tool.
// It skips integration tests if not found.
func TestMain(m *testing.M) {
	path, err := filepath.Abs("cmd/morfx")
	if err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			entryPath = path
		}
	}
	os.Exit(m.Run())
}

// setupTestFile creates a temporary Go file with the given content.
// It returns the file path and a cleanup function to be deferred.
func setupTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp(t.TempDir(), "test_*.go")
	require.NoError(t, err, "Failed to create temp file")

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err, "Failed to write to temp file")

	err = tmpFile.Close()
	require.NoError(t, err, "Failed to close temp file")

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})
	return tmpFile.Name()
}

// assertFileContent asserts that the content of a file matches the expected string.
func assertFileContent(t *testing.T, filePath, expectedContent string) {
	t.Helper()
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err, "Failed to read file for assertion")
	assert.Equal(t, expectedContent, string(actualContent))
}

func TestASTCommand_GetOperations(t *testing.T) {
	if entryPath == "" {
		t.Skip("Skipping integration test: morfx binary not found")
	}
	content := `package main

import "fmt"

type User struct { ID int }

func GetUser() {
	// some logic
}
`
	testFile := setupTestFile(t, content)

	testCases := []struct {
		name           string
		target         string
		expectedOutput string
	}{
		{
			name:   "GetFunction",
			target: "function:GetUser",
			expectedOutput: `func GetUser() {
	// some logic
}`,
		},
		{
			name:           "GetStruct",
			target:         "struct:User",
			expectedOutput: `type User struct { ID int }`,
		},
		{
			name:           "GetImport",
			target:         "import:fmt",
			expectedOutput: `"fmt"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(entryPath, "ast", "--target", tc.target, "--file", testFile)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Command failed unexpectedly. Output: %s", string(output))
			assert.Equal(t, tc.expectedOutput, strings.TrimSpace(string(output)))
		})
	}
}

func TestASTCommand_ModificationOperations(t *testing.T) {
	if entryPath == "" {
		t.Skip("Skipping integration test: morfx binary not found")
	}
	baseContent := `package main

func GetUser() {
	// old logic
}

func KeepThisOne() {}
`
	t.Run("ReplaceFunction", func(t *testing.T) {
		testFile := setupTestFile(t, baseContent)
		newFunction := `func GetUser() { /* new logic */ }`
		cmd := exec.Command(
			entryPath,
			"ast",
			"--target",
			"function:GetUser",
			"--operation",
			"replace",
			"--file",
			testFile,
		)
		cmd.Stdin = strings.NewReader(newFunction)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Replace command failed. Output: %s", string(output))

		expectedContent := `package main

func GetUser() { /* new logic */ }

func KeepThisOne() {}
`
		assertFileContent(t, testFile, expectedContent)
	})

	t.Run("DeleteFunction", func(t *testing.T) {
		testFile := setupTestFile(t, baseContent)
		cmd := exec.Command(
			entryPath,
			"ast",
			"--target",
			"function:GetUser",
			"--operation",
			"delete",
			"--file",
			testFile,
		)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Delete command failed. Output: %s", string(output))

		expectedContent := `package main



func KeepThisOne() {}
`
		assertFileContent(t, testFile, expectedContent)
	})

	t.Run("InsertAfterFunction", func(t *testing.T) {
		testFile := setupTestFile(t, baseContent)
		insertedFunc := `
func InsertedFunc() {}`
		cmd := exec.Command(
			entryPath,
			"ast",
			"--target",
			"function:GetUser",
			"--operation",
			"insert-after",
			"--file",
			testFile,
		)
		cmd.Stdin = strings.NewReader(insertedFunc)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Insert-after command failed. Output: %s", string(output))

		expectedContent := `package main

func GetUser() {
	// old logic
}
func InsertedFunc() {}

func KeepThisOne() {}
`
		assertFileContent(t, testFile, expectedContent)
	})

	t.Run("InsertBeforeFunction", func(t *testing.T) {
		testFile := setupTestFile(t, baseContent)
		insertedFunc := `// A new function will be inserted after this comment.
func InsertedFunc() {}
`
		cmd := exec.Command(
			entryPath,
			"ast",
			"--target",
			"function:KeepThisOne",
			"--operation",
			"insert-before",
			"--file",
			testFile,
		)
		cmd.Stdin = strings.NewReader(insertedFunc)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Insert-before command failed. Output: %s", string(output))

		expectedContent := `package main

func GetUser() {
	// old logic
}

// A new function will be inserted after this comment.
func InsertedFunc() {}
func KeepThisOne() {}
`
		assertFileContent(t, testFile, expectedContent)
	})
}

func TestASTCommand_FailureScenarios(t *testing.T) {
	if entryPath == "" {
		t.Skip("Skipping integration test: morfx binary not found")
	}
	t.Run("InvalidTargetFormat", func(t *testing.T) {
		cmd := exec.Command(entryPath, "ast", "--target", "invalidformat", "--file", "dummy.go")
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "Command should have failed but did not")
		assert.Contains(t, string(output), "invalid --target format")
	})

	t.Run("NoFileSpecified", func(t *testing.T) {
		cmd := exec.Command(entryPath, "ast", "--target", "function:foo")
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "Command should have failed but did not")
		assert.Contains(t, string(output), "at least one -file is required")
	})

	t.Run("UnsupportedLanguage", func(t *testing.T) {
		testFile := setupTestFile(t, "content")
		cmd := exec.Command(
			entryPath,
			"ast",
			"--target",
			"function:foo",
			"--lang",
			"brainfuck",
			"--file",
			testFile,
		)
		output, err := cmd.CombinedOutput()
		require.Error(t, err, "Command should have failed but did not")
		assert.Contains(t, string(output), "unsupported language: brainfuck")
	})

	t.Run("TargetNotFound", func(t *testing.T) {
		testFile := setupTestFile(t, `package main`)
		cmd := exec.Command(
			entryPath,
			"ast",
			"--target",
			"function:nonexistent",
			"--file",
			testFile,
		)
		// This should succeed with no output and no changes
		output, err := cmd.CombinedOutput()
		require.NoError(
			t,
			err,
			"Command failed for a non-existent target. Output: %s",
			string(output),
		)
		assert.Empty(t, string(output), "Expected no output for a non-existent target")
	})
}
