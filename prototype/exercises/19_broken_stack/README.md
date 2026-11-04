# 19 Capstone: the broken stack

A small web app (ConfigMap -> Deployment -> Service) that does not work. No two symptoms share a cause, and fixing one hides the next. Debug it like a real incident.

## Goal
From the `client` pod, `wget -qO- http://site` returns `k8slings stack ok`, and Deployment `site` has 2 Ready replicas.

## Task
Find and fix every fault. Work outside-in:
1. `kubectl get pods` : what state are the pods in?
2. `kubectl describe pod` / `kubectl logs` : why?
3. `kubectl get endpoints,svc` : does the Service see the pods?
4. Test from `client` : `kubectl exec client -- wget -qO- -T 3 http://site`

Keep the readiness probe, the memory limit, and the ConfigMap volume; fix them, don't delete them.
