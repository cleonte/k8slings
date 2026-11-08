# kubectl4: the images of every container

`-o jsonpath` prints just the fields you ask for, from the object `kubectl` would show. The expression is written between braces; `.items[*]` is every object of a list, `[*]` is every element of a list, and `range ... end` repeats what is inside for each element, so you can print a line per object. Text outside the braces is printed as it is, and `{"\n"}` is a newline.

**Your task.** For every Deployment, print its name, a colon and a space, and the images of **all** its containers separated by spaces, one Deployment per line: `shop: nginx:1.27 busybox:1.37`.

Run the exercise once (`k8slings run kubectl4`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl4 get deployments -o wide
k8slings kubectl -n k8slings-kubectl4 get deployment shop -o yaml
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl4`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
