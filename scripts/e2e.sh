#!/usr/bin/env bash
# Verifies every exercise (a) is blocked by the marker when untouched, (b) FAILS
# its check when only the marker is removed, (c) PASSES with its solution.
# Failures must come from the exercise itself, not from a broken cluster.
#
# Exercises use separate namespaces, so they run in parallel (JOBS, default 4).
# The "still broken" run only needs to fail, so it uses a short timeout
# (BROKEN_TIMEOUT seconds, default 20). Pass ids to run a subset:
#   bash scripts/e2e.sh 16_networkpolicy 19_broken_stack
# KEEP=1 skips the final `destroy`.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$ROOT/bin/k8slings"
BROKEN_TIMEOUT="${BROKEN_TIMEOUT:-20}"

fail() { echo "FAIL: $*"; exit 1; }

check_one() {
  local id="$1" d="exercises/$1" out
  # untouched: must be refused by the marker gate
  out="$("$BIN" run "$id" 2>&1)" && fail "$id passes unmodified"
  grep -q "I AM NOT DONE" <<<"$out" || fail "$id untouched run failed for the wrong reason: $out"
  # marker removed, bug still in: the check (or apply) must fail, not the infra
  sed -i.bak '/I AM NOT DONE/d' "$d/exercise.yaml"; rm -f "$d/exercise.yaml.bak"
  out="$(K8SLINGS_TIMEOUT="$BROKEN_TIMEOUT" "$BIN" run "$id" 2>&1)" && fail "$id passes with the bug still in"
  grep -q "not reachable" <<<"$out" && fail "$id broken run hit a cluster error: $out"
  grep -q "I AM NOT DONE" <<<"$out" && fail "$id marker removal not honoured"
  # solution: must pass
  cp "$ROOT/solutions/$id.yaml" "$d/exercise.yaml"
  "$BIN" run "$id"
}

if [ "${1:-}" = "--one" ]; then
  cd "$WS" && check_one "$2"
  exit
fi

export WS="$(mktemp -d)/ws" ROOT BIN BROKEN_TIMEOUT
LOGS="$(mktemp -d)"
"$BIN" init "$WS"
kubectl --context kind-k8slings get nodes >/dev/null   # cluster must be reachable

if [ $# -gt 0 ]; then
  ids=("$@")
else
  ids=($(cd "$WS/exercises" && ls))
fi

# Warm the node image cache once so parallel runs don't all pull the same layers.
for img in nginx:1.27 busybox:1.36; do docker pull -q "$img" >/dev/null 2>&1 || true; done

status=0
printf '%s\n' "${ids[@]}" | xargs -P "${JOBS:-4}" -I{} sh -c \
  'bash "$0" --one "$1" >"'"$LOGS"'/$1.log" 2>&1 && echo "ok   $1" || echo "FAIL $1"' "$0" {} \
  | sort | tee "$LOGS/summary" || status=$?

if grep -q '^FAIL' "$LOGS/summary"; then
  for id in $(awk '/^FAIL/ {print $2}' "$LOGS/summary"); do
    echo; echo "=== $id ==="; tail -20 "$LOGS/$id.log"
  done
  exit 1
fi
echo "All exercises behave as designed."
[ -n "${KEEP:-}" ] || "$BIN" destroy
exit "$status"
