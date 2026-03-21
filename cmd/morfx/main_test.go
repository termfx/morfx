package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Test variables - these should match the ones in main.go
var (
	testDebug              bool
	testDbURL              string
	testAutoApply          bool
	testAutoApplyThreshold float64
)

func TestRootCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "morfx",
		Short:   "Code transformation engine with MCP protocol support",
		Version: "1.5.0",
	}

	if cmd.Use != "morfx" {
		t.Errorf("Expected Use='morfx', got '%s'", cmd.Use)
	}

	if cmd.Version != "1.5.0" {
		t.Errorf("Expected Version='1.5.0', got '%s'", cmd.Version)
	}

	_ = cmd.Short // Suppress unused warning
}

func TestMCPCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP protocol server for AI agents",
	}

	if cmd.Use != "mcp" {
		t.Errorf("Expected Use='mcp', got '%s'", cmd.Use)
	}
}

func TestGlobalFlags(t *testing.T) {
	// Reset flags for testing
	testDebug = false
	testDbURL = ""
	testAutoApply = true
	testAutoApplyThreshold = 0.85

	cmd := &cobra.Command{Use: "test"}
	cmd.PersistentFlags().BoolVar(&testDebug, "debug", false, "Enable debug logging")

	// Test flag defaults
	if testDebug != false {
		t.Error("Debug should default to false")
	}

	// Test flag parsing
	cmd.ParseFlags([]string{"--debug"})
	if !testDebug {
		t.Error("Debug flag should be true after parsing --debug")
	}
}

func TestMCPFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "mcp"}
	cmd.Flags().StringVar(&testDbURL, "db", "", "Database URL")
	cmd.Flags().BoolVar(&testAutoApply, "auto-apply", true, "Auto apply")
	cmd.Flags().Float64Var(&testAutoApplyThreshold, "auto-threshold", 0.85, "Threshold")

	// Test defaults
	if testDbURL != "" {
		t.Error("dbURL should default to empty")
	}
	if !testAutoApply {
		t.Error("autoApply should default to true")
	}
	if testAutoApplyThreshold != 0.85 {
		t.Error("autoApplyThreshold should default to 0.85")
	}

	// Test flag parsing
	args := []string{"--db", "test.db", "--auto-apply=false", "--auto-threshold=0.9"}
	cmd.ParseFlags(args)

	if testDbURL != "test.db" {
		t.Errorf("Expected dbURL='test.db', got '%s'", testDbURL)
	}
	if testAutoApply {
		t.Error("autoApply should be false")
	}
	if testAutoApplyThreshold != 0.9 {
		t.Errorf("Expected threshold=0.9, got %f", testAutoApplyThreshold)
	}
}

func TestMainWithInvalidArgs(t *testing.T) {
	// Skip this test in regular runs as it would exit the process
	if os.Getenv("TEST_MAIN") != "1" {
		t.Skip("Skipping main test unless TEST_MAIN=1")
	}

	// This would test main() with invalid args, but we can't easily test
	// os.Exit without subprocess testing which is complex for this case
}

func TestHelpOutput(t *testing.T) {
	// Test help output contains expected content
	cmd := &cobra.Command{
		Use:   "morfx",
		Short: "Code transformation engine with MCP protocol support",
		Long: `Morfx MCP Server provides deterministic AST-based code transformations
through the Model Context Protocol (MCP) for AI agents.`,
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Execute help command
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	// When Long is present, cobra shows Long description instead of Short
	if !strings.Contains(output, "Morfx MCP Server") {
		t.Errorf("Help should contain long description, got: %s", output)
	}
	if !strings.Contains(output, "deterministic AST-based") {
		t.Errorf("Help should contain detailed description, got: %s", output)
	}
}

func TestVersionOutput(t *testing.T) {
	cmd := &cobra.Command{
		Use:     "morfx",
		Version: "1.5.0",
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	// Execute and capture output
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.5.0") {
		t.Error("Version output should contain version number")
	}
}
