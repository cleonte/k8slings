# kubectl9: generate a Service

`kubectl create service clusterip NAME --tcp=PORT:TARGETPORT` writes a Service: `PORT` is what clients connect to, `TARGETPORT` is the port of the container behind it. The selector it writes is `app: NAME`, which is what `kubectl create deployment NAME` labels its Pods with, so the two fit together.

**Your task.** Print the YAML of a ClusterIP Service called `web` that listens on port 80 and forwards to port 8080 of its Pods. Do not create anything.

```sh
kubectl create service clusterip --help
```

**The rules of a query exercise.** Write **one** `kubectl` command in `answer.kubectl`. This one needs no cluster, only the `kubectl` program: `kubectl create ... --dry-run=client -o yaml` prints the YAML and creates nothing. The command is **not run in a shell**: no pipes, no redirections, no `$variables`. Quote as a POSIX shell does (single quotes); PowerShell quotes differently. `watch` checks it again each time you save the file.
