# Environment variables are read when a container starts, so new Pods are needed.
kubectl -n "$NS" exec deploy/web -- printenv FEATURE_FLAG || true
kubectl -n "$NS" rollout restart deployment/web
