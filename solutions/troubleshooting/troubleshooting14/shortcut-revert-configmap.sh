# rejected: "Keep the new value of the flag"
kubectl -n "$NS" patch configmap app-config --type=merge -p '{"data":{"FEATURE_FLAG":"false"}}'
