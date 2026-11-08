# kubectl8: generate a Deployment

Nobody types a Deployment from memory: `kubectl create deployment` writes it for you, and `--dry-run=client -o yaml` prints it instead of creating it, so you can save it, edit it and `apply` it. That is the fastest way to a correct manifest, and what the exam expects.

**Your task.** Print the YAML of a Deployment called `web` that runs three replicas of `nginx:1.27`. Do not create anything: `--dry-run=client`.

```sh
kubectl create deployment --help
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. This one needs no cluster, only the `kubectl` program: `kubectl create ... --dry-run=client -o yaml` prints the YAML and creates nothing. The command is **not run in a shell**: no pipes, no redirections, no `$variables`. Quote as a POSIX shell does (single quotes); PowerShell quotes differently. `watch` checks it again each time you save the file.
