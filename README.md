# Morfx

> Deterministic AST-based refactoring for AI agents and automation, via MCP, standalone JSON tools, and an optional TFX runtime.

Morfx is a deterministic AST transformation engine that ships both an MCP server
and standalone JSON tools. It gives AI coding agents and local automation the
ability to query, transform, and safely modify code using tree-sitter AST
analysis — not regex, not string matching.

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

## Modes

Morfx ships in three operational shapes:

- **`morfx mcp`** for MCP-compatible AI clients such as Claude Desktop or Codex
- **Standalone binaries** such as `query`, `replace`, `file_query`, and `apply`
  for direct local automation
- **A TFX-orchestrated runtime** for repeatable flows such as smoke checks,
  dogfooding, quality gates, and release packaging

For the standalone stdin/stdout contracts, see
[docs/standalone-tools.md](./docs/standalone-tools.md). For shell-level usage
patterns and TFX recipes, see
[docs/standalone-recipes.md](./docs/standalone-recipes.md).

## Morfx vs TFX

Morfx and TFX solve different problems:

- **Morfx** is the refactoring engine: AST query, replace, delete, insert, and
  staged apply.
- **TFX** is the terminal runtime around tools like Morfx: prompts, flow
  selection, progress, logs, artifacts, and release-style orchestration.

That means TFX does not replace Morfx. It hosts Morfx more effectively when the
same standalone commands need to run as a product workflow instead of as ad-hoc
shell snippets.

## Install

### From source

```bash
git clone https://github.com/oxhq/morfx.git
cd morfx
go build -o bin/morfx ./cmd/morfx
make build-standalone
```

### Quick install

```bash
go install github.com/oxhq/morfx/cmd/morfx@latest
```

### Windows build notes

Windows support is practical, but current provider builds require `CGO_ENABLED=1` and a compatible C compiler for tree-sitter grammars. The verified path in this repo uses Zig; set `CC` before running Go commands:

```powershell
$env:CGO_ENABLED = "1"
$env:CC = "zig cc -target x86_64-windows-gnu"
go build ./cmd/morfx
go test ./...
```

For the canonical Windows verification path, run `.\tools\scripts\verify-windows.ps1`. For the standalone smoke path, run `.\tools\scripts\smoke-standalone.ps1`. The other PowerShell helpers in `tools/scripts/` are the Windows-native path; the Unix `make` targets and `.sh` scripts remain for macOS/Linux.

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

## Standalone Tools

The standalone binaries are **JSON-over-stdin tools**, not flag-heavy CLIs.
Outside of `-h`/`--help`, you normally send them one JSON request on stdin and
read one JSON response from stdout.

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

Build them locally with:

```bash
make build-standalone
```

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

## TFX Dogfooding

This repository ships a root [`tfx.yaml`](./tfx.yaml) so Morfx can run as a
product workflow, not only as an MCP backend or a pile of standalone binaries.

Typical flows:

```bash
tfx --flow ci --run
tfx --flow standalone --run
tfx --flow dogfood-tfx --run
tfx --flow quality --run
tfx --flow release --lane canary --run
```

The `standalone` flow builds the local binaries and exercises `query`,
`replace`, and `file_query` against a temporary fixture through
[`tools/scripts/smoke-standalone.sh`](./tools/scripts/smoke-standalone.sh).

The `dogfood-tfx` flow targets a local checkout of `oxhq/tfx` through
[`tools/scripts/dogfood-tfx.sh`](./tools/scripts/dogfood-tfx.sh), running
read-only queries on the real repo and a safe replacement against a temporary
copy of a TFX source file. Override the target checkout with
`MORFX_DOGFOOD_TFX_DIR=/path/to/tfx`.

The important boundary is:

- Morfx owns AST edits and standalone JSON contracts.
- TFX owns flow control, prompts, runtime logs, progress, hooks, and artifacts.

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
make build-standalone       # Build standalone JSON tools
make smoke-standalone       # Run standalone fixture smoke tests
make verify                 # Strict local verification
```

Windows equivalents:

```powershell
$env:CGO_ENABLED = "1"
$env:CC = "zig cc -target x86_64-windows-gnu"
.\tools\scripts\build-standalone.ps1
.\tools\scripts\verify-windows.ps1
.\tools\scripts\smoke-standalone.ps1
```

## License

MIT
