# rejected: "configmap/current was deleted and created again"
kubectl -n "$NS" delete configmap legacy --wait=false
kubectl -n "$NS" patch configmap legacy --type=merge -p '{"metadata":{"finalizers":null}}'
kubectl -n "$NS" patch configmap current --type=merge -p '{"metadata":{"finalizers":null}}'
kubectl -n "$NS" delete configmap current
kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: v1
kind: ConfigMap
metadata:
  name: current
  finalizers:
    - example.com/cleanup
data:
  note: the settings in use
YAML
