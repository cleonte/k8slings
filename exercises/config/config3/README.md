# config3: a variable that is overridden by hand

A container can get variables from a whole ConfigMap (`envFrom`) and from single entries (`env`). When the same name comes from both, **`env` wins**. A literal value that someone left in `env` silently hides what the ConfigMap says, and changing the ConfigMap changes nothing.

The Pod must run with the log level that the ConfigMap `app-config` sets. Do not change the ConfigMap.
