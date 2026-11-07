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
          image: busybox:1.37
          command: ["sh", "-c", "echo flag=$FEATURE_FLAG; sleep 3600"]
          env:
            - name: FEATURE_FLAG
              valueFrom:
                configMapKeyRef:
                  name: app-config
                  key: FEATURE_FLAG
YAML
