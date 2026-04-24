#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_DIR="${MORFX_BIN_DIR:-$ROOT_DIR/bin}"
ARTIFACT_DIR="${MORFX_ARTIFACT_DIR:-$ROOT_DIR/artifacts/dogfood}"

cd "$ROOT_DIR"

mkdir -p "$ARTIFACT_DIR"
rm -f "$ARTIFACT_DIR"/*.go "$ARTIFACT_DIR"/*.json "$ARTIFACT_DIR"/*.txt

for bin in morfx query replace file_query apply recipe; do
	if [[ ! -x "$BIN_DIR/$bin" ]]; then
		echo "missing binary: $BIN_DIR/$bin" >&2
		exit 1
	fi
done

"$BIN_DIR/morfx" --help > "$ARTIFACT_DIR/morfx-help.txt"
"$BIN_DIR/apply" --help > "$ARTIFACT_DIR/apply-help.txt"
"$BIN_DIR/recipe" --help > "$ARTIFACT_DIR/recipe-help.txt"

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
"$BIN_DIR/query" < "$TMP_DIR/query.json" > "$ARTIFACT_DIR/query.json"
grep -q '"matches"' "$ARTIFACT_DIR/query.json"
grep -q 'HelloUser' "$ARTIFACT_DIR/query.json"

cat > "$TMP_DIR/replace.json" <<EOF
{"language":"go","path":"$SAMPLE_FILE","target":{"type":"function","name":"HelloUser"},"replacement":"func HelloUser() string { return \"updated\" }"}
EOF
"$BIN_DIR/replace" < "$TMP_DIR/replace.json" > "$ARTIFACT_DIR/replace.json"
grep -q 'updated' "$SAMPLE_FILE"

cat > "$TMP_DIR/file_query.json" <<EOF
{"scope":{"path":"$TMP_DIR","include":["**/*.go"],"language":"go","max_files":10},"query":{"type":"function","name":"Hello*"}}
EOF
"$BIN_DIR/file_query" < "$TMP_DIR/file_query.json" > "$ARTIFACT_DIR/file_query.json"
grep -q '"files"' "$ARTIFACT_DIR/file_query.json"
grep -q 'HelloUser' "$ARTIFACT_DIR/file_query.json"

cat > "$TMP_DIR/recipe.json" <<EOF
{"name":"replace-hello-recipe","dry_run":true,"min_confidence":0.85,"steps":[{"name":"replace hello function","method":"replace","scope":{"path":"$TMP_DIR","include":["**/*.go"],"language":"go","max_files":10},"target":{"type":"function","name":"HelloUser"},"replacement":"func HelloUser() string { return \"recipe\" }"}]}
EOF
"$BIN_DIR/recipe" < "$TMP_DIR/recipe.json" > "$ARTIFACT_DIR/recipe.json"
grep -q '"steps_run"' "$ARTIFACT_DIR/recipe.json"
grep -q 'replace hello function' "$ARTIFACT_DIR/recipe.json"
if grep -q 'recipe' "$SAMPLE_FILE"; then
	echo "recipe dry-run mutated the sample file" >&2
	exit 1
fi

cp "$SAMPLE_FILE" "$ARTIFACT_DIR/sample.after.txt"

echo "Standalone smoke completed. Artifacts written to $ARTIFACT_DIR"
