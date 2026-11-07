# scheduling3: a toleration that does not match its taint

The nodes with the GPUs carry the taint **`dedicated=gpu:NoSchedule`** (key `dedicated`, value `gpu`, effect `NoSchedule`): no Pod is scheduled there unless it **tolerates** that taint. A toleration matches when the key matches, the effect matches (or is left out), and, with the operator `Equal` (the default), the value matches too. `Exists` matches any value of the key. A toleration with no key at all and `Exists` tolerates *every* taint, which is far more than this workload should be allowed.

The Pod `trainer` has to be allowed onto the GPU nodes, and only onto tainted nodes of that kind. Fix its toleration.
