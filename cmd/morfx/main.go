package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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
		Version: "1.1.0",
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
	noAuth     bool

	// OAuth flags
	oauthProvider     string
	oauthClientID     string
	oauthClientSecret string
	oauthIssuerURL    string
	oauthDomain       string
	oauthAudience     string
)

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging to stderr")

	// MCP server flags
	mcpCmd.Flags().StringVar(&dbURL, "db", "", "PostgreSQL connection string (default: postgres://localhost/morfx_dev)")
	mcpCmd.Flags().BoolVar(&autoApply, "auto-apply", true, "Enable auto-apply for high confidence operations")
	mcpCmd.Flags().
		Float64Var(&autoApplyThreshold, "auto-threshold", 0.85, "Confidence threshold for auto-apply (0.0-1.0)")

	// HTTP server flags
	serveCmd.Flags().
		StringVar(&dbURL, "db", "", "PostgreSQL connection string (default: postgres://localhost/morfx_dev)")
	serveCmd.Flags().BoolVar(&autoApply, "auto-apply", true, "Enable auto-apply for high confidence operations")
	serveCmd.Flags().
		Float64Var(&autoApplyThreshold, "auto-threshold", 0.85, "Confidence threshold for auto-apply (0.0-1.0)")
	serveCmd.Flags().IntVar(&httpPort, "port", 8080, "HTTP server port")
	serveCmd.Flags().StringVar(&httpHost, "host", "localhost", "HTTP server host")
	serveCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication (required unless --no-auth)")
	serveCmd.Flags().StringVar(&corsOrigin, "cors", "*", "CORS allowed origins")
	serveCmd.Flags().BoolVar(&noAuth, "no-auth", false, "Disable authentication (USE ONLY in trusted environments)")

	// OAuth flags
	serveCmd.Flags().
		StringVar(&oauthProvider, "oauth-provider", "", "OAuth provider (openai, github, google, auth0, custom)")
	serveCmd.Flags().StringVar(&oauthClientID, "oauth-client-id", "", "OAuth client ID")
	serveCmd.Flags().StringVar(&oauthClientSecret, "oauth-client-secret", "", "OAuth client secret")
	serveCmd.Flags().
		StringVar(&oauthIssuerURL, "oauth-issuer", "", "OAuth issuer URL (auto-detected for known providers)")
	serveCmd.Flags().StringVar(&oauthDomain, "oauth-domain", "", "OAuth domain (for Auth0)")
	serveCmd.Flags().
		StringVar(&oauthAudience, "oauth-audience", "", "OAuth audience (auto-detected for known providers)")

	// Add commands to root
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(serveCmd)
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

