# Launcher Schedule Design

A [launcher](launcher.md) can run on a schedule as well as on events. A schedule starts the work stream when no event does, such as a daily check or a catch-up after a missed event.

## Schedule

`prompt.schedule` holds one cron expression or a list of them. A launcher can have events, a schedule, or both.

* A launcher with a schedule and no `prompt.events` runs on the schedule alone.
* A launcher with no schedule and no `prompt.events` uses the default `push` event.
* An empty `prompt.events` list accepts every event, with or without a schedule.

`schedule.timezone` sets the time zone for every expression in the release. It defaults to the Argo Workflows controller's zone.

## Prompt

A scheduled run carries no GitHub event. It passes `--event schedule` and reads `prompts.schedule` from the repository's `.ralph/config.yaml`, as an event run reads the key for its normalized event name.

The `schedule` key is reserved for this run.

A missing `prompts.schedule` or an empty prompt fails the run.

## Resources

The chart renders one `CronWorkflow` for a scheduled launcher in the release namespace, named `<prefix>-schedule-cron-launcher`. It submits the launcher's `WorkflowTemplate` with the same arguments as an event trigger, plus `--event schedule`. Argo names each run `<cronworkflow-name>-<unix-seconds>`, because a `CronWorkflow` run receives an explicit name rather than a generated one. See [Launcher Design](launcher.md#workflow-names).

The cron and the trigger share the same workflow settings: labels, TTL, pod garbage collection, and mutex rules. One helper renders the workflow body for both, so they cannot drift apart.

## Overlap

The `CronWorkflow` forbids concurrency. It skips a firing while the previous scheduled run still runs.

The schedule's mutex uses the `schedule` prefix. An event uses its own event prefix, so a scheduled run and an event run work at the same time. A scheduled run never overlaps another scheduled run.

## Open Pull Request

A scheduled run checks for an open pull request at the `schedule` prefix, as an event run does. A match ends the run successfully with no branch, no service, and no pass.
