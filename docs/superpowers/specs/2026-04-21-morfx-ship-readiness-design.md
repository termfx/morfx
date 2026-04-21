# Morfx Ship-Readiness Design

Date: 2026-04-21
Status: Proposed
Owner: Codex + user

## Goal

Make Morfx genuinely shippable, mature, and production-ready without a full rewrite.

The chosen direction is to keep Morfx in Go, fix the architectural weaknesses that are currently blocking confidence, and preserve a clean future option to swap the parser core to Rust or Zig if the isolated seam still proves painful after the refactor.

## Problem Statement

Morfx has a credible product shape:

- a shared AST transformation engine,
- an MCP server surface,
- standalone JSON tools,
- staging, safety, and audit capabilities,
- multi-language provider support.

The current blockers are narrower than "the repo is bad":

1. The parser dependency leaks into core types and provider contracts.
2. MCP and standalone surfaces compose runtime dependencies separately.
3. Standalone command binaries duplicate too much lifecycle and formatting logic.
4. Cross-platform readiness is not yet honest or repeatable.
5. Quality and CI configuration has drifted from the current repo shape.

These problems make Morfx harder to ship, harder to evolve, and harder to evaluate fairly against a future rewrite.

## Non-Goals

- Full Rust rewrite in this phase.
- Full Zig rewrite in this phase.
- Broad feature expansion unrelated to ship-readiness.
- Reworking the product positioning beyond tightening the existing surfaces and claims.
- Large speculative performance work without evidence of a current bottleneck.

## Success Criteria

Morfx is considered ready for the first ship-ready milestone when all of the following are true:

1. The parser backend is isolated behind a real internal seam and no parser-native type leaks into core product contracts.
2. MCP and standalone surfaces share one runtime composition path for provider registration and file-processor setup.
3. Standalone tools preserve their current JSON contracts while using shared execution and formatting helpers.
4. The advertised support matrix is backed by repeatable verification commands and CI workflows.
5. README, scripts, workflows, and repo quality gates agree with the actual product behavior.

## Decision

Proceed with a Go-hardening architecture program, not a rewrite program.

### Why this decision

- The current product shell is mostly salvageable.
- The main architectural weakness is concentrated at the parser seam.
- Rewrites would force Morfx to re-earn confidence in MCP behavior, standalone contracts, staging behavior, and packaging at the same time.
- Isolating the seam now improves the current Go codebase and also improves any later Rust/Zig experiment.

### Deferred decision

After the parser seam is isolated and the runtime is stable, revisit whether the parser core should remain in Go or move to Rust or Zig. That decision should be based on:

- maintenance burden,
- cross-platform reliability,
- parser binding complexity,
- packaging simplicity,
- benchmarked transform/query behavior on real fixtures.

## Proposed Architecture

### 1. Parser Isolation Layer

Create an internal parser adapter boundary owned by the provider subsystem.

#### Target shape

- `core` contains semantic query/transform inputs and outputs only.
- `core` does not import or expose `tree-sitter` types.
- provider code can depend on parser-native node/tree types internally, but only behind adapter interfaces or implementation-local structs.
- provider-facing semantic matches are converted before crossing back into `core`.

#### Required changes

- Remove parser-native fields such as `*sitter.Node` from core-level types.
- Introduce provider-internal match/target structures that can carry parser-native state without leaking across package boundaries.
- Move parser selection and parser lifecycle management into a parser-focused internal package or provider-owned subpackage.

#### Result

Morfx gains a real backend seam. Go remains viable, and Rust/Zig become optional implementation replacements instead of repo-wide rewrites.

### 2. Shared Runtime Composition

Unify provider registration and runtime wiring so MCP and standalone tools compose the same dependencies in one place.

#### Target shape

- one shared runtime builder package creates:
  - provider registry,
  - file processor,
  - transaction log setup,
  - parser-backed capabilities,
  - optional persistence hooks.
- MCP server uses that builder.
- standalone tools use that builder.

#### Required changes

- eliminate duplicated provider registration logic across standalone and MCP boot paths,
- centralize the mapping from supported languages to provider instances,
- centralize file-processor creation and transaction-log defaults.

#### Result

Changes to providers, parser backend, or platform setup happen once and affect both product surfaces consistently.

### 3. Standalone Command Surface Consolidation

Keep the current standalone tool product model, but consolidate repeated command execution logic.

#### Target shape

- shared helpers for:
  - environment startup,
  - JSON request loading,
  - provider lookup,
  - source/path resolution,
  - transform/query execution,
  - optional file writeback,
  - confidence formatting,
  - common success and error envelopes.
- individual commands keep only their input schema differences and command-specific response wording.

#### Result

The standalone tools stay first-class, but command behavior cannot drift as easily across binaries.

### 4. Honest Ship Contract

Define one explicit release bar and make docs and workflows match it.

#### Required release bar

- build passes for supported targets,
- `go test ./...` passes for supported targets,
- standalone smoke passes on supported targets,
- MCP startup and core operation smoke passes,
- packaging scripts produce real release artifacts,
- CI enforces the same checks the README claims,
- release docs describe the real support matrix and prerequisites.

#### Support statement rule

Morfx should only claim support for a platform or workflow when:

- the build path exists,
- the verification path exists,
- CI or a documented local verification path exercises it,
- known prerequisites are documented.

### 5. Quality and Config Realignment

Bring linting, CI, Go versioning, exclusions, and scripts into sync with the actual repo.

#### Required changes

- align `.golangci.yml` with the current repository layout and Go version,
- remove stale exclusions that refer to non-existent directories or old shapes,
- ensure workflows, scripts, and local commands all point to the same supported verification flows,
- keep `third_party` exclusions only where intentional.

