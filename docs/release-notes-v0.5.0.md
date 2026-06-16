# Morfx v0.5.0

Morfx v0.5.0 promotes the public `engine` package into the canonical owner of
the staged-change lifecycle and aligns the public release surface with that
engine-first model.

## Highlights

- Engine-native staged lifecycle for create, list, inspect, apply, and expire.
- Shared staged-apply validation for digests, expiry, root policy, and status
  across standalone and MCP entry points.
- Public docs updated to describe the engine-managed stage store and current
  `apply` flags.
- Public-surface guard extended to keep the release-note timeline and `develop`
  CI validation aligned with the current release.

## Verification

- Focused public-surface tests for release timeline, staging docs, and CI
  branch coverage.
- Full repository verification through the standard `make verify` gate.
