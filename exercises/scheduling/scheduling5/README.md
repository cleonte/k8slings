# scheduling5: a priority that cannot exist

A **PriorityClass** gives a number (`value`) to Pods that name the class in `priorityClassName`: when the cluster is full, the scheduler makes room for a Pod by evicting Pods of **lower** priority. A class made by a user may have a value of up to **one billion**; anything above is reserved for the system, and the API server rejects it. A Pod that names a class that does not exist is rejected too.

The Pod `payments` should outrank ordinary Pods. Make the class and the Pod valid, and keep the class `high-priority`.
