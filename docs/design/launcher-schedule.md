# Launcher Schedule Design

A [launcher](launcher.md) entry can run on a schedule as well as on events. A schedule starts the entry when no event does, such as a daily check or a catch-up after a missed event.

## Entry

`schedule` holds one cron expression or a list of them. An entry can have `events`, `schedule`, or both.

* An entry with `schedule` and no `events` runs on the schedule alone.
* An entry with no `schedule` and no `events` runs on `push`.
* An empty `events` list accepts every event.

`schedule.timezone` sets the time zone for every expression in the release. It defaults to the Argo Workflows controller's zone.

## Resources

The chart renders one `CronWorkflow` for each scheduled entry in the release namespace. It submits the launcher's `WorkflowTemplate` with the same arguments as the entry's sensor trigger.

The cron and the trigger share the same workflow settings: labels, TTL, pod garbage collection, and mutex. One helper renders them for both, so they cannot drift apart.

A scheduled run carries no event, so the prompt text is all it has.

## Overlap

The `CronWorkflow` forbids concurrency. It skips a firing while the previous scheduled run still runs.

The entry mutex covers both kinds of start. A scheduled run and an event run of one entry never work at the same time.

## Open Pull Request

A scheduled run checks for an open pull request at the entry prefix, as an event run does. A match ends the run successfully with no branch, no service, and no pass.
