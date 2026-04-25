# Changelog

## v0.4.0

- Added deeper structural DSL matching: capture patterns, direct-child
  containment, argument predicates, source/text predicates, and sibling
  ordering predicates.
- Added agent-readable MCP capability data for DSL support and per-language
  provider selector vocabularies.
- Fixed MCP prompt retrieval so prompts listed through `prompts/list` can also
  be fetched through `prompts/get`.
- Added release `MANIFEST.json`, `SBOM.spdx.json`, and GitHub artifact
  provenance attestation generation to the release workflow.
- Cleaned the public release timeline so the repo only advertises released
  `v0.x` notes.

## v0.3.0

- Added the Morfx structural DSL for read tools, mutation tools, and recipes.
- Added provider-owned DSL alias handling so language-specific words stay
  inside the selected language provider.
- Added agent-readable DSL documentation through MCP schemas and resources.
- Added contributor documentation for adding language providers.
- Added the repo-local Codex plugin wrapper for Morfx.
- Removed internal alpha project references from the public surface and added a
  guard test to keep them out.

## v0.2.0

- Added first-class Morfx recipes: named repeatable transformations composed
  from existing AST primitives.
- Added the standalone `recipe` JSON tool and MCP `recipe` tool.
- Added dry-run preflight and confidence gates before apply-mode recipe steps
  mutate files.
- Added recipe tests, standalone smoke coverage, TFX artifact wiring, and
  release packaging support.
- Updated standalone tool documentation and recipe usage examples.

## v0.1.0

- Initial public release of Morfx.
