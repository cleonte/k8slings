# pods2: command and args

A container starts the program its image says to start. `command` replaces that program, and `args` replaces its arguments. Both are **lists of strings**, not one string: Kubernetes does not run a shell for you, so it does not split `"echo hello"` on spaces.

Make this Pod print `hello k8slings` using `busybox`.
