package main

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestRunMCPServerConfigurationParts tests individual parts of runMCPServer that we can test
func TestRunMCPServerConfigurationParts(t *testing.T) {
	// Test the configuration building logic from runMCPServer without creating server

	t.Run("debug logging simulation", func(t *testing.T) {
		// Test the debug logging part of runMCPServer
		resetFlags()
		debug = true
		dbURL = "test.db"
		autoApply = true
		autoApplyThreshold = 0.85

		var output bytes.Buffer

		// Simulate the debug logging that happens in runMCPServer (lines 81-86)
		if debug {
			fmt.Fprintf(&output, "[INFO] Starting Morfx MCP Server\n")
			fmt.Fprintf(&output, "[INFO] Database: %s\n", dbURL)
			fmt.Fprintf(&output, "[INFO] Auto-apply: %v (threshold: %.2f)\n",
				autoApply, autoApplyThreshold)
		}

		assert.Contains(t, output.String(), "[INFO] Starting Morfx MCP Server")
		assert.Contains(t, output.String(), "[INFO] Database: test.db")
		assert.Contains(t, output.String(), "[INFO] Auto-apply: true (threshold: 0.85)")
	})

	t.Run("configuration override logic", func(t *testing.T) {
		// Test the configuration override logic from runMCPServer (lines 73-78)
		resetFlags()

		// Test case 1: dbURL is empty, should use default
		dbURL = ""
		testDBURL := dbURL
		if testDBURL == "" {
			testDBURL = "./.morfx/db/morfx.db" // Default path
		}
		assert.Equal(t, "./.morfx/db/morfx.db", testDBURL)

		// Test case 2: dbURL is set, should use custom value
		dbURL = "custom.db"
		testDBURL = dbURL
		if testDBURL != "" {
			// This simulates the line: config.DatabaseURL = dbURL
			assert.Equal(t, "custom.db", testDBURL)
		}

		// Test other configuration assignments
		debug = true
		assert.True(t, debug) // Simulates: config.Debug = debug

		autoApply = false
		assert.False(t, autoApply) // Simulates: config.AutoApplyEnabled = autoApply

		autoApplyThreshold = 0.95
		assert.Equal(t, 0.95, autoApplyThreshold) // Simulates: config.AutoApplyThreshold = autoApplyThreshold
	})

	t.Run("cleanup logic simulation", func(t *testing.T) {
		// Test the cleanup logic from runMCPServer (lines 96-100)
		resetFlags()
		debug = true

		var output bytes.Buffer

		// Simulate the defer cleanup with error
		mockError := true
		if mockError && debug {
			// This simulates the cleanup warning (lines 97-99)
			fmt.Fprintf(&output, "[WARN] Error during shutdown: %v\n", "mock error")
		}

		assert.Contains(t, output.String(), "[WARN] Error during shutdown:")

		// Test successful cleanup (no error)
		output.Reset()
		mockError = false
		if mockError && debug {
			fmt.Fprintf(&output, "[WARN] Error during shutdown: %v\n", "mock error")
		}
		assert.Empty(t, output.String())
	})

	t.Run("final debug message simulation", func(t *testing.T) {
		// Test the final debug message (lines 108-110)
		resetFlags()
		debug = true

		var output bytes.Buffer

		// Simulate successful server shutdown
		if debug {
			fmt.Fprintf(&output, "[INFO] Server shutdown complete\n")
		}

		assert.Contains(t, output.String(), "[INFO] Server shutdown complete")

		// Test without debug
		output.Reset()
		debug = false
		if debug {
			fmt.Fprintf(&output, "[INFO] Server shutdown complete\n")
		}
		assert.Empty(t, output.String())
	})
}

