# Milestones

## Launcher Pod Labels

Launcher workflow pods carry `workflow.labels`, as remote run pods do. A cluster that needs a pod label at creation can run launcher prompts.

* [Launcher Design](design/launcher.md#workflow)

## Scheduled Launcher Entries

A launcher entry runs on a cron schedule, with or without events. A schedule covers work that no event starts and events a launcher missed.

* [Launcher Schedule Design](design/launcher-schedule.md)

## Durable Event Delivery

A launcher that is down when an event arrives receives it when it returns, within the bus retention window. A launcher chooses the subject path or the bus path. A cluster that uses only the bus path turns routing off.

* [Events Delivery Design](design/events-delivery.md)

## Webhook TLS, Routes, and Cleanup

The intake terminates TLS itself, attaches to a Gateway API gateway, and deletes its GitHub hooks when it stops. Each option is off by default.

* [Events Webhook Design](design/events-webhook.md)

## Launcher Target Namespace

A launcher entry submits its prompt as a remote run in another namespace. The agent runs with that repository's own config and the launcher pod holds no agent credentials.

* [Launcher Target Design](design/launcher-target.md)
* [Kubernetes Design](design/kube.md#in-cluster)
