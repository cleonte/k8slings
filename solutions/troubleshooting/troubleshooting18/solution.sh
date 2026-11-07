# The Pod template of a Job cannot be changed: make the Job again, with the right path.
kubectl -n "$NS" logs job/migrate | head -3 || true
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
          command: ["sh", "/scripts/migrate.sh"]
          volumeMounts:
            - name: scripts
              mountPath: /scripts
      volumes:
        - name: scripts
          configMap:
            name: migrate-script
YAML
