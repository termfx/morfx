# Morfx v1.5.1

Morfx now ships as a cleaner product, not just an MCP backend.

## Highlights

- Embedded build metadata for `morfx --version`, plus the same version surfaced
  through the MCP handshake and server resources.
- Standalone binaries are treated as first-class release artifacts instead of
  side tools that only exist in local builds.
- The repository now dogfoods itself through TFX with root flows for `ci`,
  `standalone`, `dogfood-tfx`, `quality`, and `release`.
- New practical docs cover standalone shell recipes and TFX orchestration.
- Naming drift to the old `termfx` module path has been removed from build and
  lint configuration.

## Release contents

- `morfx`
- `query`
- `replace`
- `delete`
- `insert_before`
- `insert_after`
- `append`
- `file_query`
- `file_replace`
- `file_delete`
- `apply`

## Verification

- `make verify`
- `tfx --flow standalone --run`
- `tfx --flow dogfood-tfx --run`
- `tfx --flow quality --run`
- `tfx --flow release --lane canary --run`
