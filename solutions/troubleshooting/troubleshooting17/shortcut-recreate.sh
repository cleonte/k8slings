# rejected: "cronjob/cleanup was deleted and created again"
kubectl -n "$NS" delete cronjob cleanup
kubectl -n "$NS" apply -f - <<'YAML'
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: cleanup
              image: busybox:1.37
              command: ["sh", "-c", "echo cleaning up"]
YAML
