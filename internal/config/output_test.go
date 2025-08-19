package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/termfx/morfx/internal/model"
)

func captureOutput(f func()) (string, string) {
	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	// Run the function
	f()

	// Close writers and restore
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Read captured output
	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	return bufOut.String(), bufErr.String()
}

func TestPrintResultCLI_GetOperation(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpGet,
		JSONOutput: false,
		ShowDiff:   false,
		StdoutMode: false,
		Verbose:    true, // Need verbose mode to show matches
	}

	res := &model.Result{
		File:          "test.go",
		Success:       true,
		ModifiedCount: 2,
		Changes: []model.Change{
			{
				New:       "func test()",
				LineStart: 10,
				LineEnd:   12,
			},
			{
				New:       "func main()",
				LineStart: 20,
				LineEnd:   22,
			},
		},
	}

	stdout, _ := captureOutput(func() {
		PrintResultCLI(res, cfg)
	})

	if !strings.Contains(stdout, "test.go") {
		t.Error("Expected output to contain filename")
	}
	if !strings.Contains(stdout, "func test()") {
		t.Error("Expected output to contain first match")
	}
	if !strings.Contains(stdout, "func main()") {
		t.Error("Expected output to contain second match")
	}
	if !strings.Contains(stdout, "lines 10-12") {
		t.Error("Expected output to contain line numbers for first match")
	}
	if !strings.Contains(stdout, "lines 20-22") {
		t.Error("Expected output to contain line numbers for second match")
	}
}

func TestPrintResultCLI_NoChanges(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpGet,
		JSONOutput: false,
		ShowDiff:   false,
		StdoutMode: false,
		Verbose:    true, // Need verbose mode for consistent behavior
	}

	res := &model.Result{
		File:          "test.go",
		Success:       true,
		ModifiedCount: 0,
		Changes:       []model.Change{},
	}

	stdout, _ := captureOutput(func() {
		PrintResultCLI(res, cfg)
	})

	if !strings.Contains(stdout, "✓ test.go — No changes") {
		t.Errorf("Expected output to show no changes message, got: %s", stdout)
	}
}

func TestPrintResultCLI_JSONOutput(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpReplace,
		JSONOutput: true,
		ShowDiff:   false,
		StdoutMode: false,
	}

	res := &model.Result{
		File:            "test.go",
		Success:         true,
		ModifiedCount:   1,
		OriginalContent: "original",
		ModifiedContent: "modified",
	}

	stdout, _ := captureOutput(func() {
		PrintResultCLI(res, cfg)
	})

	// Verify it's valid JSON
	var jsonRes model.Result
	err := json.Unmarshal([]byte(stdout), &jsonRes)
	if err != nil {
		t.Errorf("Expected valid JSON output, got error: %v", err)
	}

	if jsonRes.File != "test.go" {
		t.Errorf("Expected file to be 'test.go', got %s", jsonRes.File)
	}
}

func TestPrintResultCLI_StdoutMode(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpReplace,
		JSONOutput: false,
		ShowDiff:   false,
		StdoutMode: true,
	}

	res := &model.Result{
		File:            "test.go",
		Success:         true,
		ModifiedCount:   1,
		ModifiedContent: "modified content",
	}

	stdout, _ := captureOutput(func() {
		PrintResultCLI(res, cfg)
	})

	if stdout != "modified content" {
		t.Errorf("Expected stdout to contain modified content, got: %s", stdout)
	}
}

func TestPrintResultCLI_ShowDiff(t *testing.T) {
	cfg := &model.Config{
		Operation:   model.OpReplace,
		JSONOutput:  false,
		ShowDiff:    true,
		StdoutMode:  false,
		DiffContext: 3,
	}

	res := &model.Result{
		File:            "test.go",
		Success:         true,
		ModifiedCount:   1,
		OriginalContent: "line1\nline2\nline3",
		ModifiedContent: "line1\nmodified\nline3",
	}

	stdout, _ := captureOutput(func() {
		PrintResultCLI(res, cfg)
	})

	// The output should contain diff information
	// Note: The actual diff format depends on the util.UnifiedDiff implementation
	if len(stdout) == 0 {
		t.Error("Expected diff output, got empty string")
	}
}

