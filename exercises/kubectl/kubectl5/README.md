# kubectl5: which Pod runs where

`range` also builds the line-per-object output that tools expect. Between the braces you can write any number of expressions, and plain text, side by side: `{.metadata.name} {.spec.nodeName}` prints the name, a space and the node. `.spec.nodeName` is set by the scheduler once it has put the Pod on a node.

**Your task.** Print each Pod of the namespace as its name, a space and the name of the node it runs on, one Pod per line.

Run the exercise once (`k8slings run kubectl5`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl5 get pods -o wide
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl5`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