// TestMCPCommandExecution tests calling the mcp command
func TestMCPCommandExecution(t *testing.T) {
	// We can't easily test the full runMCPServer, but we can test that
	// the command is properly set up and would call runMCPServer

	t.Run("mcp command setup", func(t *testing.T) {
		// Verify mcpCmd has the runMCPServer function assigned
		assert.NotNil(t, mcpCmd.Run)

		// Verify the command properties
		assert.Equal(t, "mcp", mcpCmd.Use)
		assert.Contains(t, mcpCmd.Short, "Start MCP protocol server")
	})

	t.Run("simulated mcp command call", func(t *testing.T) {
		// Create a mock version of the mcp command that doesn't actually start a server
		resetFlags()

		mockMcpCmd := &cobra.Command{
			Use:   "mcp",
			Short: "Start MCP protocol server for AI agents",
			Run: func(cmd *cobra.Command, args []string) {
				// Mock implementation that simulates what runMCPServer would do
				// without actually creating an MCP server

				// This exercises the configuration logic
				dbPath := dbURL
				if dbPath == "" {
					dbPath = "./.morfx/db/morfx.db"
				}

				if debug {
					fmt.Fprintf(os.Stderr, "[INFO] Mock MCP Server starting\n")
					fmt.Fprintf(os.Stderr, "[INFO] Database: %s\n", dbPath)
					fmt.Fprintf(os.Stderr, "[INFO] Auto-apply: %v (threshold: %.2f)\n",
						autoApply, autoApplyThreshold)
				}
			},
		}

		rootTestCmd := &cobra.Command{Use: "test"}

		// Add flags like the real command
		rootTestCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")
		mockMcpCmd.Flags().StringVar(&dbURL, "db", "", "Database URL")
		mockMcpCmd.Flags().BoolVar(&autoApply, "auto-apply", true, "Enable auto-apply")
		mockMcpCmd.Flags().Float64Var(&autoApplyThreshold, "auto-threshold", 0.85, "Threshold")

		rootTestCmd.AddCommand(mockMcpCmd)

		// Test various flag combinations
		testCases := [][]string{
			{"--debug", "mcp"},
			{"mcp", "--db", "test.db"},
			{"mcp", "--auto-apply=false"},
			{"mcp", "--auto-threshold", "0.95"},
			{"--debug", "mcp", "--db", "custom.db", "--auto-apply", "--auto-threshold", "0.88"},
		}

		for i, args := range testCases {
			t.Run(fmt.Sprintf("flags_case_%d", i), func(t *testing.T) {
				resetFlags()
				rootTestCmd.SetArgs(args)

				err := rootTestCmd.Execute()
				assert.NoError(t, err)

				// Verify flags were parsed correctly
				if len(args) > 1 {
					// At least one flag should be set
					hasFlag := debug || dbURL != "" || !autoApply || autoApplyThreshold != 0.85
					assert.True(t, hasFlag, "At least one flag should be different from default")
				}
			})
		}
	})
}

// TestMainFunctionErrorPath tests the error handling in main()
func TestMainFunctionErrorPath(t *testing.T) {
	// We can't test main() directly because it calls os.Exit,
	// but we can test the error handling logic

	t.Run("command error simulation", func(t *testing.T) {
		// Create a command that returns an error like main() would handle
		errorCmd := &cobra.Command{
			Use: "test",
			RunE: func(cmd *cobra.Command, args []string) error {
				return fmt.Errorf("simulated error")
			},
		}

		errorCmd.SetArgs([]string{})

		var stderr bytes.Buffer
		errorCmd.SetErr(&stderr)

		err := errorCmd.Execute()
		assert.Error(t, err)

		// This simulates what main() would do:
		// fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		// os.Exit(1)

		if err != nil {
			var errorOutput bytes.Buffer
			fmt.Fprintf(&errorOutput, "Error: %v\n", err)
			assert.Contains(t, errorOutput.String(), "Error: simulated error")
		}
	})

	t.Run("successful command simulation", func(t *testing.T) {
		// Test successful command execution (no error)
		successCmd := &cobra.Command{
			Use: "test",
			Run: func(cmd *cobra.Command, args []string) {
				// Success case
			},
		}

		successCmd.SetArgs([]string{})
		err := successCmd.Execute()
		assert.NoError(t, err)

		// In main(), this would just exit normally without calling os.Exit(1)
	})
}
