# Morfx v0.2.0

Morfx now has first-class recipes for repeatable AST transformations.

## Highlights

- New `recipe` standalone JSON tool.
- New MCP `recipe` tool for agent-driven repeatable transformations.
- Recipe schema with named steps, file scopes, targets, transform methods, and
  confidence gates.
- Apply-mode recipes run a dry-run preflight first and stop before mutation when
  a step falls below `min_confidence`.
- Windows and Unix standalone smoke checks now cover recipe dry-run behavior.
- Release packaging now includes the `recipe` binary.

## Verification

- `go test -timeout=120s ./...`
- `tools/scripts/build-standalone.ps1`
- `tools/scripts/smoke-standalone.ps1`
