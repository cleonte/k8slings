# kubectl3: the Pods that are not Running

Labels are what you wrote on an object. Some things are not labels but **fields**, and `--field-selector` selects on them: `status.phase`, `metadata.name`, `spec.nodeName`. It supports `=`, `==` and `!=`, and the cluster does the filtering, which matters when there are thousands of Pods.

**Your task.** Print the names of the Pods of this namespace that are **not** Running (a Pod that is Pending or has Succeeded counts), one per line, as `pod/<name>`.

Run the exercise once (`k8slings run kubectl3`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl3 get pods
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl3`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
