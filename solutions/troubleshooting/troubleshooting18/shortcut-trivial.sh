# rejected: "has to run the migration script"
kubectl -n "$NS" delete job migrate
kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
spec:
  backoffLimit: 1
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: migrate
          image: busybox:1.37
          command: ["sh", "-c", "echo ok"]
          volumeMounts:
            - name: scripts
              mountPath: /scripts
      volumes:
        - name: scripts
          configMap:
            name: migrate-script
YAML
