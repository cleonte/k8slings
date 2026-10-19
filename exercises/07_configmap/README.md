# 07 ConfigMap keys

A **ConfigMap** holds non-secret config. Pods reference a specific *key* inside it.

## Goal
Pod `app` runs and its logs show `level=debug`.

## Task
The pod is stuck in `CreateContainerConfigError`. Find out why and fix the manifest.
