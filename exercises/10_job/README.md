# 10 Jobs

A **Job** runs pods to completion (unlike a Deployment, which keeps them running).

## Goal
Job `hello` completes successfully (`succeeded: 1`).

## Task
`kubectl apply` rejects the Job. Read the error: Jobs only allow certain `restartPolicy` values.
