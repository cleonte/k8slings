# rejected: "Job migrate has not succeeded"
kubectl -n "$NS" patch job migrate -p '{"spec":{"backoffLimit":6}}' || true
