#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_REPO="${MORFX_DOGFOOD_TFX_DIR:-$ROOT_DIR/../tfx}"
ARTIFACT_DIR="$ROOT_DIR/artifacts/dogfood-tfx"

cd "$ROOT_DIR"

if [[ ! -d "$TARGET_REPO" ]]; then
	echo "missing TFX repo: $TARGET_REPO" >&2
	echo "set MORFX_DOGFOOD_TFX_DIR to point at a local oxhq/tfx checkout" >&2
	exit 1
fi

if [[ ! -f "$TARGET_REPO/go.mod" ]] || ! grep -q 'github.com/oxhq/tfx' "$TARGET_REPO/go.mod"; then
	echo "target does not look like the oxhq/tfx repository: $TARGET_REPO" >&2
	exit 1
fi

mkdir -p "$ARTIFACT_DIR"
rm -f "$ARTIFACT_DIR"/*.go "$ARTIFACT_DIR"/*.json "$ARTIFACT_DIR"/*.txt

for bin in query replace file_query; do
	if [[ ! -x "$ROOT_DIR/bin/$bin" ]]; then
		echo "missing binary: $ROOT_DIR/bin/$bin" >&2
		exit 1
	fi
done

"$ROOT_DIR/bin/file_query" <<JSON > "$ARTIFACT_DIR/tfx-new-functions.json"
{"scope":{"path":"$TARGET_REPO","include":["**/*.go"],"exclude":["vendor/**"],"language":"go","max_files":250},"query":{"type":"function","name":"New*"}}
JSON
grep -q '"matches"' "$ARTIFACT_DIR/tfx-new-functions.json"
grep -q 'parseRunOptions\|NewMultiplexer\|NewThemeFromPalette' "$ARTIFACT_DIR/tfx-new-functions.json"

"$ROOT_DIR/bin/query" <<JSON > "$ARTIFACT_DIR/tfx-main-query.json"
{"language":"go","path":"$TARGET_REPO/cmd/tfx/main.go","query":{"type":"function","name":"parse*"}}
JSON
grep -q 'parseRunOptions' "$ARTIFACT_DIR/tfx-main-query.json"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/morfx-tfx.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

cp "$TARGET_REPO/runfx/api.go" "$TMP_DIR/api.go"

"$ROOT_DIR/bin/replace" <<JSON > "$ARTIFACT_DIR/tfx-runfx-replace.json"
{"language":"go","path":"$TMP_DIR/api.go","target":{"type":"function","name":"New"},"replacement":"func New() *LoopBuilder {\n\treturn &LoopBuilder{config: DefaultConfig()}\n}"}
JSON
grep -q 'LoopBuilder' "$ARTIFACT_DIR/tfx-runfx-replace.json"
grep -q 'return &LoopBuilder{config: DefaultConfig()}' "$TMP_DIR/api.go"

cp "$TMP_DIR/api.go" "$ARTIFACT_DIR/runfx-api.after.txt"

echo "External TFX dogfood completed. Artifacts written to $ARTIFACT_DIR"
