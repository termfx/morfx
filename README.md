# Morfx MCP Server

> **Enterprise-grade code transformations for AI agents**

Morfx is a Model Context Protocol (MCP) server that provides deterministic AST-based code transformations for AI agents. Transform code with confidence using natural language queries, intelligent staging, and automatic safety verification.

## 🚀 Quick Start

### Prerequisites

- Go 1.24+ 
- PostgreSQL 14+
- Any AI agent with MCP protocol support

### Installation

```bash
# Clone and build
git clone https://github.com/termfx/morfx.git
cd morfx
go build -o bin/morfx cmd/morfx/main.go
```

### Configuration

Add morfx to your AI agent's MCP server configuration. Here's an example showing morfx integrated with other MCP tools:

```json
{
  "mcpServers": {
    "desktop-commander": {
      "command": "npx",
      "args": ["@wonderwhy-er/desktop-commander@latest"]
    },
    "herd": {
      "command": "php",
      "args": ["/Applications/Herd.app/Contents/Resources/herd-mcp.phar"],
      "env": {
        "PATH": "/Users/username/Library/Application Support/Herd/bin:/usr/local/bin:/usr/bin:/bin",
        "SITE_PATH": "/Users/username/Documents/projects/app"
      }
    },
    "laravel-boost": {
      "command": "/Users/username/Library/Application Support/Herd/bin/php",
      "args": ["/Users/username/Documents/projects/app/artisan", "boost:mcp"]
    },
    "morfx": {
      "command": "/your/bin/path/morfx",
      "args": ["mcp"],
      "env": {
        "MORFX_AUTO_APPLY": "true"
      }
    }
  }
}
```

### Usage

**Local MCP Protocol (AI Agents)**

Ask your AI agent to transform code using morfx:

```
"Use morfx to find all functions starting with 'Get' in this Go code"
"Replace function GetUser with GetUserByID and add context parameter"
"Add a Validate method to all structs in this file"
```

**Remote HTTP API**

Use morfx from any HTTP client or web application:

```bash
# Start HTTP server
morfx serve --api-key your-secret-key --port 8080 --host 0.0.0.0

# Use from anywhere
curl -X POST http://your-server:8080/mcp \
  -H "Authorization: Bearer your-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "tools/call",
    "params": {
      "name": "query",
      "arguments": {
        "language": "go",
        "source": "package main\n\nfunc GetUser() {}",
        "query": {"type": "function", "name": "Get*"}
      }
    },
    "id": 1
  }'
```

## 🏗️ Architecture

**MCP Protocol Integration**
- JSON-RPC 2.0 communication over stdio
- Tool discovery and registration with AI agents
- Structured error handling and response formatting

**AST-Based Transformations**
- Tree-sitter parsing for surgical precision
- Language-native semantic understanding
- Context-aware code placement algorithms

**Confidence Scoring System**
- Risk assessment for every transformation
- Auto-apply based on configurable thresholds
- Factor-based explanation of decisions

**Enterprise Staging**
- Two-phase commit: stage → review → apply
- PostgreSQL audit trail with full history
- Session management and automatic cleanup

## 🎯 Core Features

### Dual Access Modes

**MCP Protocol (Local)**
- Integrated with AI agents that support MCP
- Stdio communication with JSON-RPC 2.0
- Zero-latency local transformations
- Perfect for development workflows

**HTTP API (Remote)**
- RESTful endpoints with JSON-RPC payloads
- Bearer token authentication with API keys
- CORS support for web applications
- Ideal for CI/CD, web apps, and distributed teams

### API Endpoints

**POST /mcp** - Main transformation endpoint
- Same JSON-RPC interface as local MCP
- Requires `Authorization: Bearer <api-key>` header
- Returns standard JSON-RPC 2.0 responses

**GET /health** - Health check endpoint
- Returns server status and database connectivity
- No authentication required
- Perfect for load balancer health checks

### Natural Language Queries

Find code elements using intuitive patterns:

- `{"type": "function", "name": "Get*"}` - Functions starting with "Get"
- `{"type": "struct", "name": "*User*"}` - Structs containing "User"
- `{"type": "method", "name": "Validate"}` - All Validate methods

### Safe Transformations

Transform code with built-in safety:

