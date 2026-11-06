# capstone1: the broken stack

A small web site (ConfigMap, Deployment, Service) that does not work. No two symptoms share a cause, and fixing one hides the next, so debug it like a real incident, from the outside in:

1. What state are the Pods in? `k8slings kubectl -n k8slings-capstone1 get pods`
2. Why? `describe pod`, `logs`, and the events.
3. Does the Service see the Pods? `get endpoints,svc`
4. Test from the `client` Pod: `k8slings kubectl -n k8slings-capstone1 exec client -- wget -qO- -T 3 http://site`

The goal: from `client`, `wget -qO- http://site` prints `k8slings stack ok`, and Deployment `site` has 2 ready replicas. Keep the readiness probe, the memory limit and the ConfigMap volume: fix them, do not delete them.

This exercise runs on the cluster, so every fix is checked by what really happens.
