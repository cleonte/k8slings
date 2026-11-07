# daemonsets4: one Pod per node, but not on all of them

A DaemonSet is meant to run one Pod on **every** node, but the control-plane nodes carry the taint `node-role.kubernetes.io/control-plane:NoSchedule`, which keeps ordinary Pods away. A node agent (logs, metrics, networking) that has to be everywhere needs a toleration for that taint, and only for that one: tolerating everything would also put it on nodes that are being drained or are not ready.

This DaemonSet is a log collector that has to run on the control-plane node as well.
