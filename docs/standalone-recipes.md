# Standalone Morfx Recipes

This guide focuses on practical shell usage for the standalone Morfx binaries
and on the same workflows when they are orchestrated by TFX.

## Build the tools

```bash
make build-standalone
```

The binaries land in `bin/`:

- `bin/morfx`
- `bin/query`
- `bin/replace`
- `bin/delete`
- `bin/insert_before`
- `bin/insert_after`
- `bin/append`
- `bin/file_query`
- `bin/file_replace`
- `bin/file_delete`
- `bin/apply`

## Quick shell recipes

### Find all exported Go functions

```bash
cat <<'JSON' | ./bin/query
{"language":"go","path":"./cmd/morfx/main.go","query":{"type":"function","name":"*"}}
JSON
```

Use `source` instead of `path` when you want to query generated text from a
shell pipeline.

### Search a whole tree

```bash
cat <<'JSON' | ./bin/file_query
{"scope":{"path":".","include":["**/*.go"],"exclude":["vendor/**"],"language":"go","max_files":200},"query":{"type":"function","name":"Handle*"}}
JSON
```

### Replace one function safely

```bash
cat <<'JSON' | ./bin/replace
{"language":"go","path":"./pkg/example.go","target":{"type":"function","name":"Hello"},"replacement":"func Hello() string {\n\treturn \"updated\"\n}"}
JSON
```

### Remove a code block

```bash
cat <<'JSON' | ./bin/delete
{"language":"go","path":"./pkg/example.go","target":{"type":"function","name":"Legacy"}}
JSON
```

### Apply staged work

```bash
cat <<'JSON' | ./bin/apply
{"latest":true}
JSON
```

### Run a repeatable recipe

Use `recipe` when you want a named transformation that can be checked into a
repo, reused by an agent, or wrapped by TFX.

```bash
cat <<'JSON' | ./bin/recipe
{
  "name": "replace-legacy-handlers",
  "dry_run": true,
  "min_confidence": 0.85,
  "steps": [
    {
      "name": "replace legacy handlers",
      "method": "replace",
      "scope": {
        "path": ".",
        "include": ["**/*.go"],
        "exclude": ["vendor/**"],
        "language": "go",
        "max_files": 100
      },
      "target": {"type": "function", "name": "Legacy*"},
      "replacement": "func Replacement() {}"
    }
  ]
}
JSON
```

Set `dry_run` to `false` only after reviewing the result. Apply-mode recipes
still preflight first and stop before mutation when a step falls below
`min_confidence`.

## Local automation patterns

### Preflight a refactor

Run a read-only search first, then decide if the replacement is narrow enough:

```bash
cat <<'JSON' | ./bin/file_query
{"scope":{"path":".","include":["**/*.go"],"exclude":["vendor/**"],"language":"go"},"query":{"type":"function","name":"Old*"}}
JSON
```

If the output is narrow and deterministic, run the mutation next.

### Pair `replace` with `git diff`

```bash
cat <<'JSON' | ./bin/replace
{"language":"go","path":"./pkg/example.go","target":{"type":"function","name":"Hello"},"replacement":"func Hello() string {\n\treturn \"updated\"\n}"}
JSON

git diff -- ./pkg/example.go
```

### Use `file_replace` for bulk changes

```bash
cat <<'JSON' | ./bin/file_replace
{"scope":{"path":".","include":["**/*.go"],"exclude":["vendor/**"],"language":"go"},"target":{"type":"function","name":"Debug*"},"replacement":"func Debug() {}","dry_run":true,"backup":false}
JSON
```

Set `dry_run` to `false` only after validating the scope and diff.

## TFX usage

TFX works well when you want a repeatable wrapper around the standalone tools.
This repository already ships a root `tfx.yaml` with flows for CI, standalone
smoke checks, quality, and release packaging.

### Run the standalone flow

```bash
tfx --flow standalone --run
```

That flow builds the local binaries and runs the fixture smoke script that
exercises `query`, `replace`, and `file_query`.

### Dogfood against a local TFX checkout

```bash
tfx --flow dogfood-tfx --run
```

That flow expects a local checkout at `../tfx` by default. Override it with
`MORFX_DOGFOOD_TFX_DIR=/path/to/tfx` when your workspace layout differs.

### Run strict verification

```bash
tfx --flow quality --run
```

Use this when you want the repository-level checks plus the dogfood artifacts.

### Run the release flow

```bash
tfx --flow release --lane canary --run
```

That path is useful when you want to validate packaging and the final release
bundle before cutting a tag.

## Practical rules

- Query first, mutate second.
- Prefer `file_query` and `file_replace` for tree-wide work.
- Prefer `recipe` when the same transformation will be reused.
- Keep shell recipes explicit about `path`, `scope`, and `language`.
- Use TFX when the same sequence needs to be repeatable across runs.
