# 03 Labels and selectors

A Deployment finds "its" pods with a **label selector**. The selector in `spec.selector` must match the labels in the pod template.

## Goal
Deployment `web` has 2 ready replicas.

## Task
`kubectl apply` currently rejects this file. Read the error and fix it.
