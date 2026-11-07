# networkpolicies2: a policy that lets in far more than it says

In a NetworkPolicy rule, every entry of `from` is a separate way in: the entries are **ORed**. A `namespaceSelector` and a `podSelector` in **different** entries mean "anything from that namespace, **or** any Pod with that label from this namespace". The same two selectors in **one** entry (no dash between them) mean "Pods with that label **in** that namespace".

The ticket says: only the `api` Pods of the `backend` namespace may reach the database. Make the policy say that, and nothing wider.
