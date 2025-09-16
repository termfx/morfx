# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.4.0] - 2024-09-16

### Added
- **Interactive Demo System** - Complete demonstration framework
  - Live AST transformations with real-time visual feedback
  - Automatic backup/restore for safe experimentation
  - Multi-language scenario demonstrations (Go, PHP, JavaScript, TypeScript)
  - Before/after diff visualization with syntax highlighting
  - Direct provider integration for authentic transformations
- **MCP Prompts System** - AI-guided code analysis prompts
  - Code analysis prompt for understanding code structure
  - Transformation guide for operation planning
  - Confidence explanation for understanding scoring
  - Query builder for creating complex patterns
  - Best practices guide for each language
- **MCP Resources System** - Server capability exposure
  - Server information and version details
  - Supported languages and provider capabilities
  - Current session state and statistics
  - Configuration settings exposure
  - Dynamic resource discovery
- **Modular Tool Architecture** - Refactored tool system
  - Separate tool modules for better organization
  - Standardized tool interface and registry
  - Enhanced parameter validation
  - Improved error handling and reporting
- **Cross-Platform Process Detection** - Platform-specific optimizations
  - Unix process detection with signal verification
  - Windows process handle validation
  - Stale lock detection and cleanup
  - Process ownership verification

### Fixed
- **Wildcard Pattern Matching** - Critical bug fix
  - Fixed `*middle*` patterns not matching correctly
  - Proper handling of prefix/suffix wildcards
  - Support for complex patterns like `*User*` in queries
  - Edge case handling for empty patterns
- **Provider Method Mapping** - Language-specific fixes
  - PHP method detection now correctly maps to method_declaration
  - TypeScript interface detection improvements
  - JavaScript arrow function name extraction
  - Go struct vs interface differentiation
- **Async Staging Resilience** - Improved error handling
  - Graceful fallback to synchronous operations on channel closure
  - Context cancellation handling
  - Race condition prevention in worker pools
  - Proper cleanup on manager shutdown

### Changed
- **Provider Match Expansion** - More accurate transformations
  - Multi-variable declarations properly expanded (Go)
  - Destructuring assignments handled correctly (JS/TS)
  - Property declarations expanded in classes (PHP)
  - Tuple unpacking support (Python)
- **Error Propagation** - Better debugging experience
  - Detailed error messages with context
  - Syntax error detection before transformation
  - Line/column information in error reports
  - Stack traces for internal errors
- **Demo Command** - Production-ready demonstration
  - `make demo` now shows real transformations
  - Connected to actual provider implementations
  - Error handling for missing targets
  - Colorized output for better readability

### Performance
- **AST Caching** - Reduced parsing overhead
  - Global cache for parsed AST trees
  - Cache invalidation on file changes
  - Memory-efficient cache management
  - 3x speedup for repeated queries
- **Batch Operations** - Optimized parallel processing
  - Worker pool size auto-tuning
  - Intelligent work distribution
  - Memory usage optimization
  - Progress reporting for long operations

### Developer Experience
- **Testing Infrastructure** - Comprehensive test coverage
  - 80%+ coverage across all core modules
  - Integration tests for all providers
  - Cross-platform compatibility tests
  - Performance benchmarks included
- **Documentation Updates** - Better guidance
  - Updated README with correct tool usage
  - Provider-specific documentation
  - Demo usage instructions
  - Troubleshooting guide

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

## [v1.2.0] - 2024-09-13

### Added
- **Advanced Safety Mechanisms**
  - Transaction logging with automatic rollback capability
  - File integrity verification before modifications
  - Atomic file operations with proper locking
  - Configurable safety thresholds and validation rules
  - Comprehensive backup system for all modifications

### Changed
- **MCP-Only Architecture**: Complete removal of HTTP/OAuth, focused purely on MCP protocol
- **Enhanced Staging System**: Improved staging with TTL, automatic cleanup, and bulk operations
- **Simplified Configuration**: Streamlined environment variables and configuration options

### Fixed
- Staging system memory leaks during long-running sessions
- Race conditions in concurrent file operations
- Error propagation in nested transformations

## [v1.1.0] - 2024-09-12

### Added
- OAuth 2.0 authentication system with GitHub, Google, and GitLab support
- Enhanced HTTP server with authentication middleware
- Session management with secure token handling
- Team collaboration features with shared transformation sessions

### Changed
- Extended MCP protocol with authentication support
- Improved error handling and logging
- Enhanced API documentation

## [v1.0.0] - 2024-09-11

### Added
- Initial release with core MCP server functionality
- Basic AST transformation operations (query, replace, delete, insert)
- Go language provider using tree-sitter
- Confidence scoring system
- SQLite-based staging system
- Basic file operations and safety checks

### Core Features
- MCP 2024-11-05 protocol compliance
- JSON-RPC 2.0 communication over stdio
- Tool discovery and registration
- Resource and prompt management
- Real-time notifications and logging

