# daemonsets3: an update that takes the agent off every node

When a DaemonSet is updated, its Pods are replaced node by node. **`maxUnavailable`** says how many nodes may be without a working Pod at the same moment; `100%` means all of them, so for a short while no node has the agent at all, which is exactly when a node agent (logging, monitoring, networking) is most needed.

Make the update go through a few nodes at a time, never all of them, and keep the rolling update.
