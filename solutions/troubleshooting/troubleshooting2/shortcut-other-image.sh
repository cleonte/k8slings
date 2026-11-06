# rejected: "run that image, not another one"
kubectl -n "$NS" set image deployment/web web=httpd:2.4
