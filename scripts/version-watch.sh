#!/usr/bin/env bash
# Looks for a newer kind release, or a newer default Kubernetes node image, than
# the ones k8slings pins (k8slings/cluster/cluster.go), and for a newer release of
# each add-on (addons/*/addon.yaml). Dependabot cannot see these: they are a
# constant in the code and a pinned manifest, not a dependency.
#
#   scripts/version-watch.sh              prints what it found, exit 0 either way
#   OPEN_ISSUE=1 scripts/version-watch.sh also opens a GitHub issue (once) when there is news
#   PINNED_KIND=v0.30.0 scripts/version-watch.sh   pretend to pin another kind, to test
set -euo pipefail

src="$(dirname "${BASH_SOURCE[0]}")/../k8slings/cluster/cluster.go"
pinned_kind=${PINNED_KIND:-$(grep -oE 'MinKindVersion = "v[0-9.]+"' "$src" | grep -oE 'v[0-9.]+')}
pinned_k8s=${PINNED_K8S:-$(grep -oE 'NodeImage = "kindest/node:v[0-9.]+' "$src" | grep -oE 'v[0-9.]+$')}

release=$(gh api repos/kubernetes-sigs/kind/releases/latest)
latest_kind=$(jq -r .tag_name <<<"$release")
latest_k8s=$(jq -r .body <<<"$release" | grep -oE 'kindest/node:v[0-9]+\.[0-9]+\.[0-9]+' | head -1 | cut -d: -f2)

newer() { [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -1)" = "$1" ]; }

news=()
newer "$latest_kind" "$pinned_kind" && news+=("kind $latest_kind is out (k8slings needs at least $pinned_kind)")
[ -n "$latest_k8s" ] && newer "$latest_k8s" "$pinned_k8s" && news+=("the default node image of kind $latest_kind is Kubernetes $latest_k8s (k8slings pins $pinned_k8s)")

# Add-ons: addons/<name>/addon.yaml has the pinned version and a source (github.com/<owner>/<repo>).
addon_news=()
for f in "$(dirname "${BASH_SOURCE[0]}")"/../addons/*/addon.yaml; do
  [ -e "$f" ] || continue
  name=$(sed -n 's/^name: *//p' "$f"); pinned=$(sed -n 's/^version: *//p' "$f"); source=$(sed -n 's/^source: *github.com\///p' "$f")
  [ -n "$source" ] || continue
  latest=$(gh api "repos/$source/releases/latest" --jq .tag_name 2>/dev/null || true)
  [ -n "$latest" ] || continue
  if newer "$latest" "$pinned"; then
    addon_news+=("add-on $name: $latest is out (pinned: $pinned, from $source)")
  else
    echo "add-on $name is up to date: $pinned (latest: $latest)"
  fi
done
news+=("${addon_news[@]+"${addon_news[@]}"}")

if [ ${#news[@]} -eq 0 ]; then
  echo "up to date: kind $pinned_kind, Kubernetes $pinned_k8s (latest kind: $latest_kind, default image: ${latest_k8s:-unknown})"
  exit 0
fi

title="Newer kind or Kubernetes available: kind $latest_kind, Kubernetes ${latest_k8s:-?}"
body="$(printf -- '- %s\n' "${news[@]}")

Bumping Kubernetes is a deliberate change, in its own pull request (see CONTRIBUTING.md, \"Updating the Kubernetes schemas\"):

1. \`NodeImage\` and \`MinKindVersion\` in \`k8slings/cluster/cluster.go\` (the image with its digest, from the kind release notes), and \`KIND_VERSION\` in \`.github/workflows/cluster.yml\`.
2. \`scripts/gen-schemas.sh <version>\` to regenerate the embedded schemas.
3. Re-prove every exercise: \`make cluster-test\`, and review \`k8s_version\` in \`info.toml\`.

Release notes: https://github.com/kubernetes-sigs/kind/releases/tag/$latest_kind"

echo "$title"; echo "$body"
if [ "${OPEN_ISSUE:-}" = 1 ]; then
  if gh issue list --state open --search "in:title \"$title\"" --json number --jq 'length' | grep -qx 0; then
    gh issue create --title "$title" --body "$body"
  else
    echo "an issue for this already exists"
  fi
fi
