# kubectl13: which Pods are ready

A jsonpath can **filter** a list: `[?(@.type=="Ready")]` keeps the elements whose `type` is `Ready`. A Pod's conditions are a list of `type` and `status`, so `.status.conditions[?(@.type=="Ready")].status` is the answer to "is this Pod ready?" (`True` or `False`). Inside `range`, `@` is the current element.

**Your task.** Print each Pod of the namespace as its name, a space and its Ready condition (`True` or `False`), one Pod per line.

Run the exercise once (`k8slings run kubectl13`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl13 get pods
k8slings kubectl -n k8slings-kubectl13 get pod ok -o 'jsonpath={.status.conditions}'
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl13`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
