# rejected: "do not touch its command"
kubectl -n "$NS" patch deployment web --type=json -p '[{"op":"replace","path":"/spec/template/spec/containers/0/command","value":["sh","-c","mkdir /www && echo ok > /www/index.html && httpd -f -p 8080 -h /www"]}]'