func runHTTPServer(cmd *cobra.Command, args []string) {
	// Load configuration from env vars (flags override env vars)
	loadEnvConfig()

	// Determine auth mode
	authMode := determineAuthMode()

	// Validate auth configuration
	if err := validateAuthConfig(authMode); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "[INFO] Authentication: %s\n",
			map[bool]string{true: "DISABLED", false: "enabled"}[noAuth])
		fmt.Fprintf(os.Stderr, "[INFO] Auto-apply: %v (threshold: %.2f)\n",
			config.AutoApplyEnabled, config.AutoApplyThreshold)
	}

	// Create HTTP server with OAuth config
	oauthConfig := buildOAuthConfig(authMode)
	server, err := mcp.NewHTTPServer(config, httpHost, httpPort, apiKey, corsOrigin, noAuth, oauthConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create HTTP server: %v\n", err)
		os.Exit(1)
	}

	// Print MASSIVE warning if no-auth is enabled
	if noAuth {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨\n")
		fmt.Fprintf(os.Stderr, "🚨                                                          🚨\n")
		fmt.Fprintf(os.Stderr, "🚨  WARNING: AUTHENTICATION IS DISABLED (--no-auth)        🚨\n")
		fmt.Fprintf(os.Stderr, "🚨                                                          🚨\n")
		fmt.Fprintf(os.Stderr, "🚨  This server is now COMPLETELY UNPROTECTED!             🚨\n")
		fmt.Fprintf(os.Stderr, "🚨  Anyone can execute code transformations!               🚨\n")
		fmt.Fprintf(os.Stderr, "🚨                                                          🚨\n")
		fmt.Fprintf(os.Stderr, "🚨  ONLY use this for local development on localhost       🚨\n")
		fmt.Fprintf(os.Stderr, "🚨  NEVER expose this to the internet or public networks   🚨\n")
		fmt.Fprintf(os.Stderr, "🚨                                                          🚨\n")
		fmt.Fprintf(os.Stderr, "🚨  Server running on: %s:%d                        \n", httpHost, httpPort)
		if httpHost == "0.0.0.0" || httpHost == "" {
			fmt.Fprintf(os.Stderr, "🚨  ⚠️  DANGER: Listening on ALL interfaces!               🚨\n")
		}
		fmt.Fprintf(os.Stderr, "🚨                                                          🚨\n")
		fmt.Fprintf(os.Stderr, "🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨🚨\n")
		fmt.Fprintf(os.Stderr, "\n")

		// Extra warning if listening on all interfaces
		if httpHost == "0.0.0.0" || httpHost == "" {
			fmt.Fprintf(os.Stderr, "💀 YOU ARE EXPOSING AN UNAUTHENTICATED SERVER TO ALL NETWORK INTERFACES!\n")
			fmt.Fprintf(os.Stderr, "💀 This is EXTREMELY DANGEROUS unless you have proper firewall rules.\n")
			fmt.Fprintf(os.Stderr, "💀 Consider using --host 127.0.0.1 for local-only access.\n\n")
		}
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

// determineAuthMode determines authentication mode based on flags
func determineAuthMode() mcp.AuthMode {
	if noAuth {
		return mcp.AuthModeNone
	}
	if oauthProvider != "" {
		return mcp.AuthModeOAuth
	}
	return mcp.AuthModeAPIKey
}

// validateAuthConfig validates authentication configuration
func validateAuthConfig(mode mcp.AuthMode) error {
	switch mode {
	case mcp.AuthModeNone:
		return nil
	case mcp.AuthModeAPIKey:
		if apiKey == "" {
			return fmt.Errorf("--api-key is required for API key authentication")
		}
		return nil
	case mcp.AuthModeOAuth:
		if oauthProvider == "" {
			return fmt.Errorf("--oauth-provider is required for OAuth authentication")
		}

		// Validate provider-specific requirements
		switch mcp.OAuthProvider(oauthProvider) {
		case mcp.OAuthProviderAuth0:
			if oauthDomain == "" {
				return fmt.Errorf("--oauth-domain is required for Auth0 provider")
			}
		case mcp.OAuthProviderCustom:
			if oauthIssuerURL == "" {
				return fmt.Errorf("--oauth-issuer is required for custom provider")
			}
		}

		return nil
	default:
		return fmt.Errorf("unknown authentication mode")
	}
}

// buildOAuthConfig creates OAuth configuration from flags
func buildOAuthConfig(mode mcp.AuthMode) *mcp.OAuthConfig {
	if mode != mcp.AuthModeOAuth {
		return nil
	}

	return &mcp.OAuthConfig{
		Provider:     mcp.OAuthProvider(oauthProvider),
		ClientID:     oauthClientID,
		ClientSecret: oauthClientSecret,
		IssuerURL:    oauthIssuerURL,
		JWKSURL:      "", // Auto-detected by provider
		Audience:     oauthAudience,
		Domain:       oauthDomain,
		Scopes:       []string{}, // Can be extended later
	}
}

// loadEnvConfig loads configuration from environment variables
// Precedence: CLI flags > .env file > defaults
func loadEnvConfig() {
	// Database
	if dbURL == "" {
		dbURL = os.Getenv("MORFX_DATABASE_URL")
	}

	// HTTP Server
	if httpPort == 8080 { // Only override default
		if port := os.Getenv("MORFX_PORT"); port != "" {
			if p, err := strconv.Atoi(port); err == nil {
				httpPort = p
			}
		}
	}
	if httpHost == "localhost" { // Only override default
		if host := os.Getenv("MORFX_HOST"); host != "" {
			httpHost = host
		}
	}
	if apiKey == "" {
		apiKey = os.Getenv("MORFX_API_KEY")
	}
	if corsOrigin == "*" { // Only override default
		if cors := os.Getenv("MORFX_CORS_ORIGIN"); cors != "" {
			corsOrigin = cors
		}
	}

	// OAuth
	if oauthProvider == "" {
		oauthProvider = os.Getenv("MORFX_OAUTH_PROVIDER")
	}
	if oauthClientID == "" {
		oauthClientID = os.Getenv("MORFX_OAUTH_CLIENT_ID")
	}
	if oauthClientSecret == "" {
		oauthClientSecret = os.Getenv("MORFX_OAUTH_CLIENT_SECRET")
	}
	if oauthIssuerURL == "" {
		oauthIssuerURL = os.Getenv("MORFX_OAUTH_ISSUER")
	}
	if oauthDomain == "" {
		oauthDomain = os.Getenv("MORFX_OAUTH_DOMAIN")
	}
	if oauthAudience == "" {
		oauthAudience = os.Getenv("MORFX_OAUTH_AUDIENCE")
	}
}
