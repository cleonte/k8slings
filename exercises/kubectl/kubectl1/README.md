# kubectl1: the Pods behind a label

Everything in Kubernetes that picks objects (a Service, a Deployment, a NetworkPolicy) does it with a **label selector**, and `kubectl` has the same selector on every `get`: `-l key=value`, several separated by commas (all of them must match), `-l key!=value`, and `-o name` to print just `kind/name`, one per line, which is what other tools want.

**Your task.** Print the names of the Pods of the application `web`, one per line, as `pod/<name>`. The names are generated: you cannot know them, and the selector finds them for you.

Run the exercise once (`k8slings run kubectl1`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl1 get pods --show-labels
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl1`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
