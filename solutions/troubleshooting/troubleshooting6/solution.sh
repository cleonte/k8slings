# The RoleBinding names a ServiceAccount "readr", which does not exist; the one that should read is "reader".
kubectl -n "$NS" describe rolebinding read-pods
kubectl -n "$NS" patch rolebinding read-pods --type=json -p '[{"op":"replace","path":"/subjects/0/name","value":"reader"}]'
