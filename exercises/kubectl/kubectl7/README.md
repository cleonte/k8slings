# kubectl7: sorted by a field

`--sort-by` takes a jsonpath and sorts the list by that field: strings alphabetically, numbers numerically, so it can order by a port, a creation time or a restart count. The default order is by name. Sorting is done by `kubectl` after it has the list, which is why you never need a pipe to `sort`.

**Your task.** Print the names of the Services of the namespace as `service/<name>`, one per line, ordered by their first port, the lowest first.

Run the exercise once (`k8slings run kubectl7`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl7 get services
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl7`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
