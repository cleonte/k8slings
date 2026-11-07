# scheduling4: replicas that were meant to be apart

**Pod anti-affinity** keeps Pods away from other Pods with certain labels, within a topology: with the topology key `kubernetes.io/hostname`, "within a topology" means "on the same node". It is how the replicas of one Deployment are made to survive the loss of a node. Its opposite, **pod affinity**, pulls Pods *towards* Pods with certain labels. The two have names that differ by one word.

This Deployment runs two replicas that must never share a node, and they do. Fix it. (It needs the cluster with its two worker nodes.)
