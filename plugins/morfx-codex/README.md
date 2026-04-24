# Morfx Codex Plugin

This is a thin Codex plugin wrapper around Morfx. The plugin does not reimplement Morfx; it exposes the existing Morfx MCP server and adds a Codex skill for AST-safe code transformation workflows.

## Requirements

Install or build Morfx so the `morfx` command is available to the Codex host:

```bash
morfx mcp
```

From this repository, a local development build is available at `bin/morfx.exe` on Windows after running the normal build flow.

## Components

- `.codex-plugin/plugin.json` defines the plugin metadata.
- `.mcp.json` registers the `morfx` MCP server.
- `skills/morfx/SKILL.md` describes when Codex should prefer Morfx over text-only edits.

## Local Marketplace

The repo-local marketplace entry lives at:

```text
.agents/plugins/marketplace.json
```
