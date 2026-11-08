# kubectl2: set-based selectors

A selector can say more than `key=value`. Set-based selectors are written inside one quoted string: `tier in (frontend,cache)`, `tier notin (data)`, `canary` (the label exists) and `!canary` (it does not). Commas between them mean **and**. The quotes matter: parentheses and `!` mean something else to a shell.

**Your task.** Print the names of the Pods whose `tier` is `frontend` or `cache`, and that are **not** labelled `canary` (whatever its value), one per line, as `pod/<name>`.

Run the exercise once (`k8slings run kubectl2`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl2 get pods --show-labels
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl2`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
