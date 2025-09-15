# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.3.0] - 2024-09-14

### Added
- **Multi-Language Provider Architecture** - Complete AST transformation system
  - Go provider with full `go/ast` integration and confidence scoring
  - PHP provider supporting modern syntax (7.4+) with Eloquent models
  - JavaScript provider with ES6+ and JSX support
  - TypeScript provider with full type system support
  - Unified provider interface with standardized operations
- **Enhanced Query Operations** - Advanced code pattern matching
  - Support for wildcard patterns in name matching (`*`, `?`)
  - Multi-file batch operations across entire codebases
  - Confidence scoring for transformation safety (0.0-1.0)
  - Context-aware AST node targeting and manipulation
- **File-Based Operations** - Comprehensive file system integration
  - `file_query` and `file_replace` for multi-file transformations
  - Backup creation and dry-run mode for safe operations
  - Directory scanning with include/exclude pattern support
  - Language detection and provider auto-selection
- **Advanced Insert Operations** - Precise code injection
  - `insert_before` and `insert_after` with target-based positioning
  - Smart code placement with proper indentation handling
  - Context-aware insertion for functions, structs, and classes

### Changed
- **Provider System** - Modular, extensible architecture
  - Standardized confidence scoring across all language providers
  - Unified error handling with provider-specific error types
  - Improved AST parsing performance with optimized tree-sitter integration
  - Enhanced memory management for large file operations
- **MCP Tool Definitions** - More intuitive and powerful API
  - Simplified parameter validation with clear error messages
  - Consistent response format across all operations
  - Better documentation and examples in tool descriptions
  - Enhanced debugging capabilities with detailed operation logs

### Performance
- **Batch Processing** - Optimized for large-scale operations
  - Concurrent file processing with worker pool architecture
  - Memory-efficient streaming for large codebases
  - Intelligent caching of parsed AST trees
  - Performance metrics and operation timing

### Developer Experience
- **Enhanced Error Messages** - Clear, actionable feedback
  - Detailed context for transformation failures
  - Suggestions for fixing common syntax issues
  - Provider-specific error codes and resolution hints
- **Testing Infrastructure** - Comprehensive validation
  - Provider-specific test suites with real-world examples
  - Performance benchmarks and regression testing
  - Multi-language sample projects for integration testing

## [v1.2.0] - 2024-09-14

### Removed
- **HTTP Server Support** - Simplified for MCP-only usage
  - Removed OAuth authentication (OpenAI, GitHub, Auth0, Google providers)
  - Removed HTTP API endpoints and CORS support
  - Removed API key authentication for HTTP
  - Removed `.env` OAuth configuration options
- **Database Changes**
  - Migrated from PostgreSQL to SQLite for simplicity
  - Removed OAuth session models and client registration
  - Simplified configuration for MCP-only usage

### Changed
- **MCP Protocol Only** - Focus on local AI agent integration
  - Simplified to stdio communication only
  - Streamlined configuration with fewer options
  - SQLite database auto-creation and management
- **Documentation Updates**
  - Removed HTTP server usage examples
  - Updated installation and configuration guides
  - Simplified deployment documentation

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

[v1.3.0]: https://github.com/termfx/morfx/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/termfx/morfx/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/termfx/morfx/compare/v1.0.0...v1.1.0
[v1.0.0]: https://github.com/termfx/morfx/releases/tag/v1.0.0
