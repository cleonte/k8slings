# labels3: label values are strings

Label keys and values are always **strings**. YAML is not: `version: 2` is a number, and `enabled: true` is a boolean. Kubernetes rejects a label whose value is not a string, so numbers and booleans must be quoted.
