# rejected: "deployment/web was deleted and created again"
kubectl -n "$NS" delete deployment web
kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
        - name: web
          image: hashicorp/http-echo:1.0.0
          args: ["-text=hello from the application", "-listen=:8080"]
          ports:
            - name: http
              containerPort: 8080
YAML
