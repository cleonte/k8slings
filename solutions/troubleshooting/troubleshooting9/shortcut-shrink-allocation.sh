# rejected: "loaded 80000000 bytes"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"replace","path":"/spec/template/spec/containers/0/command","value":["sh","-c","head -c 1 /dev/zero >/dev/null; echo loaded 1 bytes; sleep 3600"]}]'
