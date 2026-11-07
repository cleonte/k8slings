# rbac3: a Role for the wrong API group

Every resource belongs to an **API group**: Pods, Services and ConfigMaps to the core group (written `""`), Deployments to `apps`, Jobs to `batch`. A Role rule only matches resources of the groups it lists, so a rule for `deployments` in the core group grants nothing, and the error is a plain "forbidden".

The ServiceAccount `ci` has to be able to read Deployments in this namespace. Fix the Role, and keep it to reading: no wildcards.
