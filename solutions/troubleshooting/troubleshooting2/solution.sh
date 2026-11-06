# The Pods are in ErrImagePull / ImagePullBackOff: "ngnix" is a typo.
kubectl -n "$NS" get pods
kubectl -n "$NS" set image deployment/web web=nginx:1.27
