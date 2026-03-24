package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	"github.com/oxhq/morfx/internal/buildinfo"
	"github.com/oxhq/morfx/mcp"
)

var (
	// Root command
	rootCmd = &cobra.Command{
		Use:   "morfx",
		Short: "Deterministic AST transformations with MCP and standalone tools",
		Long: `Morfx provides deterministic AST-based code transformations
through the Model Context Protocol (MCP) and standalone JSON tools.

The server communicates via JSON-RPC 2.0 over stdio and also ships standalone
stdin/stdout binaries for direct local automation. It supports language-agnostic
code querying, replacement, deletion, and insertion operations with confidence scoring.`,
	}

	// MCP server command
	mcpCmd = &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP protocol server for AI agents",
		Long: `Start the Model Context Protocol (MCP) server that communicates via JSON-RPC 2.0 over stdio.
		
This server is designed to integrate with AI agents that support the MCP protocol,
providing code transformation capabilities with confidence scoring and staged operations.`,
		Run: runMCPServer,
	}

	// Configuration flags
	dbURL              string
	debug              bool
	autoApply          bool
	autoApplyThreshold float64
)

func init() {
	refreshRootVersion()

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging to stderr")

	// MCP server flags
	mcpCmd.Flags().StringVar(&dbURL, "db", "", "SQLite/Turso database path or URL (default: ./.morfx/db/morfx.db)")
	mcpCmd.Flags().BoolVar(&autoApply, "auto-apply", true, "Enable auto-apply for high confidence operations")
	mcpCmd.Flags().
		Float64Var(&autoApplyThreshold, "auto-threshold", 0.85, "Confidence threshold for auto-apply (0.0-1.0)")

	// Add commands to root
	rootCmd.AddCommand(mcpCmd)
}

func refreshRootVersion() {
	rootCmd.Version = buildinfo.FormattedVersion()
}

func main() {
	// Load .env file if it exists (fail silently if not found)
	_ = godotenv.Load()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runMCPServer(cmd *cobra.Command, args []string) {
	// Build configuration
	config := mcp.DefaultConfig()

	// Override with command line flags
	if dbURL != "" {
		config.DatabaseURL = dbURL
	}
	config.Debug = debug
	config.AutoApplyEnabled = autoApply
	config.AutoApplyThreshold = autoApplyThreshold

	// Log startup info if debug enabled
	if debug {
		fmt.Fprintf(os.Stderr, "[INFO] Starting Morfx MCP Server\n")
		fmt.Fprintf(os.Stderr, "[INFO] Database: %s\n", config.DatabaseURL)
		fmt.Fprintf(os.Stderr, "[INFO] Auto-apply: %v (threshold: %.2f)\n",
			config.AutoApplyEnabled, config.AutoApplyThreshold)
	}

	// Create server
	server, err := mcp.NewStdioServer(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create server: %v\n", err)
		os.Exit(1)
	}

	// Ensure cleanup
	defer func() {
		if err := server.Close(); err != nil && debug {
			fmt.Fprintf(os.Stderr, "[WARN] Error during shutdown: %v\n", err)
		}
	}()

	// Start server (blocks until shutdown)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "[INFO] Server shutdown complete\n")
	}
}

// resetFlags resets all command-line flags to their default values
// Used primarily in tests to ensure clean state between test runs
func resetFlags() {
	debug = false
	dbURL = ""
	autoApply = true
	autoApplyThreshold = 0.85
}

// setupTestEnvironment sets up a test environment for integration tests
func setupTestEnvironment(t *testing.T) string {
	// Create temporary directory for test database
	tmpDir := "/tmp/morfx-test-" + fmt.Sprintf("%d", os.Getpid())
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	return tmpDir
}
