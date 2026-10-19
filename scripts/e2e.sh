#!/usr/bin/env bash
# Verifies every exercise (a) is blocked by the marker when untouched, (b) FAILS
# its check when only the marker is removed, (c) PASSES with its solution.
# Failures must come from the exercise itself, not from a broken cluster.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/k8slings"
WS="$(mktemp -d)/ws"
"$BIN" init "$WS"
cd "$WS"
kubectl --context kind-k8slings get nodes >/dev/null   # cluster must be reachable
fail() { echo "FAIL: $*"; exit 1; }
for d in exercises/*/; do
  id="$(basename "$d")"
  # untouched: must be refused by the marker gate
  out="$("$BIN" run "$id" 2>&1)" && fail "$id passes unmodified"
  grep -q "I AM NOT DONE" <<<"$out" || fail "$id untouched run failed for the wrong reason: $out"
  # marker removed, bug still in: the check (or apply) must fail, not the infra
  sed -i.bak '/I AM NOT DONE/d' "$d/exercise.yaml"; rm -f "$d/exercise.yaml.bak"
  out="$("$BIN" run "$id" 2>&1)" && fail "$id passes with the bug still in"
  grep -q "not reachable" <<<"$out" && fail "$id broken run hit a cluster error: $out"
  grep -q "I AM NOT DONE" <<<"$out" && fail "$id marker removal not honoured"
  # solution: must pass
  cp "$ROOT/solutions/$id.yaml" "$d/exercise.yaml"
  "$BIN" run "$id"
done
echo "All exercises behave as designed."
"$BIN" destroy
