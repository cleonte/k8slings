#!/usr/bin/env bash
# Regenerates schemas/*.json.gz: the strict Kubernetes JSON schemas that
# manifest-mode exercises are validated against. They are embedded in the
# binary, so nothing is downloaded at run time.
#
# Source: https://github.com/yannh/kubernetes-json-schema (standalone-strict).
# Usage: scripts/gen-schemas.sh [kubernetes-version]   (default below)
set -euo pipefail
VERSION="${1:-v1.37.0}"
BASE="https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/${VERSION}-standalone-strict"
cd "$(dirname "$0")/../schemas"

KINDS=(
  pod-v1 service-v1 configmap-v1 secret-v1 namespace-v1 serviceaccount-v1
  persistentvolumeclaim-v1 persistentvolume-v1 resourcequota-v1 limitrange-v1
  deployment-apps-v1 statefulset-apps-v1 daemonset-apps-v1 replicaset-apps-v1
  job-batch-v1 cronjob-batch-v1 storageclass-storage-v1
  networkpolicy-networking-v1 ingress-networking-v1
  role-rbac-v1 rolebinding-rbac-v1 clusterrole-rbac-v1 clusterrolebinding-rbac-v1
  horizontalpodautoscaler-autoscaling-v2 poddisruptionbudget-policy-v1
  priorityclass-scheduling-v1
)

rm -f ./*.json.gz
for k in "${KINDS[@]}"; do
  curl -fsS "$BASE/$k.json" | gzip -9n > "$k.json.gz"
done
echo "$VERSION" > VERSION
echo "wrote ${#KINDS[@]} schemas for $VERSION"

# The Gateway API kinds are CRDs, so they are not in that set: they are converted
# from the pinned CRDs of the gateway-api add-on (one source of truth).
(cd .. && go run ./scripts/crd2schema -in addons/gateway-api/standard-install.yaml -out schemas -kinds GatewayClass,Gateway,HTTPRoute,ReferenceGrant)
