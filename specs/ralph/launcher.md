# Launcher Specification

## Purpose

The `ralph-launcher` release listens to the repository subject and starts prompt workflows for the prompt entries, GitHub event types, and cron schedules it is configured with. Each entry maps one event list, one schedule list, or both to one prompt and one prefix. The launcher passes the prefix to the prompt workflow and does not name the prompt branch.

## Requirements

### Requirement: Prompt entries

The chart SHALL read a required `prompt.entries` list. Each entry SHALL have a required `slug` and a required `text`. Each entry MAY have an `events` list and MAY have a `schedule`.

A slug SHALL be a simple, branch-safe string. Slugs SHALL be unique within a launcher. The chart SHALL reject a launcher with a missing slug, a missing text, or duplicate slugs.

`text` SHALL be the prompt sent to the workflow.

`events` SHALL list the GitHub event names that start the entry. When omitted and the entry has no `schedule`, `events` SHALL default to `[push]`. When omitted and the entry has a `schedule`, the entry SHALL have no events. An empty list SHALL accept every event.

#### Scenario: Multiple entries map events to prompts

- GIVEN a launcher has a `review` entry with `events: [push]`
- AND a `triage` entry with `events: [pull_request]`
- WHEN the chart renders
- THEN each entry has its own prompt and prefix

#### Scenario: Duplicate slugs rejected

- GIVEN two entries use the slug `review`
- WHEN the chart renders
- THEN rendering fails with an error

#### Scenario: Entry without a slug rejected

- GIVEN an entry has no slug
- WHEN the chart renders
- THEN rendering fails with an error

#### Scenario: Entry without text rejected

- GIVEN an entry has no text
- WHEN the chart renders
- THEN rendering fails with an error

#### Scenario: Events default to push

- GIVEN an entry has no `events` and no `schedule`
- WHEN the chart renders
- THEN the entry handles `push` events

#### Scenario: Empty events accept every event

- GIVEN an entry has `events: []`
- WHEN an event arrives
- THEN the entry runs for that event

---

### Requirement: Event mapping

Each entry and event pair SHALL become a sensor dependency and trigger. An entry with an empty event list SHALL become one dependency and trigger with no event filter. Every entry whose event list contains the arriving event SHALL run. More than one entry MAY handle the same event.

The global `sensor.ignoreUsers` and `sensor.filters` SHALL apply to every dependency. `sensor.defaultBranchOnly` SHALL apply only to push dependencies.

#### Scenario: One entry handles several event types

- GIVEN an entry has `events: [push, pull_request]`
- WHEN a push or a pull request event arrives
- THEN the entry runs for that event

#### Scenario: One event starts several entries

- GIVEN two entries list `push`
- WHEN a push event arrives
- THEN both entries run

#### Scenario: Ignored users do not start an entry

- GIVEN a push event comes from a user in `sensor.ignoreUsers`
- WHEN the sensor receives the event
- THEN no entry runs

#### Scenario: Default branch filter applies to pushes

- GIVEN `sensor.defaultBranchOnly` is enabled
- AND an entry lists `push`
- WHEN a push arrives on a branch other than `repo.baseBranch`
- THEN the entry does not run

#### Scenario: Default branch filter does not apply to other events

- GIVEN `sensor.defaultBranchOnly` is enabled
- AND an entry lists `pull_request`
- WHEN a pull request event arrives
- THEN the entry can run

---

### Requirement: Scheduled entries

Each entry MAY have a `schedule` holding one cron expression or a list of expressions. An entry MAY have `events`, `schedule`, or both.

The release MAY set `schedule.timezone` to apply one time zone to every schedule expression. When `schedule.timezone` is unset, schedules SHALL use the Argo Workflows controller's time zone.

An entry with a schedule and no `events` SHALL start no workflow from an event. An entry with neither a schedule nor `events` SHALL handle `push`. An empty `events` list SHALL accept every event, with or without a schedule.

#### Scenario: Schedule alone starts the entry

- GIVEN an entry has `schedule: "0 9 * * *"` and no `events`
- WHEN no event arrives but the schedule fires
- THEN the entry starts a workflow
- AND a push event does not start the entry

#### Scenario: Events alone start the entry

- GIVEN an entry has `events: [pull_request]` and no `schedule`
- WHEN a pull request event arrives
- THEN the entry starts a workflow
- AND no schedule fires for the entry

#### Scenario: Events and schedule both start the entry

- GIVEN an entry has `schedule: "0 9 * * *"` and `events: [push]`
- WHEN either the schedule fires or a push event arrives
- THEN the entry starts a workflow

#### Scenario: A schedule list fires at each expression

- GIVEN an entry has `schedule: ["0 9 * * *", "0 17 * * *"]`
- WHEN either time arrives
- THEN the entry starts a workflow

#### Scenario: Empty events accept every event with a schedule

- GIVEN an entry has `schedule: "0 9 * * *"` and `events: []`
- WHEN any event arrives
- THEN the entry starts a workflow
- AND it also starts on the schedule

#### Scenario: Timezone applies to every schedule expression

- GIVEN the release sets `schedule.timezone: America/New_York`
- AND entries have different schedules
- WHEN the schedules fire
- THEN each expression fires in `America/New_York`

---

### Requirement: Scheduled workflow submission

