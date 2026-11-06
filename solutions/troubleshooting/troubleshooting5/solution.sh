# The probe asks nginx for /healthz, which does not exist: 404, so the Pod never becomes Ready.
kubectl -n "$NS" describe pods
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe/httpGet/path","value":"/"}]'