#### Result

Quality gates become trustworthy signals instead of historical residue.

## Component Plan

### Core

Responsibilities after refactor:

- semantic query/transform types,
- file-scope orchestration,
- safety and transaction behavior,
- parser-agnostic result contracts.

Must not own:

- parser-native node types,
- parser-library-specific lifecycle assumptions.

### Provider Subsystem

Responsibilities after refactor:

- parser-backed syntax traversal,
- language-specific node mapping and extraction,
- semantic conversion from parser matches into core results,
- parser cache and parser pool ownership.

### Runtime Composition

Responsibilities after refactor:

- provider registration,
- file processor construction,
- environment defaults,
- bootstrapping both standalone and MCP surfaces.

### MCP Surface

Responsibilities after refactor:

- protocol handling,
- request validation,
- progress and cancellation handling,
- formatting MCP-specific response shapes.

Must not duplicate provider/runtime setup logic beyond MCP-specific concerns.

### Standalone Surface

Responsibilities after refactor:

- command-specific input parsing,
- stdin/stdout JSON contract handling,
- command-specific response shaping.

Must not duplicate provider/runtime wiring or core transform lifecycle logic.

## Data Flow

### Query flow

1. MCP tool or standalone command receives request.
2. Shared runtime resolves environment and language provider.
3. Provider uses parser adapter to parse/traverse source.
4. Provider maps parser-native matches to semantic results.
5. Surface formats the result for MCP or standalone JSON output.

### Transform flow

1. MCP tool or standalone command receives transform request.
2. Shared runtime resolves source and provider.
3. Provider uses parser adapter to find targets and produce modified source.
4. Core/file processor applies safety, transaction, and write rules.
5. Surface returns semantic result plus surface-specific metadata.

## Error Handling

### Rules

- Parser errors must surface as semantic transform/query errors, not backend-specific leakage.
- Platform/toolchain prerequisite failures must be explicit and actionable.
- Runtime composition failures must happen in one place, not with different messages across MCP and standalone.
- File-write and staging failures must continue to preserve transaction integrity.

### Examples

- unsupported language -> consistent error from shared runtime/provider registry.
- parser backend unavailable -> clear startup or provider initialization failure.
- path read/write failure -> consistent structured error in standalone and mapped MCP error in MCP mode.
- low-confidence or safety rejection -> semantic refusal with preserved diagnostics.

## Testing Strategy

### Required test layers

1. **Unit tests**
   - parser adapter boundaries,
   - provider semantic mapping,
   - shared runtime builder,
   - command helpers.

2. **Contract tests**
   - standalone JSON request/response envelopes,
   - MCP tool behavior for core operations,
   - language capability discovery.

3. **Cross-platform verification**
   - Windows,
   - Linux,
   - macOS.

4. **Smoke tests**
   - standalone tool fixture flow,
   - MCP startup plus representative query/transform path,
   - packaging scripts.

5. **Regression tests**
   - Windows-specific file permission and JSON/BOM issues,
   - parser initialization failures,
   - file glob/path behavior across OS boundaries,
   - transaction rollback edge cases.

### Benchmark policy

Performance tuning is secondary in this phase. Keep benchmarks for regression detection, but do not expand optimization work unless a measured hot path affects the ship-ready bar.

## Rollout Plan

### Phase 1: Architecture Stabilization

- isolate parser-native types from `core`,
- introduce shared runtime composition,
- reduce standalone command duplication without changing public contracts.

### Phase 2: Platform and Verification Hardening

- fix current Windows/platform blockers,
- align CI and release workflows with supported targets,
- ensure local verification scripts are real and documented.

### Phase 3: Product Contract Cleanup

- update README and docs to match the real support matrix,
- tighten release notes and packaging behavior,
- remove misleading claims and stale configuration drift.

### Phase 4: Rewrite Re-evaluation Gate

After the above is green:

- measure maintenance complexity of the isolated Go parser seam,
- optionally spike Rust or Zig for the parser core only,
- decide from evidence, not frustration, whether replacement is justified.

## Risks

### Risk: seam refactor spreads too broadly

Mitigation:

- change boundaries first, behavior second,
- preserve current JSON/MCP contracts,
- keep parser replacement out of scope during seam isolation.

### Risk: command consolidation accidentally changes behavior

Mitigation:

- add contract tests before consolidating helpers,
- keep command-specific text differences where product-visible.

### Risk: platform fixes keep uncovering hidden assumptions

Mitigation:

- treat platform verification as a first-class workstream,
- convert discovered platform assumptions into regression tests and scripts.

### Risk: rewrite pressure returns before isolation is complete

Mitigation:

- set a formal re-evaluation checkpoint after Phase 2 or Phase 3,
- defer rewrite decisions until the seam is explicit and measured.

## Implementation Assumptions

These assumptions should be used during planning and implementation unless new evidence forces a change:

1. The intended support matrix remains Windows, Linux, and macOS. If one platform cannot meet the same bar in the first pass, the docs and release contract must narrow claims explicitly rather than implying equal support.
2. SQLite remains the default local persistence backend after the hardening pass, with remote/libsql support treated as an optional extension rather than the baseline path.
3. Any future Rust or Zig parser experiment starts with one reference language and one query/transform path, not the full provider matrix.

## Recommendation

Execute this as one coordinated ship-readiness program in Go.

The correct product and architectural decision is:

- preserve the current Morfx product model,
- isolate the parser seam,
- unify runtime composition,
- reduce surface duplication,
- make platform support and quality gates honest,
- only then reconsider Rust or Zig for the parser core.
