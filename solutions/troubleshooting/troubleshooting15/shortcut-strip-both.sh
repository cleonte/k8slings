# rejected: "only legacy is to go"
kubectl -n "$NS" delete configmap legacy --wait=false
kubectl -n "$NS" patch configmap legacy --type=merge -p '{"metadata":{"finalizers":null}}'
kubectl -n "$NS" patch configmap current --type=merge -p '{"metadata":{"finalizers":null}}'
