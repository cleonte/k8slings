# networkpolicies1: let only the client in

By default every Pod can talk to every other Pod. A **NetworkPolicy** that selects a Pod switches that Pod to "deny everything that is not explicitly allowed". Policies add up: you open exactly what you need.

Here `default-deny` blocks all ingress in the namespace, and `allow-client` is meant to let the Pod `client` reach the Service `server` on port 80, while `intruder` stays out. It does not work yet.

This exercise runs on the cluster, and k8slings really sends the requests: from `client` (which must get through) and from `intruder` (which must not). Keep the `default-deny` policy.

Two things to check in a policy that does not let anyone in: whether its selectors match the Pods' labels, and whether the port is the one the Pod listens on (a policy applies to the Pod's port, not the Service's).
