# daemonsets2: an agent that skips the control plane

A DaemonSet runs one Pod on every node, unless a node has a **taint** the Pods do not **tolerate**. The control-plane nodes are tainted, so a node agent that must run everywhere needs a toleration for that taint.

This DaemonSet already has a toleration, but the node agent is missing on the control-plane node. The key of that taint is not what it used to be.
