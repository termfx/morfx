#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ARTIFACT_DIR="$ROOT_DIR/artifacts/dogfood"

cd "$ROOT_DIR"

mkdir -p "$ARTIFACT_DIR"
rm -f "$ARTIFACT_DIR"/*.go "$ARTIFACT_DIR"/*.json "$ARTIFACT_DIR"/*.txt

for bin in morfx query replace file_query apply; do
	if [[ ! -x "$ROOT_DIR/bin/$bin" ]]; then
		echo "missing binary: $ROOT_DIR/bin/$bin" >&2
		exit 1
	fi
done

"$ROOT_DIR/bin/morfx" --help > "$ARTIFACT_DIR/morfx-help.txt"
"$ROOT_DIR/bin/apply" --help > "$ARTIFACT_DIR/apply-help.txt"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/morfx-standalone.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

SAMPLE_FILE="$TMP_DIR/sample.go"
cat > "$SAMPLE_FILE" <<'EOF'
package sample

func HelloUser() string {
	return "hello"
}
EOF

cat > "$TMP_DIR/query.json" <<EOF
{"language":"go","path":"$SAMPLE_FILE","query":{"type":"function","name":"Hello*"}}
EOF
"$ROOT_DIR/bin/query" < "$TMP_DIR/query.json" > "$ARTIFACT_DIR/query.json"
grep -q '"matches"' "$ARTIFACT_DIR/query.json"
grep -q 'HelloUser' "$ARTIFACT_DIR/query.json"

cat > "$TMP_DIR/replace.json" <<EOF
{"language":"go","path":"$SAMPLE_FILE","target":{"type":"function","name":"HelloUser"},"replacement":"func HelloUser() string { return \"updated\" }"}
EOF
"$ROOT_DIR/bin/replace" < "$TMP_DIR/replace.json" > "$ARTIFACT_DIR/replace.json"
grep -q 'updated' "$SAMPLE_FILE"

cat > "$TMP_DIR/file_query.json" <<EOF
{"scope":{"path":"$TMP_DIR","include":["**/*.go"],"language":"go","max_files":10},"query":{"type":"function","name":"Hello*"}}
EOF
"$ROOT_DIR/bin/file_query" < "$TMP_DIR/file_query.json" > "$ARTIFACT_DIR/file_query.json"
grep -q '"files"' "$ARTIFACT_DIR/file_query.json"
grep -q 'HelloUser' "$ARTIFACT_DIR/file_query.json"

cp "$SAMPLE_FILE" "$ARTIFACT_DIR/sample.after.txt"

echo "Standalone smoke completed. Artifacts written to $ARTIFACT_DIR"
