# Morfx v1.1.0

Morfx now ships as a clearer product, not just an MCP backend with side tools.

## Highlights

- Embedded build metadata for `morfx --version`, plus matching version data in
  MCP initialization and resource output.
- Standalone binaries are treated as first-class release artifacts:
  `query`, `replace`, `delete`, `insert_before`, `insert_after`, `append`,
  `file_query`, `file_replace`, `file_delete`, and `apply`.
- Root `tfx.yaml` adds real product flows for `ci`, `standalone`,
  `dogfood-tfx`, `quality`, and `release`.
- New standalone recipes document shell automation and TFX orchestration.
- Build and lint configuration no longer carry the old `termfx` naming drift.

## Verification

- `make verify`
- `tfx --flow standalone --run`
- `tfx --flow dogfood-tfx --run`
- `tfx --flow quality --run`
- `tfx --flow release --lane canary --run`
