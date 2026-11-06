# rejected: "running in production mode"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"replace","path":"/spec/template/spec/containers/0/command","value":["sleep","3600"]}]'
