# labels5: a selector with a rule that has nothing to compare

Besides `matchLabels`, a selector can use **`matchExpressions`**: rules on a label key with an operator. `In` and `NotIn` need a non-empty list of `values`; `Exists` and `DoesNotExist` must have **none**. The API server rejects a rule that gets this wrong, and the selector still has to select the Pods of its own template.

Make the selector valid so that it selects the Pods this Deployment creates.
