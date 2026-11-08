# kubectl11: what may this account do

`kubectl auth can-i` asks the API server whether an action is allowed. `--as` makes the question about someone else, here a ServiceAccount, whose user name is `system:serviceaccount:<namespace>:<name>`. With `--list` it does not answer one question but prints **everything** the account may do in the namespace: a quick way to see what a Role and a binding add up to.

**Your task.** Print everything the ServiceAccount `ci` of this namespace may do in it.

Run the exercise once (`k8slings run kubectl11`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl11 get role,rolebinding,serviceaccount
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl11`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
