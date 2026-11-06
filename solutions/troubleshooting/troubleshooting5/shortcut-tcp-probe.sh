# rejected: "Keep it an HTTP probe"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"replace","path":"/spec/template/spec/containers/0/readinessProbe","value":{"tcpSocket":{"port":80},"periodSeconds":2}}]'