- **Replace**: Replace functions, structs, or methods
- **Delete**: Remove code elements safely  
- **Insert**: Add code before/after specific locations
- **Append**: Smart placement within appropriate scope

### Performance Optimization

- **< 100ms** query operations with global AST caching
- **Parallel pipeline** auto-activation for batch operations (14x speedup)
- **Intelligent batching** for multiple transformations
- **Memory efficient** streaming for large codebases

### Confidence Scoring

Every transformation includes confidence assessment:

```json
{
  "confidence": {
    "score": 0.92,
    "level": "high", 
    "factors": [
      {"name": "single_target", "impact": 0.1, "reason": "Only one match found"},
      {"name": "exported_api", "impact": -0.2, "reason": "Modifying public API"}
    ]
  }
}
```

## 🔧 Configuration

### Database Setup

```sql
-- Create database
CREATE DATABASE morfx;

-- Morfx automatically creates tables on first run
```

### Server Configuration

**MCP Protocol (Local)**

```bash
# Basic usage
morfx mcp

# Custom database
morfx mcp --db "postgres://user:pass@localhost/morfx"

# Confidence threshold
morfx mcp --auto-threshold 0.9

# Debug mode
morfx mcp --debug
```

**HTTP Server (Remote)**

```bash
# Basic HTTP server
morfx serve --api-key your-secret-key

# Production configuration
morfx serve \
  --api-key your-secret-key \
  --port 8080 \
  --host 0.0.0.0 \
  --db "postgres://user:pass@localhost/morfx" \
  --cors "https://your-domain.com"

# Development mode
morfx serve --api-key dev-key --debug
```

### Environment Variables

```bash
MORFX_AUTO_APPLY=true          # Enable auto-apply
MORFX_AUTO_THRESHOLD=0.85      # Confidence threshold (0.0-1.0)
MORFX_DEBUG=true              # Enable debug logging
```

## 🛡️ Safety & Reliability

**Staging System**
- All transformations staged before application
- 15-minute expiration for review
- Full rollback capability with audit trail

**Validation**
- Syntax validation before and after transformations
- Semantic analysis for language-specific safety
- Graceful degradation on parsing errors

**Error Recovery**
- Comprehensive error categorization
- Automatic retry with exponential backoff
- Session isolation prevents cascading failures

## 🚀 Use Cases

**Development Workflows**
- Local AI agent integration for real-time code assistance
- IDE plugins and editor extensions
- Interactive development with immediate feedback

**CI/CD Pipelines**
- Automated code refactoring in build pipelines
- Bulk transformations across repositories
- Code quality enforcement and standardization

**Web Applications**
- Code transformation services in web IDEs
- SaaS platforms offering code analysis
- Educational platforms teaching refactoring patterns

**Enterprise Integration**
- Internal developer tools and platforms
- Code migration and modernization projects
- Distributed development team collaboration

## 🚀 Performance

Morfx is designed for production workloads:

- **Query Operations**: Sub-100ms response time
- **Transform Operations**: <200ms for single targets
- **Memory Usage**: <50MB for typical operations
- **Concurrency**: Full session isolation
- **Scalability**: Stateless server design

## 🧪 Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Integration tests (requires database)
go test -tags=integration ./...
```

## 📋 Requirements

**Runtime**
- Go 1.24 or later
- PostgreSQL 14 or later
- 512MB RAM minimum
- Linux/macOS/Windows

**AI Integration**
- Any MCP-compatible AI agent
- MCP protocol support
- JSON-RPC 2.0 compatible client

## 🤝 Contributing

Morfx follows semantic versioning and conventional commits:

```bash
# Format: type(scope): description
feat(provider): add Python language support
fix(staging): resolve expiration cleanup race condition
docs(readme): update installation instructions
```

**Development Setup**

```bash
# Install development dependencies
go mod download

# Run linter
golangci-lint run

# Start development server
go run cmd/morfx/main.go mcp --debug
```

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🔗 Links

- **Documentation**: [GitHub Wiki](https://github.com/termfx/morfx/wiki)
- **MCP Protocol**: [Model Context Protocol](https://github.com/anthropics/mcp)
- **AI Agents**: [Compatible MCP clients](https://github.com/anthropics/mcp#implementations)
- **Tree-sitter**: [Grammar Documentation](https://tree-sitter.github.io/tree-sitter/)

---

**Built with ❤️ for AI-driven development workflows**
