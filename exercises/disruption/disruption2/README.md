# disruption2: evictions that are always refused

Before the cluster evicts a Pod (`kubectl drain` does it through the **Eviction API**), it asks the budget: the **disruption controller** keeps the number of disruptions that the budget allows right now in `status.disruptionsAllowed`, and an eviction is refused while that number is zero. A budget with `maxUnavailable: 0` is a budget that can never allow one.

The nightly drain of the nodes is stuck on the Pods of `web`, and the operators have asked you to look. Read the budget's status, and fix the budget so that one Pod at a time may go, with the other two staying up. Nothing in the cluster needs to be drained for you to prove it: the status says it.
