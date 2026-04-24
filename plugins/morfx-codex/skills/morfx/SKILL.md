---
name: morfx
description: Use Morfx when Codex needs deterministic AST-based code queries, replacements, staged edits, or recipe-driven refactors through the Morfx MCP server instead of brittle text-only matching.
---

# Morfx

Use Morfx when a code change benefits from AST-aware matching or safer rewrite semantics:

- Search for code structure, not just text.
- Replace, delete, insert, or stage edits around specific AST nodes.
- Run a repeatable Morfx recipe.
- Inspect supported query types for the active language before writing a broad transform.

## MCP Server

This plugin exposes the `morfx` MCP server. It expects the `morfx` binary to be installed on `PATH` or otherwise resolvable by the host environment.

The MCP command is:

```bash
morfx mcp
```

## Usage Guidance

Prefer Morfx over raw text replacement when the target is syntax-aware, repeated across files, or risky to match by string alone. Keep changes bounded: query first, inspect matches, then apply the smallest replacement or recipe that proves the intended transformation.

Use `dsl` on read tools (`query`, `file_query`) and `target_dsl` on mutation tools (`replace`, `delete`, `insert_before`, `insert_after`, `append`, `file_replace`, `file_delete`, `recipe`) when the target is structural.

Morfx DSL syntax:

- `kind:name` selects an AST element by provider-owned kind and name. `*` is a wildcard.
- `>` means descendant containment, for example `func:* > call:os.Getenv`.
- `&`, `|`, `!`, and parentheses compose selectors.
- Attributes use `key=value`, and shorthand attributes map to provider-owned defaults. Example: `struct:* > field:Secret type=string`.
- Common selector kinds include `func`, `def`, `function`, `method`, `class`, `struct`, `interface`, `field`, `call`, `return`, `assignment`, `condition`, `block`, `loop`, and `import`.

Examples:

```json
{"language":"go","path":"./config.go","dsl":"func:* > call:os.Getenv"}
```

```json
{"scope":{"path":".","include":["**/*.go"],"language":"go"},"target_dsl":"func:Legacy*","replacement":"func Current() {}","dry_run":true}
```

For direct local debugging outside MCP, the same engine is available through Morfx standalone commands such as `query`, `replace`, `file_query`, `recipe`, and `apply`.
