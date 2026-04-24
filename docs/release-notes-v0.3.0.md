# Morfx v0.3.0

Morfx now ships a compact structural DSL for AST-safe agent workflows.

## Highlights

- Added `dsl` selectors for read tools and `target_dsl` selectors for mutation
  tools across MCP, standalone JSON tools, and recipes.
- Added nested AST matching with `>`, logical composition with `&`, `|`, `!`,
  parentheses, wildcard names, and selector attributes such as
  `field:Secret type=string`.
- Expanded provider-owned selector vocabularies for Go, JavaScript,
  TypeScript, PHP, and Python, including calls, returns, assignments,
  conditions, blocks, loops, fields, functions, methods, and classes.
- Exposed DSL guidance through MCP tool schemas and the `docs://dsl` MCP
  resource so agents can learn the syntax from `tools/list` and
  `resources/read`.
- Added contributor documentation for language providers and release packaging
  now includes the DSL and provider-contribution docs.
- Added a Codex plugin wrapper with a Morfx skill for AST-safe transformation
  workflows.

## Verification

- `go test ./mcp/tools ./mcp/resources ./mcp -count=1`
- `CGO_ENABLED=1 CC='zig cc -target x86_64-windows-gnu' go test ./...`
- `tools/scripts/build-standalone.ps1`
- `tools/scripts/smoke-standalone.ps1`
- `git diff --check`
