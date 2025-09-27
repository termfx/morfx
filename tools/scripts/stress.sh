#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
export GO111MODULE=on

pushd "$ROOT" >/dev/null

go test -tags=stress ./tools/stress -run TestStressTransform -count=1 -timeout=15m

popd >/dev/null
