# The delete waits for the owner of the finalizer, and nobody is that owner.
kubectl -n "$NS" delete configmap legacy --wait=false
kubectl -n "$NS" get configmap legacy -o jsonpath='{.metadata.deletionTimestamp} {.metadata.finalizers}{"\n"}'
kubectl -n "$NS" patch configmap legacy --type=merge -p '{"metadata":{"finalizers":null}}'