The chart SHALL render one scheduled workflow for each entry with a schedule, in the release namespace. A scheduled workflow SHALL submit the launcher `WorkflowTemplate` with the same arguments as the entry's event trigger: repository, clone branch, prefix, base branch, prompt ConfigMap key, and workflow options.

The scheduled workflow and the event trigger SHALL share one source for workflow labels, TTL, pod garbage collection, and mutex, so the two paths cannot drift apart.

A scheduled workflow SHALL forbid concurrency and skip a firing while its previous run still runs. Its runs SHALL use the entry mutex, so a scheduled run and an event run of the same entry never work at the same time.

A scheduled run SHALL carry no event, so its prompt text is its only input.

#### Scenario: A scheduled entry renders one scheduled workflow

- GIVEN one entry has a schedule
- WHEN the chart renders
- THEN the release has one scheduled workflow for that entry
- AND it submits the launcher `WorkflowTemplate`

#### Scenario: Scheduled and event runs share workflow settings

- GIVEN an entry has both a schedule and `events`
- WHEN the chart renders
- THEN the scheduled workflow and the event trigger use the same labels, TTL, pod garbage collection, and mutex

#### Scenario: A firing is skipped while the previous run runs

- GIVEN a scheduled entry still has a scheduled run in progress
- WHEN the schedule fires again
- THEN the new firing is skipped

#### Scenario: A scheduled run and an event run do not overlap

- GIVEN a scheduled run of an entry is in progress
- WHEN an event starts the same entry
- THEN the event run waits for the scheduled run
- AND the two runs do not work at the same time

#### Scenario: A scheduled run has no event input

- GIVEN a scheduled run starts
- WHEN its arguments are built
- THEN it receives the entry prompt text
- AND it receives no event data

---

### Requirement: Workflow submission

The trigger SHALL start the launcher `WorkflowTemplate` with the entry prompt and the entry slug as `--prefix` to `ralph workflow prompt`. The launcher SHALL NOT send a prompt branch.

Each entry SHALL have its own ConfigMap key. The workflow `prompt` argument SHALL read the key for its entry.

Each workflow SHALL take a mutex named from the entry prefix. The mutex SHALL serialize runs of the same prefix.

#### Scenario: Entry slug is the prefix

- GIVEN an entry has the slug `review`
- WHEN its workflow is submitted
- THEN `ralph workflow prompt` receives `--prefix review`

#### Scenario: Launcher sends no branch

- GIVEN an entry starts a workflow
- WHEN the container arguments are built
- THEN they contain no prompt branch

#### Scenario: Entry prompt comes from its ConfigMap key

- GIVEN two entries have different prompts
- WHEN either workflow is submitted
- THEN its `prompt` argument reads that entry's ConfigMap key

#### Scenario: Same prefix serializes runs

- GIVEN two events start workflows for the same entry
- WHEN both workflows are submitted
- THEN they use the same mutex
- AND they do not run at the same time

#### Scenario: Different prefixes run independently

- GIVEN two events start workflows for different entries
- WHEN both workflows are submitted
- THEN they use different mutexes
- AND they may run at the same time

---

### Requirement: Workflow labels

`workflow.labels` SHALL add Kubernetes labels to each launcher `Workflow`, whether an event or a schedule starts it, and to its pod. The labels SHALL be present when the pod is created.

An empty `workflow.labels` SHALL add no labels.

#### Scenario: Labels reach the workflow and its pod

- GIVEN `workflow.labels` sets `team: platform`
- WHEN an entry starts a workflow
- THEN the `Workflow` has the label `team=platform`
- AND the pod has the label `team=platform` at creation

#### Scenario: Labels reach a scheduled workflow and its pod

- GIVEN `workflow.labels` sets `team: platform`
- AND an entry has a schedule
- WHEN the schedule starts a workflow
- THEN the `Workflow` has the label `team=platform`
- AND the pod has the label `team=platform` at creation

#### Scenario: No configured labels

- GIVEN `workflow.labels` is empty
- WHEN an entry starts a workflow
- THEN no configured label appears on the workflow or its pod

---

### Requirement: Prompt branch and pull request

The workflow SHALL run the prefix flow defined in [prompt.md](prompt.md#requirement-prefix-work-stream). The container SHALL propose the prompt slug and create the branch `<slug>/<prompt-slug>`.

When an open pull request has a head branch starting with `<slug>/`, the workflow SHALL end successfully before branch, service, or pass work. The pull request title SHALL start with `[<slug>] `.

#### Scenario: Container chooses the branch

- GIVEN the `review` entry starts a workflow
- WHEN the container proposes the prompt slug `fix-auth`
- THEN the prompt branch is `review/fix-auth`

#### Scenario: Open pull request stops the workflow

- GIVEN an open pull request has a head branch starting with `review/`
- WHEN the `review` entry starts a workflow
- THEN the workflow ends successfully without branch, service, or pass work

#### Scenario: Open pull request stops a scheduled run

- GIVEN an open pull request has a head branch starting with `review/`
- WHEN the `review` entry starts a run from its schedule
- THEN the run ends successfully without branch, service, or pass work

#### Scenario: Pull request title carries the entry prefix

- GIVEN the `review` entry opens a pull request
- WHEN the pull request opens
- THEN its title starts with `[review] `
