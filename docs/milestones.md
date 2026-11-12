# Milestones

## Webhook TLS, Routes, and Cleanup

The intake terminates TLS itself, attaches to a Gateway API gateway, and deletes its GitHub hooks when it stops. Each option is off by default.

* [Events Webhook Design](design/events-webhook.md)

## Launcher Target Namespace

A launcher entry submits its prompt as a remote run in another namespace. The agent runs with that repository's own config and the launcher pod holds no agent credentials.

* [Launcher Target Design](design/launcher-target.md)
* [Kubernetes Design](design/kube.md#in-cluster)
