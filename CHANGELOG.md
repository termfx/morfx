# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.1.0] - 2024-09-11

### Added
- **OAuth Authentication Support** - Complete OAuth 2.0/OIDC integration
  - OpenAI (ChatGPT Enterprise) provider support
  - GitHub Actions OIDC provider
  - Auth0 provider with domain configuration
  - Google OAuth provider
  - Custom OAuth provider support with flexible issuer configuration
- **Enhanced HTTP Server** - Production-ready HTTP API server
  - Multiple authentication modes (OAuth, API Key, No Auth)
  - Environment variable configuration support
  - `.env` file support with precedence: CLI flags > .env > defaults
  - Comprehensive CORS configuration
  - Security warnings for dangerous configurations
- **New CLI Features**
  - `--oauth-provider` flag with multiple provider support
  - `--oauth-client-id`, `--oauth-client-secret` configuration
  - `--oauth-issuer`, `--oauth-domain`, `--oauth-audience` for custom setups
  - `--no-auth` flag with extensive security warnings
  - Environment variable integration for all configuration options
- **Security Enhancements**
  - JWT token validation and verification
  - OAuth session management with expiration
  - API key authentication improvements
  - Massive security warnings for dangerous configurations

### Changed
- **Configuration System** - Complete rewrite with hierarchical precedence
  - CLI flags override environment files
  - Environment files override defaults
  - Support for `MORFX_*` environment variables
  - Improved configuration validation and error messages
- **HTTP Server Architecture** - Enhanced for production use
  - Better error handling and response formatting
  - Improved logging and debugging capabilities
  - Enhanced CORS support with configurable origins
- **Database Models** - Extended for OAuth and session tracking
  - New `MCPSession` model for HTTP session management
  - `OAuthClient` model for RFC7591 client registration
  - Enhanced session tracking with OAuth integration

### Security
- **OAuth Token Validation** - Complete JWT verification pipeline
  - JWKS endpoint auto-discovery for known providers
  - Token signature verification and claims validation
  - Automatic token refresh and revocation support
- **Authentication Modes** - Three distinct security levels
  - OAuth mode for enterprise SSO integration
  - API Key mode for traditional bearer token auth
  - No Auth mode with comprehensive warnings for development only
- **Session Security** - Enhanced session management
  - Secure session tokens with SHA256 hashing
  - Automatic session expiration and cleanup
  - OAuth subject tracking for audit trails

### Dependencies
- Added `github.com/joho/godotenv` for .env file support
- Added JWT and OAuth 2.0 libraries for authentication
- Updated existing dependencies to latest stable versions

## [v1.0.0] - 2024-09-10

### Added
- Initial release of Morfx MCP Server
- AST-based code transformations using tree-sitter
- Go language provider with comprehensive support
- MCP protocol implementation over stdio
- Basic HTTP server support
- Confidence scoring system
- Staging and apply operations
- PostgreSQL integration for persistence
- Query, replace, delete, insert operations
- Natural language query support
- Performance optimizations with caching
- Parallel processing pipeline for batch operations

### Core Features
- Model Context Protocol (MCP) server implementation
- JSON-RPC 2.0 communication over stdio and HTTP
- Tree-sitter based AST parsing and manipulation
- Confidence-based transformation safety
- Two-phase commit with staging system
- Session management and audit trails
- Language-agnostic architecture with Go provider

[v1.1.0]: https://github.com/termfx/morfx/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/termfx/morfx/releases/tag/v1.0.0
