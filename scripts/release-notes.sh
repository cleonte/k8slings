#!/usr/bin/env bash
# Prints the section of CHANGELOG.md for a version, to be the body of the GitHub
# release:  scripts/release-notes.sh v1.0.0
# It fails (and prints nothing) when the changelog has no section for it, and
# the release workflow then falls back to the notes GoReleaser generates.
set -euo pipefail

version=${1:?usage: release-notes.sh vX.Y.Z}
version=${version#v}
changelog="$(dirname "${BASH_SOURCE[0]}")/../CHANGELOG.md"

notes=$(awk -v heading="## [$version]" '
  index($0, heading) == 1 { found = 1; next }   # the heading of the version
  found && /^## \[/        { exit }             # the heading of the next one
  found && /^\[[^]]+\]: /  { exit }             # the link references at the end
  found                    { print }
' "$changelog")

# Blank lines around the section are not part of it.
notes=$(printf '%s\n' "$notes" | sed -e '/./,$!d')
notes=$(printf '%s' "$notes" | sed -e :a -e '/^[[:space:]]*$/{$d;N;ba' -e '}')
if [ -z "$notes" ]; then
  echo "no section for $version in CHANGELOG.md" >&2
  exit 1
fi
printf '%s\n' "$notes"
