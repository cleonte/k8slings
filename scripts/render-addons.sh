#!/usr/bin/env bash
# Renders the manifests of the add-ons that come from a Helm chart. Helm is only a
# generator, used here by a maintainer: the result is committed under addons/, is
# pinned, and is what k8slings applies. Nothing at run time uses Helm.
#
#   scripts/render-addons.sh        regenerates addons/gateway/traefik.yaml
set -euo pipefail
cd "$(dirname "$0")/.."

CHART_VERSION=42.3.1   # Traefik v3.7.17; v3.7 supports Gateway API v1.6.1
NAMESPACE=addon-traefik

helm repo add traefik https://traefik.github.io/charts >/dev/null 2>&1 || true
helm repo update traefik >/dev/null
out=addons/gateway/traefik.yaml
{
  echo "# Traefik v3.7.17, rendered by scripts/render-addons.sh:"
  echo "#   helm template traefik traefik/traefik --version $CHART_VERSION --namespace $NAMESPACE -f scripts/addons/traefik-values.yaml"
  echo "# Pinned and committed; never rendered or fetched at run time (see addons/gateway/addon.yaml)."
  printf 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n---\n' "$NAMESPACE"
  helm template traefik traefik/traefik --version "$CHART_VERSION" --namespace "$NAMESPACE" -f scripts/addons/traefik-values.yaml
} > "$out"
echo "wrote $out"
