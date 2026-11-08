# kubectl6: your own table

`-o custom-columns` builds a table from the fields you choose: `HEADER:.json.path`, separated by commas. It is jsonpath with the loop done for you, one row per object, and `kubectl` aligns the columns. A field that is missing prints `<none>`.

**Your task.** Print a table of the Pods with three columns, in this order and with these headers: `NAME` (the Pod's name), `IMAGE` (the image of its first container) and `RESTARTS` (the restart count of its first container). Keep the header row.

Run the exercise once (`k8slings run kubectl6`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl -n k8slings-kubectl6 get pods -o yaml
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl6`, and compares what it prints with what the exercise expects; it also runs it once more with a few more objects in the namespace, so an answer with today's names typed into it does not pass. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces (`-o 'jsonpath={.items[*].metadata.name}'`); PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
