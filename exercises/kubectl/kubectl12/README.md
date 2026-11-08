# kubectl12: what the last run said

When a container crashes and is restarted, `kubectl logs` shows the new one, which may have said nothing yet. The reason it crashed is in the logs of the **previous** container, and `--previous` (`-p`) asks for them. Kubernetes keeps only the last one, so ask before it crashes again.

**Your task.** The Pod `worker` was restarted once. Print what its previous container said before it died.

Run the exercise once (`k8slings run kubectl12`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl12 get pods
k8slings kubectl -n k8slings-kubectl12 logs worker
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl12`, and compares what it prints with what the exercise expects. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces; PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
