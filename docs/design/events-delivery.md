# Events Delivery Design

How a repository event reaches a [launcher](launcher.md), and what survives a launcher outage.

## Paths

An event takes one of two paths from the [intake](events.md). The launcher's `delivery` value picks the path.

### Subject

`delivery: subject` is the default. The intake's routing sensor publishes the event to the repository subject. The launcher's event source subscribes with core NATS and republishes the message to the namespace bus.

Core NATS delivers only to connected subscribers. The launcher never sees an event published while its event source is down.

### Bus

`delivery: bus` skips the subject. The launcher's sensor depends on the intake's GitHub event source through the shared event bus.

Every Argo Events bus on one NATS server and account uses one JetStream stream. A sensor holds a durable consumer on it, named from the sensor. A sensor that is down receives the events retained meanwhile when it comes back.

The launcher renders no event source and holds no subject subscription. The namespace bus is still required, and it must point at the same NATS server and account as the intake's bus.

## Intake Reference

A bus launcher must set `intake.eventSource.name` to the intake's event source. `intake.eventName` defaults to `github`, the name of the intake's GitHub event.

The chart cannot check that the two buses share a server, so the release notes say so.

## Filters

A bus dependency reads the GitHub delivery as the intake's event source publishes it. It does not read the routing sensor's envelope. The paths are:

* The repository is `body.repository.full_name`.
* The event name is the `X-Github-Event` header.
* The push branch is `body.ref`.
* An ignored user is `body.sender.login`.

`sensor.defaultBranchOnly`, `sensor.branch`, and `sensor.ignoreUsers` keep their meaning on both paths. Paths in `sensor.filters` are relative to the data of the chosen path, so a launcher that switches paths rewrites them.

## Window

The bus stream's `maxAge` limits what a returning sensor receives. `eventBus.streamConfig` sets it. An event older than the window is lost.

Delivery is at least once, so a launcher can see an event twice. The open pull request check and the entry mutex stop a repeat.

An entry that must not miss a change adds a [schedule](launcher-schedule.md). The schedule runs the entry even when its event never arrived.

## Routing

The routing sensors and the stream job serve the subject path. `routing.enabled` defaults to `true`. With `false`, the intake renders no routing sensor and no stream job, only the event source and the bus.

The subject path connects to NATS with a static username and password, and no TLS settings. A server that needs TLS or token authentication takes the bus path, whose connection is the bus credential alone. A cluster whose launchers all use the bus sets `routing.enabled` to `false`.
