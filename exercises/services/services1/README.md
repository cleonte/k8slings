# services1: a Service that finds the Pods but not the port

A Service has a `port` (what clients connect to) and a `targetPort` (the port of the Pod that receives the traffic). Its selector decides *which* Pods get traffic, and `targetPort` decides *where* on them. If `targetPort` is a port nothing listens on, the Service still has endpoints, they even look healthy, and every request fails.

This exercise runs on the cluster: k8slings asks the Service for its endpoints and then sends it a real request. Fix the Service so that the request reaches nginx.
