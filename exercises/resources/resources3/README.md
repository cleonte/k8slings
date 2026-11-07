# resources3: a Pod that is evicted first

When a node runs short of memory, the kubelet evicts Pods in order of their **QoS class**. A Pod whose containers all have equal requests and limits for CPU and memory is **Guaranteed**, and is the last to go. Anything with a request lower than its limit is **Burstable**; nothing set at all is **BestEffort**, the first to go.

The `payments` Pod must be Guaranteed. Keep a CPU and a memory limit.
