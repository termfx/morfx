package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/termfx/morfx/mcp"
)

var (
	// Root command
	rootCmd = &cobra.Command{
		Use:   "morfx",
		Short: "Code transformation engine with MCP protocol support",
		Long: `Morfx MCP Server provides deterministic AST-based code transformations
through the Model Context Protocol (MCP) for AI agents.

The server communicates via JSON-RPC 2.0 over stdio and supports language-agnostic
code querying, replacement, deletion, and insertion operations with confidence scoring.`,
		Version: "0.1.0",
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

	// HTTP server command
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start HTTP server for remote access",
		Long: `Start HTTP server that provides the same MCP functionality via REST API.
		
This server exposes morfx transformations over HTTP with JSON-RPC 2.0 payloads,
enabling remote access, web integrations, and CI/CD pipeline usage.`,
		Run: runHTTPServer,
	}

	// Configuration flags
	dbURL              string
	debug              bool
	autoApply          bool
	autoApplyThreshold float64
	
	// HTTP server flags
	httpPort   int
	httpHost   string
	apiKey     string
	corsOrigin string
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging to stderr")

	// MCP server flags
	mcpCmd.Flags().StringVar(&dbURL, "db", "", "PostgreSQL connection string (default: postgres://localhost/morfx_dev)")
	mcpCmd.Flags().BoolVar(&autoApply, "auto-apply", true, "Enable auto-apply for high confidence operations")
	mcpCmd.Flags().Float64Var(&autoApplyThreshold, "auto-threshold", 0.85, "Confidence threshold for auto-apply (0.0-1.0)")

	// HTTP server flags
	serveCmd.Flags().StringVar(&dbURL, "db", "", "PostgreSQL connection string (default: postgres://localhost/morfx_dev)")
	serveCmd.Flags().BoolVar(&autoApply, "auto-apply", true, "Enable auto-apply for high confidence operations")
	serveCmd.Flags().Float64Var(&autoApplyThreshold, "auto-threshold", 0.85, "Confidence threshold for auto-apply (0.0-1.0)")
	serveCmd.Flags().IntVar(&httpPort, "port", 8080, "HTTP server port")
	serveCmd.Flags().StringVar(&httpHost, "host", "localhost", "HTTP server host")
	serveCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (required)")
	serveCmd.Flags().StringVar(&corsOrigin, "cors", "*", "CORS allowed origins")

	// Add commands to root
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(serveCmd)
}

func main() {
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

func runHTTPServer(cmd *cobra.Command, args []string) {
	// Validate required flags
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: --api-key is required for HTTP server\n")
		os.Exit(1)
	}
	
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
		fmt.Fprintf(os.Stderr, "[INFO] Starting Morfx HTTP Server\n")
		fmt.Fprintf(os.Stderr, "[INFO] Address: %s:%d\n", httpHost, httpPort)
		fmt.Fprintf(os.Stderr, "[INFO] Database: %s\n", config.DatabaseURL)
		fmt.Fprintf(os.Stderr, "[INFO] Auto-apply: %v (threshold: %.2f)\n", 
			config.AutoApplyEnabled, config.AutoApplyThreshold)
	}
	
	// Create HTTP server
	server, err := mcp.NewHTTPServer(config, httpHost, httpPort, apiKey, corsOrigin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create HTTP server: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		os.Exit(1)
	}
	
	if debug {
		fmt.Fprintf(os.Stderr, "[INFO] HTTP server shutdown complete\n")
	}
}
