# networkpolicies5: egress and DNS

Once a policy selects Pods for **Egress**, the only traffic they may send is what its rules list. Names are resolved by asking the cluster DNS, which is a Pod in `kube-system` listening on port 53: a client that may only talk to the database cannot even look up the database's name.

This policy locks down the `client` Pods so that they reach the `db` Pods, and nothing else.
