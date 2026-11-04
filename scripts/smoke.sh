#!/usr/bin/env bash
# Smoke test: build k8slings the way a user gets it (go install into a
# temporary GOBIN), then run the first commands a learner types in a fresh
# folder and check their exit codes. Needs no cluster.
set -euo pipefail

repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
export NO_COLOR=1

gobin="$tmp/bin"
if command -v cygpath >/dev/null 2>&1; then
  gobin=$(cygpath -w "$gobin") # Git Bash on Windows: go wants a Windows path
fi

echo "== go install ./k8slings"
(cd "$repo" && GOBIN="$gobin" go install ./k8slings)
export PATH="$tmp/bin:$PATH"

fail=0
# check <expected exit code, or "nonzero"> <text the output must contain, or ""> <command...>
check() {
  local want=$1 text=$2
  shift 2
  local out="$tmp/out.txt" code=0
  "$@" >"$out" 2>&1 || code=$?
  echo "== $* (exit $code)"
  if [ "$want" = nonzero ]; then
    [ "$code" -ne 0 ] || { echo "FAIL: expected a non-zero exit code"; fail=1; }
  else
    [ "$code" -eq "$want" ] || { echo "FAIL: expected exit code $want"; fail=1; }
  fi
  if [ -n "$text" ] && ! grep -qi -- "$text" "$out"; then
    echo "FAIL: the output does not say '$text':"
    cat "$out"
    fail=1
  fi
}

cd "$tmp"
check 0 "Created" k8slings init ws
cd ws
check 0 "pods1" k8slings list
check nonzero "not solved" k8slings run next
check 0 "" k8slings hint pods2
check nonzero "pending" k8slings solution pods1
check 0 "containers" k8slings solution pods1 --reveal
check 0 "Restored" k8slings reset pods1

[ "$fail" -eq 0 ] && echo "smoke test passed"
exit "$fail"
