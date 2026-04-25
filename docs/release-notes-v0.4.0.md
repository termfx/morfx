# Morfx v0.4.0

Morfx v0.4.0 folds the near-term roadmap into one trust-building release:
deeper DSL matching, better agent-facing capability data, a cleaner public
surface, and stronger release artifacts.

## Highlights

- DSL capture patterns such as `call:$client.$method`.
- Direct-child containment with `>>`, alongside existing descendant matching
  with `>`.
- Predicate attributes for arguments, source/text matching, and sibling order:
  `arg`, `arg0`, `source`, `text`, `before`, and `after`.
- MCP capability output now advertises DSL support and per-language selector
  vocabularies.
- MCP prompts listed through `prompts/list` can be retrieved through
  `prompts/get`.
- Releases now publish `MANIFEST.json` and `SBOM.spdx.json`, and the workflow
  requests GitHub build provenance attestations for release artifacts.

## Verification

- Focused parser, provider, MCP, and release metadata tests.
- Full repository verification through the standard release gate before
  publishing.
