# Events Delivery Specification

## Purpose

A repository event reaches a `ralph-launcher` release by one of two paths, and a launcher that is down when an event arrives receives it when it returns, within the bus retention window, when it uses the bus path. Each launcher chooses its path, and a cluster whose launchers all use the bus turns the `ralph-events` routing off.

## Requirements

### Requirement: Delivery path

The `ralph-launcher` chart SHALL read a `delivery` value of `subject` or `bus`. When `delivery` is unset it SHALL be `subject`. The chart SHALL reject any other value.

With `subject`, the launcher SHALL subscribe to the repository subject and republish each message to the namespace bus, as it does without this requirement. With `bus`, the launcher SHALL render no event source and hold no subject subscription, and its sensor SHALL depend on the intake's GitHub event source through the shared event bus.

#### Scenario: Subject is the default

- GIVEN a launcher does not set `delivery`
- WHEN the chart renders
- THEN the release has the launcher event source
- AND its sensor depends on that event source

#### Scenario: Bus renders no event source

- GIVEN a launcher sets `delivery: bus`
- WHEN the chart renders
- THEN the release has no event source
- AND its sensor depends on the intake event source

#### Scenario: Unknown delivery rejected

- GIVEN a launcher sets `delivery: queue`
- WHEN the chart renders
- THEN rendering fails with an error

---

### Requirement: Intake reference

A launcher with `delivery: bus` SHALL set `intake.eventSource.name` to the name of the intake's event source. The chart SHALL reject a bus launcher with no `intake.eventSource.name`. `intake.eventName` SHALL default to `github`, the name of the intake's GitHub event.

A launcher that uses `delivery: bus` SHALL still require the namespace bus, and that bus SHALL point at the same NATS server and account as the intake's bus. The chart cannot check this, so its release notes SHALL say so.

#### Scenario: Bus launcher without an intake name rejected

- GIVEN a launcher sets `delivery: bus`
- AND `intake.eventSource.name` is empty
- WHEN the chart renders
- THEN rendering fails with an error

#### Scenario: Event name defaults to github

- GIVEN a launcher sets `delivery: bus` and `intake.eventSource.name`
- AND does not set `intake.eventName`
- WHEN the chart renders
- THEN each dependency names the event `github`

#### Scenario: Release notes name the shared bus requirement

- GIVEN a launcher sets `delivery: bus`
- WHEN the release notes render
- THEN they say the namespace bus must use the intake's NATS server and account

---

### Requirement: Bus filters

A bus dependency SHALL read the GitHub delivery as the intake's event source publishes it, and SHALL NOT read the routing envelope. It SHALL read the repository from `body.repository.full_name`, the event name from the `X-Github-Event` header, the push branch from `body.ref`, and an ignored user from `body.sender.login`.

A bus dependency SHALL filter on the launcher's repository, so an event for another repository starts no entry.

`sensor.defaultBranchOnly`, `sensor.branch`, and `sensor.ignoreUsers` SHALL keep their meaning on both paths. Paths in `sensor.filters` SHALL be relative to the data of the chosen path.

#### Scenario: A bus launcher ignores other repositories

- GIVEN a launcher sets `delivery: bus`
- WHEN the intake publishes an event for a different repository
- THEN no entry runs

#### Scenario: A bus launcher reads the event name from the header

- GIVEN a bus entry lists `pull_request`
- WHEN a pull request event arrives
- THEN the entry runs
- AND a push event does not start it

#### Scenario: Default branch filter applies to bus pushes

- GIVEN `sensor.defaultBranchOnly` is enabled
- AND a bus entry lists `push`
- WHEN a push arrives on a branch other than `repo.baseBranch`
- THEN the entry does not run

#### Scenario: Ignored users do not start a bus entry

- GIVEN a push event comes from a user in `sensor.ignoreUsers`
- WHEN the sensor receives the event
- THEN no entry runs

---

### Requirement: Outage recovery

A bus sensor SHALL hold a durable consumer on the shared event bus stream. A sensor that is down when an event arrives SHALL receive the retained events when it comes back, as long as they are within the stream's `maxAge`. An event older than `maxAge` SHALL be lost.

`eventBus.streamConfig` SHALL set `maxAge` for a bus that the launcher creates. Delivery is at least once, so a launcher MAY see an event twice. The open pull request check and the entry mutex SHALL stop a repeat from starting a second pass.

A subject launcher SHALL receive only the events published while its event source is connected.

#### Scenario: A bus launcher receives events from its outage

- GIVEN a launcher uses `delivery: bus`
- AND its sensor is down while an event arrives
- WHEN the sensor comes back within the window
- THEN the entry runs for that event

#### Scenario: An event past the window is lost

- GIVEN a bus sensor is down longer than the stream `maxAge`
- WHEN the sensor comes back
- THEN it does not receive the event

#### Scenario: A repeated event does not start a second pass

- GIVEN a bus entry already started a pass for an event
- WHEN the same event is delivered again
- THEN the open pull request check or the entry mutex stops a second pass

#### Scenario: A subject launcher does not replay

- GIVEN a launcher uses `delivery: subject`
- AND its event source is down while an event is published
- WHEN the event source reconnects
- THEN the launcher does not receive that event

---

### Requirement: Routing

The `ralph-events` chart SHALL read `routing.enabled`, which defaults to `true`. With `routing.enabled: false`, the chart SHALL render no routing sensor and no stream job. It SHALL still render the GitHub event source and the event bus.

With `routing.enabled: true`, the chart SHALL render the routing sensors and the stream job as it does without this requirement.

#### Scenario: Routing is on by default

- GIVEN `routing.enabled` is unset
- WHEN the chart renders
- THEN the release has a routing sensor for each repository
- AND it has the stream job

#### Scenario: Routing off keeps the intake

- GIVEN `routing.enabled` is `false`
- WHEN the chart renders
- THEN the release has no routing sensor and no stream job
- AND it has the GitHub event source and the event bus
