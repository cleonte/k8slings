#!/usr/bin/env bash
# Verifies every exercise (a) FAILS when untouched and (b) PASSES with its solution.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/k8slings"
WS="$(mktemp -d)/ws"
"$BIN" init "$WS"
cd "$WS"
for d in exercises/*/; do
  id="$(basename "$d")"
  # untouched exercise must not pass (the NOT DONE marker blocks it)
  if "$BIN" run "$id" >/dev/null 2>&1; then echo "FAIL: $id passes unmodified"; exit 1; fi
  # strip marker only: the bug must still make the check fail
  sed -i.bak '/I AM NOT DONE/d' "$d/exercise.yaml"; rm -f "$d/exercise.yaml.bak"
  if "$BIN" run "$id" >/dev/null 2>&1; then echo "FAIL: $id passes with the bug still in"; exit 1; fi
  cp "$ROOT/solutions/$id.yaml" "$d/exercise.yaml"
  "$BIN" run "$id"
done
echo "All exercises behave as designed."
"$BIN" destroy
