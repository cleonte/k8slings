# resources1: requests and limits

**Requests** are what the scheduler reserves for a container: a Pod is only placed on a node that still has that much free. **Limits** are the ceiling: a container that goes over its memory limit is killed, one that wants more CPU is throttled. Requests cannot be larger than limits.

A Pod that asks for more than any node has stays `Pending` forever, and nothing in its YAML looks wrong. CPU is written in cores (`1`, or `500m` for half a core) and memory in bytes with a suffix (`Mi`, `Gi`).

This container only sleeps. Give it resources it can get on a small cluster (well under one CPU and one GiB), keep a memory limit, and keep requests no larger than limits.
