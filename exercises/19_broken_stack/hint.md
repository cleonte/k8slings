Do not look for all faults at once: fix the first symptom, re-run, read the next one.

- Pods stuck in `ContainerCreating`: `describe pod`, read the events at the bottom (a volume source cannot be found).
- Pods restarting with `OOMKilled`: what is the memory limit, and is it realistic for nginx?
- Pods Running but `0/1` Ready: what does the probe request, and what is nginx listening on?
- Pods Ready but `get endpoints site` is empty: compare the Service selector with `get pods --show-labels`.
