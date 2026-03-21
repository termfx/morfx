# Morfx

> Deterministic AST-based code transformations for AI agents.

Morfx is an MCP server that gives AI coding agents (Claude Desktop, OpenAI Codex, Cursor, etc.) the ability to query, transform, and safely modify code using tree-sitter AST analysis — not regex, not string matching.

## Why

AI agents edit code by generating diffs or full-file rewrites. That works until it doesn't: off-by-one replacements, broken scope, wrong indentation, silent failures. Morfx solves this by operating on the actual syntax tree. Every transform is targeted by AST node, confidence-scored, and optionally staged for review before applying.

## What it does

- **Query** — Find functions, structs, classes, methods, interfaces by type and name pattern (wildcards supported)
- **Replace** — Swap a matched code element with new code
- **Delete** — Remove a matched element cleanly
- **Insert before / after** — Add code relative to a target element
- **Append** — Smart placement at end of file or scope
- **Stage / Apply / Rollback** — Two-phase commit with SQLite audit trail
- **Confidence scoring** — Every transform gets a score with explainable factors
- **Multi-language** — Go, JavaScript, TypeScript, PHP, Python via tree-sitter

## Install

### From source

```bash
git clone https://github.com/termfx/morfx.git
cd morfx
go build -o morfx cmd/morfx/main.go
```

### Quick install

```bash
go install github.com/termfx/morfx/cmd/morfx@latest
```

## Setup

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "morfx": {
      "command": "/path/to/morfx",
      "args": ["mcp"],
      "env": {
        "HOME": "/Users/you",
        "NO_COLOR": "1"
      }
    }
  }
}
```

### OpenAI Codex

Add to `~/.codex/config.toml`:

```toml
[mcp_servers.morfx]
command = "/path/to/morfx"
args = ["mcp"]

[mcp_servers.morfx.env]
HOME = "/Users/you"
NO_COLOR = "1"
```

### Any MCP client

Morfx speaks MCP 2025-11-25 over stdio. Point any MCP-compatible client at the binary with `mcp` as the subcommand.

## Tools

| Tool | Description |
|---|---|
| `query` | Find code elements by type and name pattern |
| `file_query` | Search across multiple files |
| `replace` | Replace matched elements with new code |
| `file_replace` | Replace across multiple files |
| `delete` | Remove matched elements |
| `file_delete` | Delete across multiple files |
| `insert_before` | Insert code before a matched element |
| `insert_after` | Insert code after a matched element |
| `append` | Smart-place code at end of file or scope |
| `apply` | Apply a staged transformation |

## Usage examples

These are the JSON payloads your AI agent sends. You don't type these — your agent does.

**Find all functions matching a pattern:**
```json
{
  "language": "go",
  "path": "/project/handlers.go",
  "query": {"type": "function", "name": "Handle*"}
}
```

**Replace a method:**
```json
{
  "language": "php",
  "path": "/project/app/Http/Controllers/UserController.php",
  "target": {"type": "method", "name": "store"},
  "replacement": "public function store(StoreUserRequest $request): JsonResponse\n{\n    return new UserResource(User::create($request->validated()));\n}"
}
```

**Insert a comment before a function:**
```json
{
  "language": "go",
  "path": "/project/service.go",
  "target": {"type": "function", "name": "DeleteUser"},
  "content": "// DeleteUser removes a user permanently. Use with caution."
}
```

## Confidence scoring

Every transformation returns a confidence score (0.0–1.0) with factors:

```
Confidence: █████████░ 90.0%
Factors:
  +0.10 single_target — Only one match found
  -0.20 exported_api — Modifying public API
```

When confidence exceeds the threshold (default 0.85), transforms auto-apply. Below threshold, they stage for manual review via `apply`.

## Configuration

```bash
morfx mcp                           # Start with defaults
morfx mcp --debug                   # Debug logging to stderr
morfx mcp --db ./my.db              # Custom SQLite path
morfx mcp --auto-threshold 0.9      # Stricter auto-apply
```

## Supported languages

| Language | Provider | Query types |
|---|---|---|
| Go | tree-sitter-go | function, struct, interface, variable, constant, import, type, method, field |
| JavaScript | tree-sitter-javascript | function, class, variable, import, export |
| TypeScript | tree-sitter-typescript | function, class, interface, type, variable, import, export, enum |
| PHP | tree-sitter-php | function, class, method, interface, trait, variable, namespace |
| Python | tree-sitter-python | function, class, variable, import, decorator |

## Architecture

```
morfx mcp (stdio)
├── MCP Protocol (JSON-RPC 2.0, 2025-11-25 spec)
├── Tool Registry (10 tools)
├── Provider Registry
│   ├── Go provider (tree-sitter)
│   ├── JavaScript provider
│   ├── TypeScript provider
│   ├── PHP provider
│   └── Python provider
├── Base Provider (shared AST engine)
│   ├── Query (walkTree + pattern match)
│   ├── Transform (replace/delete/insert/append)
│   ├── Confidence scoring
│   └── AST cache + parser pool
├── Safety Manager (atomic writes, integrity checks)
└── Staging Manager (SQLite, stage/apply/rollback)
```

## Development

```bash
go build ./...              # Build everything
go test ./...               # Run all tests
go test -race ./...         # Race detector
go test -cover ./...        # Coverage
```

## License

MIT
