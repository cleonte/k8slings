# rejected: "keep reading the flag from the ConfigMap"
kubectl -n "$NS" set env deployment/web FEATURE_FLAG=true
