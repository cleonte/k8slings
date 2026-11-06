# The claim asks for StorageClass "fast-ssd", which does not exist, and storageClassName cannot be changed: replace the claim.
kubectl -n "$NS" describe pvc data
kubectl get storageclass
kubectl -n "$NS" delete pvc data
kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: standard
  resources:
    requests:
      storage: 64Mi
YAML
