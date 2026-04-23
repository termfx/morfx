# Standalone Tool Binaries

Each command-line tool reads a single JSON document from stdin and emits a JSON
response to stdout. All tools support `-h`/`--help` to print the same
information summarised below.

These are not flag-first CLIs. In normal use, the only flags you should expect
to reach for are help flags and the occasional tool-specific control flag such
as `apply --db ...`. The actual query or transform payload still travels over
stdin/stdout as JSON.

Build all standalone binaries locally with:

```bash
make build-standalone
```

This repository also ships a root [`tfx.yaml`](../tfx.yaml) so you can dogfood
Morfx through TFX:

```bash
tfx --flow standalone --run
tfx --flow dogfood-tfx --run
```

That flow builds the local binaries and runs
[`tools/scripts/smoke-standalone.sh`](../tools/scripts/smoke-standalone.sh)
against a temporary fixture.

In that setup, Morfx remains the code-transformation engine and TFX acts as the
runtime that wraps it with prompts, flow control, logs, step progress, and
artifact handling.

For more practical shell recipes and the external TFX dogfood path, see
[`standalone-recipes.md`](./standalone-recipes.md).

## `query`
- **Purpose:** Locate code elements that match a `core.AgentQuery`.
- **Input:**
  ```json
  {
    "language": "go",
    "source": "...",          // or "path": "file.go"
    "query": { /* AgentQuery */ }
  }
  ```
  Exactly one of `source` or `path` must be supplied.
- **Output:**
  ```json
  {
    "content": [{"type": "text", "text": "summary"}],
    "matches": 2,
    "results": [/* core.Match */],
    "path": "file.go"        // present only when a file was read
  }
  ```

## `replace`
- **Purpose:** Replace AST-matched code elements with new content.
- **Input:**
  ```json
  {
    "language": "go",
    "source": "...",          // or "path": "file.go"
    "target": { /* AgentQuery */ },
    "replacement": "..."
  }
  ```
- **Output:**
  ```json
  {
    "content": [{"type": "text", "text": "summary"}],
    "matches": 1,
    "diff": "...",
    "confidence": { /* core.ConfidenceScore */ },
    "modified": "...",
    "path": "file.go",
    "applied": true            // true when file writes occurred
  }
  ```

## `delete`
- **Purpose:** Remove code elements identified by a query.
- **Input:** Same structure as `replace` without `replacement`.
- **Output:**
  ```json
  {
    "content": [{"type": "text", "text": "summary"}],
    "matches": 1,
    "diff": "...",
    "confidence": { /* score */ },
    "modified": "...",
    "path": "file.go",
    "applied": true
  }
  ```

## `insert_before` / `insert_after`
- **Purpose:** Insert material relative to matched elements.
- **Input:**
  ```json
  {
    "language": "go",
    "path": "file.go",       // or "source"
    "target": { /* AgentQuery */ },
    "content": "snippet"
  }
  ```
- **Output:** Same envelope as `replace`, including `diff`, `confidence`,
  `modified`, and optional `path`/`applied` flags.

## `append`
- **Purpose:** Append content either to a specific target scope or a provider
  chosen location when `target` is omitted.
- **Input:**
  ```json
  {
    "language": "go",
    "path": "file.go",       // or "source"
    "target": { /* optional AgentQuery */ },
    "content": "snippet"
  }
  ```
- **Output:** Same response keys as the other single-file mutation tools.

## `file_query`
- **Purpose:** Search for matches across multiple files.
- **Input:**
  ```json
  {
    "scope": {
      "path": "./src",
      "include": ["**/*.go"],
      "exclude": ["vendor/**"],
      "language": "go",
      "max_files": 100
    },
    "query": { /* AgentQuery */ }
  }
  ```
- **Output:**
  ```json
  {
    "content": [{"type": "text", "text": "summary"}],
    "matches": 5,
    "files": 3,
    "results": [/* core.FileMatch */]
  }
  ```

## `file_replace`
- **Purpose:** Perform replacements across a file set.
- **Input:**
  ```json
  {
    "scope": { /* FileScope */ },
    "target": { /* AgentQuery */ },
    "replacement": "snippet",
    "dry_run": false,
    "backup": false
  }
  ```
- **Output:**
  ```json
  {
    "content": [{"type": "text", "text": "summary"}],
    "files_processed": 10,
    "files_modified": 2,
    "matches": 4,
    "dry_run": false,
    "errors": ["..."] ,
    "transaction": "tx-123",
    "details": [/* core.FileTransformDetail */]
  }
  ```

## `file_delete`
- **Purpose:** Delete matches across multiple files. Shares the same input and
  output contract as `file_replace` minus the `replacement` field.

## `apply`
- **Purpose:** Apply staged transformations stored in the Morfx database.
- **Flags:** `--db` to select the SQLite/Turso DSN (default `./.morfx/db/morfx.db`).
- **Input:**
  ```json
  {
    "id": "stg_123",   // apply a specific stage
    "all": false,
    "latest": false,
    "session_id": "ses_456"
  }
  ```
  Only one of `id`, `all`, or `latest` may be true. When none are set the tool
  defaults to `latest`.
- **Output:**
  ```json
  {
    "content": [{"type": "text", "text": "summary"}],
    "applied": ["stg_123"],
    "structuredContent": {
      "mode": "single",
      "applied": ["stg_123"]
    }
  }
  ```

All tools emit errors using the shared envelope
`{"error": {"message": "...", "details": "..."}}` when anything goes wrong.
