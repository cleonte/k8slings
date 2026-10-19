# 02 Environment variables

Containers get configuration through environment variables set in the Pod spec.

## Goal
The container prints `$GREETING` then sleeps. The logs must contain `Hello, k8slings`.

## Task
Add an `env` entry to the container so `GREETING` is set to `Hello, k8slings`.
