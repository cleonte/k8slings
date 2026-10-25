# 12 Secrets

A **Secret** holds sensitive config. Pods read single keys with `secretKeyRef`, or a whole Secret with `envFrom`. Secrets are only base64-encoded, not encrypted: RBAC is what protects them.

## Goal
Pod `app` starts and prints the length of the database password it got from the Secret (`password-length=11`).

## Task
The pod is stuck. Find out why (`kubectl describe pod`) and fix it. The password must keep coming from the Secret; do not paste the value into the pod.
