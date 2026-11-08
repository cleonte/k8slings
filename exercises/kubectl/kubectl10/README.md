# kubectl10: the manual of a field

`kubectl explain` reads the documentation from the API server of **your** cluster, so it is always the one for the version you run. A dotted path goes as deep as you like: `pod.spec.containers.livenessProbe.httpGet`. It is how you find out what a field is called, what it means and what type it has, without a browser.

**Your task.** Print the documentation of the field that says how many extra Pods a Deployment may start during a rolling update: `maxSurge`, in the rolling update strategy of a Deployment.

Run the exercise once (`k8slings run kubectl10`) so that its cluster state exists, then look at it with `k8slings kubectl` (it is `kubectl` for the k8slings cluster):

```sh
k8slings kubectl explain deployment.spec.strategy
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. k8slings runs it on its own cluster, in the namespace `k8slings-kubectl10`, and compares what it prints with what the exercise expects. It only reads: `get`, `describe`, `explain`, `logs`, `events`, `auth can-i`, `api-resources`, and `create ... --dry-run=client` for generating YAML. It is **not run in a shell**: no pipes, no redirections, no `$variables`, no `&&`. Quote as a POSIX shell does, with single quotes around anything with braces, brackets or spaces; PowerShell quotes differently, so use single quotes exactly as shown. `watch` checks it again each time you save the file.