func TestPrintFatal_JSONMode(t *testing.T) {
	err := errors.New("test error")

	stdout, _ := captureOutput(func() {
		PrintFatal(err, true)
	})

	// Should output JSON
	var cliErr model.CLIError
	jsonErr := json.Unmarshal([]byte(stdout), &cliErr)
	if jsonErr != nil {
		t.Errorf("Expected valid JSON output, got error: %v", jsonErr)
	}

	if cliErr.Message != "test error" {
		t.Errorf("Expected error message 'test error', got %s", cliErr.Message)
	}
}

func TestPrintFatal_CLIError(t *testing.T) {
	cliErr := model.CLIError{
		Code:    model.ErrInvalidQuery,
		Message: "invalid query",
	}

	stdout, _ := captureOutput(func() {
		PrintFatal(cliErr, true)
	})

	var outputErr model.CLIError
	jsonErr := json.Unmarshal([]byte(stdout), &outputErr)
	if jsonErr != nil {
		t.Errorf("Expected valid JSON output, got error: %v", jsonErr)
	}

	if outputErr.Code != model.ErrInvalidQuery {
		t.Errorf("Expected error code %v, got %v", model.ErrInvalidQuery, outputErr.Code)
	}
	if outputErr.Message != "invalid query" {
		t.Errorf("Expected error message 'invalid query', got %s", outputErr.Message)
	}
}

func TestPrintFatal_NonJSONMode(t *testing.T) {
	err := errors.New("test error")

	_, stderr := captureOutput(func() {
		PrintFatal(err, false)
	})

	if !strings.Contains(stderr, "Error: test error") {
		t.Errorf("Expected stderr to contain 'Error: test error', got: %s", stderr)
	}
}

func TestPrintSummary_CommitOperation(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpCommit,
		JSONOutput: false,
		StdoutMode: false,
	}

	results := []model.Result{}

	_, stderr := captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// Commit operations should not print summary
	if stderr != "" {
		t.Errorf("Expected no output for commit operation, got: %s", stderr)
	}
}

func TestPrintSummary_JSONOutput(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpReplace,
		JSONOutput: true,
		StdoutMode: false,
	}

	results := []model.Result{}

	_, stderr := captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// JSON output mode should not print summary
	if stderr != "" {
		t.Errorf("Expected no output for JSON mode, got: %s", stderr)
	}
}

func TestPrintSummary_StdoutMode(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpReplace,
		JSONOutput: false,
		StdoutMode: true,
	}

	results := []model.Result{}

	_, stderr := captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// Stdout mode should not print summary
	if stderr != "" {
		t.Errorf("Expected no output for stdout mode, got: %s", stderr)
	}
}

func TestPrintSummary_GetOperation(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpGet,
		JSONOutput: false,
		StdoutMode: false,
	}

	results := []model.Result{}

	_, _ = captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// Get operations use read-only writer which may or may not have summary
	// We just verify it doesn't crash
}

func TestPrintSummary_DryRunMode(t *testing.T) {
	cfg := &model.Config{
		Operation:  model.OpReplace,
		JSONOutput: false,
		StdoutMode: false,
		DryRun:     true,
	}

	results := []model.Result{}

	_, _ = captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// Dry run mode should use dry run writer
	// We just verify it doesn't crash
}

func TestPrintSummary_InteractiveMode(t *testing.T) {
	cfg := &model.Config{
		Operation:   model.OpReplace,
		JSONOutput:  false,
		StdoutMode:  false,
		Interactive: true,
	}

	results := []model.Result{}

	_, _ = captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// Interactive mode should use interactive writer
	// We just verify it doesn't crash
}

func TestPrintSummary_StagingMode(t *testing.T) {
	cfg := &model.Config{
		Operation:   model.OpReplace,
		JSONOutput:  false,
		StdoutMode:  false,
		DryRun:      false,
		Interactive: false,
	}

	results := []model.Result{}

	_, _ = captureOutput(func() {
		PrintSummary(results, cfg)
	})

	// Default staging mode
	// We just verify it doesn't crash
}

func TestPrintUsage(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("test-flag", "", "Test flag description")

	_, stderr := captureOutput(func() {
		PrintUsage(fs)
	})

	if !strings.Contains(stderr, "Usage: morfx [flags]") {
		t.Error("Expected usage message to contain 'Usage: morfx [flags]'")
	}
	if !strings.Contains(stderr, "Quick read usage:") {
		t.Error("Expected usage message to contain quick read usage")
	}
	if !strings.Contains(stderr, "Flags:") {
		t.Error("Expected usage message to contain 'Flags:'")
	}
	if !strings.Contains(stderr, "test-flag") {
		t.Error("Expected usage message to contain flag definitions")
	}
}
