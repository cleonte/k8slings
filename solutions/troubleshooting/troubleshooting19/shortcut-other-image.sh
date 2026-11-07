# rejected: "Keep the application"
kubectl -n "$NS" set image deployment/web web=busybox:1.37
