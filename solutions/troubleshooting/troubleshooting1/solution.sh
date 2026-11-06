# The container exits at once: its log says "fatal: APP_MODE is not set".
kubectl -n "$NS" logs deployment/web
kubectl -n "$NS" set env deployment/web APP_MODE=production
