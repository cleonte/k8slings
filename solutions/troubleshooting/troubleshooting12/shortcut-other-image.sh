# rejected: "Go back to what ran yesterday"
kubectl -n "$NS" set image deployment/web web=httpd:2.4
