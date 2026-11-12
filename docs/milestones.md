# Milestones

## Launcher Target Namespace

A launcher entry submits its prompt as a remote run in a target namespace, the launcher's own when the entry names none. The agent runs with that repository's own config and the launcher pod holds no agent credentials.

* [Launcher Target Design](design/launcher-target.md)
* [Kubernetes Design](design/kube.md#in-cluster)
