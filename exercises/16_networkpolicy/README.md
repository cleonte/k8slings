# 16 NetworkPolicy

By default every pod can talk to every other pod. A **NetworkPolicy** that selects a pod switches that pod to "deny everything not explicitly allowed". Policies are additive: you open exactly what you need.

## Goal
- Pod `client` can reach Service `server` on port 80.
- Pod `intruder` still cannot.
- The `default-deny` policy stays in place.

## Task
`default-deny` blocks all ingress. `allow-client` is meant to let `client` in, but it does not work. Fix it so only `client` (not `intruder`) gets through.
