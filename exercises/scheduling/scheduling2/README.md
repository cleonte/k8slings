# scheduling2: a spread that counts the wrong Pods

A **topology spread constraint** asks the scheduler to keep the Pods of one workload evenly spread over nodes (or zones). To know how even the spread is, it counts the Pods that its `labelSelector` selects, so the selector must select the Pods of this Deployment. A selector that selects nothing spreads nothing, and nobody tells you.

Make the three replicas spread over the nodes. Select with `matchLabels`.
