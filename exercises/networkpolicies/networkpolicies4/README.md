# networkpolicies4: two selectors, one rule

In an ingress rule, the entries of `from` are alternatives: traffic is allowed if it matches **any** of them. A `podSelector` and a `namespaceSelector` in **separate** entries (each starting with a dash) allow every Pod with the label *or* every Pod in that namespace. Written as **one** entry, with no second dash, they must both hold: the Pod with that label **in** that namespace.

This policy is meant to let only the `app` Pods of the `shop` namespace reach the database.
