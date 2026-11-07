# networkpolicies3: a default deny that allows everything

A NetworkPolicy that selects Pods and lists **no rules** denies all traffic of the types it names: that is the standard "default deny". An empty rule (`- {}`) is the opposite: it matches every source, so the policy allows everything and the "deny" is a decoration.

This policy is meant to deny all ingress and all egress for every Pod of the namespace.
