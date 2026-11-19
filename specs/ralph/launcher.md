# Launcher Specification

## Purpose

The `ralph-launcher` release listens to the repository subject, or to the shared event bus as defined in [events-delivery.md](events-delivery.md), and starts prompt workflows for the GitHub events and cron schedule it is configured with. The repository's `.ralph/config.yaml` supplies the prompt for each event, following [prompts.md](prompts.md). Each event and the schedule is its own work stream, and the launcher passes the event name as the work-stream prefix. The launcher does not name the prompt branch.

## Requirements

### Requirement: Launcher events

The chart SHALL read `prompt.events`, a list of GitHub event names. When the launcher has no schedule and `prompt.events` is absent, the list SHALL default to `[push]`. When the launcher has a schedule and `prompt.events` is absent, the launcher SHALL handle no events. An empty list SHALL accept every event.

Event names SHALL be the GitHub event names used verbatim, because the sensor filters on the delivered names.

The chart SHALL reject an empty event name and duplicate event names. The chart SHALL NOT reject an event that the repository config does not name, because it cannot read that config. The run SHALL report the missing prompt instead.

A launcher SHALL set at least one event or a schedule. The default `push` event satisfies this when the launcher sets neither.

#### Scenario: Events default to push

- GIVEN `prompt.events` is absent
- AND the launcher has no schedule
- WHEN the chart renders
- THEN the launcher handles `push` events

#### Scenario: A schedule alone handles no events

- GIVEN `prompt.events` is absent
- AND the launcher has a schedule
- WHEN the chart renders
- THEN the launcher handles no events
- AND it still renders the scheduled workflow

#### Scenario: An empty event list accepts every event

- GIVEN `prompt.events` is empty
- WHEN any event arrives
- THEN the launcher starts that event's work stream

#### Scenario: Duplicate events rejected

- GIVEN `prompt.events` lists `push` twice
- WHEN the chart renders
- THEN rendering fails with an error

#### Scenario: Empty event name rejected

- GIVEN `prompt.events` lists an empty string
- WHEN the chart renders
- THEN rendering fails with an error

#### Scenario: A missing repository prompt is reported at run time

- GIVEN the launcher lists `pull_request`
- AND the repository config has no `pullRequest` or `pull_request` key
- WHEN the chart renders
- THEN rendering succeeds
- AND the run fails with an error that names the event

---

### Requirement: Event mapping

Each event in `prompt.events` SHALL become one sensor dependency and trigger. A dependency SHALL filter on its event name. Every event SHALL start its own run.

An empty `prompt.events` list SHALL render one dependency with no event filter. Its trigger SHALL read the delivered event name from the delivery data and pass it as the event and the prefix.

The global `sensor.ignoreUsers` and `sensor.filters` SHALL apply to every dependency. `sensor.defaultBranchOnly` SHALL apply only to push dependencies. The data paths of these filters depend on the delivery path, as defined in [events-delivery.md](events-delivery.md#requirement-bus-filters).

#### Scenario: One event starts its run

- GIVEN `prompt.events` lists `pull_request`
- WHEN a pull request event arrives
- THEN one run starts for that event

#### Scenario: Several events run independently

- GIVEN `prompt.events` lists `push` and `pull_request`
- WHEN both events arrive
- THEN each starts its own run

#### Scenario: A wildcard reads the delivered event name

- GIVEN `prompt.events` is empty
- WHEN a `pull_request` event arrives
- THEN the run uses `pull_request` as its event and prefix
- AND its workflow and mutex names use `pull-request`

#### Scenario: Ignored users do not start a run

- GIVEN a push event comes from a user in `sensor.ignoreUsers`
- WHEN the sensor receives the event
- THEN no run starts

#### Scenario: Default branch filter applies to pushes

- GIVEN `sensor.defaultBranchOnly` is enabled
- AND `prompt.events` lists `push`
- WHEN a push arrives on a branch other than `repo.baseBranch`
- THEN the run does not start

#### Scenario: Default branch filter does not apply to other events

- GIVEN `sensor.defaultBranchOnly` is enabled
- AND `prompt.events` lists `pull_request`
- WHEN a pull request event arrives
- THEN the run starts

---

### Requirement: Repository prompt

The trigger SHALL pass the event name to `ralph workflow prompt` as `--event` and as `--prefix`. It SHALL pass no prompt text.

After the clone, the container SHALL resolve the prompt from `.ralph/config.yaml`, following [prompts.md](prompts.md). A missing key, an empty prompt, or a missing or empty file SHALL fail the run before branch, service, or pass work starts.

The chart SHALL hold no prompt text and SHALL render no prompt ConfigMap.

#### Scenario: The event selects the prompt

- GIVEN `prompt.events` lists `pull_request`
- AND the repository config sets `prompts.pullRequest`
- WHEN the run starts
- THEN `ralph workflow prompt` receives `--event pull_request`
- AND it receives no prompt text
- AND the repository config supplies the prompt

#### Scenario: A missing prompt fails before branch work

- GIVEN the repository config has no prompt for the event
- WHEN the run starts
- THEN it fails with an error that names the event
- AND no branch, service, or pass work starts

#### Scenario: The chart holds no prompt text

- GIVEN a launcher release
- WHEN the chart renders
- THEN the release has no prompt ConfigMap
- AND its workflow template carries no prompt text

---

### Requirement: Scheduled runs

`prompt.schedule` SHALL hold one cron expression or a list of expressions. The launcher MAY have events, a schedule, or both.

`schedule.timezone` SHALL set one time zone for every expression in the release. When it is unset, the expressions SHALL use the Argo Workflows controller's time zone.

The chart SHALL render one `CronWorkflow` for the launcher in the release namespace, named `<prefix>-schedule-cron-launcher`. It SHALL submit the launcher `WorkflowTemplate` with the same repository, clone branch, base branch, name prefix, target namespace, and workflow options as an event trigger, and with `schedule` as the event and prefix.

The scheduled workflow and the event trigger SHALL share one source for workflow labels, TTL, pod garbage collection, and mutex settings, so the two paths cannot drift apart.

The `CronWorkflow` SHALL forbid concurrency and skip a firing while its previous run still runs.

A scheduled run SHALL pass `--event schedule` and read `prompts.schedule` from the repository config. A missing prompt or an empty prompt SHALL fail the run.

The schedule SHALL use its own mutex. It SHALL NOT share a mutex with an event, so a scheduled run and an event run may work at the same time. A scheduled run SHALL NOT overlap another scheduled run.

#### Scenario: A schedule renders one scheduled workflow

- GIVEN `prompt.schedule` holds a cron expression
- WHEN the chart renders
- THEN the release has one `CronWorkflow`
- AND it is named `<prefix>-schedule-cron-launcher`
- AND it submits the launcher `WorkflowTemplate`

#### Scenario: A schedule list fires at each expression

- GIVEN `prompt.schedule` holds two cron expressions
- WHEN either time arrives
- THEN the scheduled workflow starts a run

#### Scenario: Timezone applies to every expression

- GIVEN the release sets `schedule.timezone: America/New_York`
- WHEN the schedule fires
- THEN the expression fires in `America/New_York`

#### Scenario: Scheduled and event runs share workflow settings

- GIVEN the launcher has a schedule and `prompt.events`
- WHEN the chart renders
- THEN both paths use the same labels, TTL, pod garbage collection, and workflow body

#### Scenario: A firing is skipped while the previous run runs

- GIVEN a scheduled run is in progress
- WHEN the schedule fires again
- THEN the new firing is skipped

#### Scenario: A scheduled run and an event run may overlap

- GIVEN a scheduled run is in progress
- WHEN an event starts a run
- THEN the event run does not wait for the scheduled run

#### Scenario: A scheduled run uses the schedule prompt

- GIVEN the repository config sets `prompts.schedule`
- WHEN a scheduled run starts
- THEN `ralph workflow prompt` receives `--event schedule`
- AND the repository config supplies the prompt

#### Scenario: A missing schedule prompt fails

- GIVEN the repository config has no `schedule` prompt
- WHEN a scheduled run starts
- THEN it fails with an error that names `schedule`

---

### Requirement: Workflow names

The chart SHALL read `prefix`, defaulting to `ralph`, and SHALL use it with the event name to name every workflow the launcher creates.

| Workflow | Name |
| --- | --- |
| Event-started launcher workflow | `<prefix>-<event>-launcher-<random>` |
| Scheduled launcher workflow | `<prefix>-schedule-cron-launcher` |
| Scheduled run | `<prefix>-schedule-cron-launcher-<unix-seconds>` |
| Target agent workflow | `<prefix>-<event>-<random>` |

The event is the GitHub event name, or `schedule` for a scheduled run. The sensor trigger SHALL use `generateName`, so Kubernetes appends a generated random suffix. The `CronWorkflow` SHALL take the fixed `<prefix>-schedule-cron-launcher` name, and Argo SHALL name each run `<cronworkflow-name>-<unix-seconds>`.

The launcher SHALL pass `prefix` to `ralph workflow prompt` as `--name-prefix`, so the target agent workflow is named `<prefix>-<event>-` with a generated suffix.

The prefix and event SHALL be lowercased, and `.`, `_`, and `/` SHALL become `-`, before they are used in a name. The branch prefix SHALL keep the event name as written.

#### Scenario: Event workflow carries the prefix and event

- GIVEN `prefix` is `ralph`
- WHEN a `pull_request` event starts a run
- THEN the launcher workflow name starts with `ralph-pull-request-launcher-`

#### Scenario: Custom prefix names the event workflow

- GIVEN `prefix` is `team`
- WHEN a push event starts a run
- THEN the launcher workflow name starts with `team-push-launcher-`

#### Scenario: Scheduled workflow has a stable name

- GIVEN `prefix` is `ralph`
- WHEN the chart renders a schedule
- THEN the `CronWorkflow` is named `ralph-schedule-cron-launcher`
- AND each run is named `ralph-schedule-cron-launcher-<unix-seconds>`

#### Scenario: Target agent workflow carries the prefix and event

- GIVEN `prefix` is `ralph`
- WHEN the launcher submits a target agent workflow for `pull_request`
- THEN its name starts with `ralph-pull-request-`

#### Scenario: Name characters are normalized

- GIVEN `prefix` is `Team.AI`
- WHEN an event named `pull_request` starts a run
- THEN the workflow name uses `team-ai-pull-request`

#### Scenario: The branch keeps the event as written

- GIVEN the launcher handles `pull_request`
- WHEN it creates the prompt branch
- THEN the branch starts with `pull_request/`

---

### Requirement: Workflow submission

The trigger SHALL start the launcher `WorkflowTemplate` with the repository, clone branch, event as `--prefix`, chart `prefix` as `--name-prefix`, base branch, event as `--event`, and target namespace. It SHALL pass the configured model, variant, agent, `--no-services`, and `--verbose` when they are set.

The launcher SHALL NOT send a prompt branch and SHALL NOT send prompt text.

Each work stream SHALL take a mutex named from the event, normalized to a Kubernetes name. `workflow.mutex.name` SHALL override the name for every event.

#### Scenario: The event is the prefix

- GIVEN the launcher handles `pull_request`
- WHEN its workflow is submitted
- THEN `ralph workflow prompt` receives `--prefix pull_request`
- AND it receives `--event pull_request`

#### Scenario: Launcher sends no branch

- GIVEN an event starts a run
- WHEN the container arguments are built
- THEN they contain no prompt branch

#### Scenario: Same event serializes runs

- GIVEN two pull request events start runs
- WHEN both workflows are submitted
- THEN they use the same mutex
- AND they do not run at the same time

#### Scenario: Different events run independently

- GIVEN a push event and a pull request event start runs
- WHEN both workflows are submitted
- THEN they use different mutexes
- AND they may run at the same time

#### Scenario: A configured mutex overrides every event

- GIVEN `workflow.mutex.name` is `repo-lock`
- WHEN any event starts a run
- THEN its workflow uses the mutex `repo-lock`

---

### Requirement: Workflow labels

`workflow.labels` SHALL add Kubernetes labels to each launcher `Workflow`, whether an event or a schedule starts it, and to its pod. The labels SHALL be present when the pod is created.

An empty `workflow.labels` SHALL add no labels.

#### Scenario: Labels reach the workflow and its pod

- GIVEN `workflow.labels` sets `team: platform`
- WHEN an event starts a run
- THEN the `Workflow` has the label `team=platform`
- AND the pod has the label `team=platform` at creation

#### Scenario: Labels reach a scheduled workflow and its pod

- GIVEN `workflow.labels` sets `team: platform`
- AND the launcher has a schedule
- WHEN the schedule starts a run
- THEN the `Workflow` has the label `team=platform`
- AND the pod has the label `team=platform` at creation

#### Scenario: No configured labels

- GIVEN `workflow.labels` is empty
- WHEN a run starts
- THEN no configured label appears on the workflow or its pod

---

### Requirement: Prompt branch and pull request

The target workflow's container SHALL propose the prompt slug from the resolved prompt and create the branch `<event>/<prompt-slug>`. The launcher pod SHALL NOT propose a slug or call a model.

Because the event name is the prefix, two launchers that handle the same event for one repository SHALL share branches and open pull request checks. A repository SHOULD have one launcher per event.

When the pass leaves commits, Ralph SHALL ask the AI for a commit slug from the commit log and rename the branch to `<event>/<commit-slug>` before the push and pull request. The prefix SHALL stay fixed.

When an open pull request has a head branch starting with `<event>/`, the workflow SHALL end successfully before branch, service, or pass work.

#### Scenario: Container chooses the branch

- GIVEN a `pull_request` run resolves a prompt
- WHEN the target container proposes the prompt slug `fix-auth`
- THEN the prompt branch is `pull_request/fix-auth`

#### Scenario: The launcher pod proposes no slug

- GIVEN the launcher pod has no OpenCode configuration
- WHEN a run starts
- THEN the launcher pod makes no model call
- AND it submits the target workflow with the event prefix and no branch

#### Scenario: Open pull request stops the run

- GIVEN an open pull request has a head branch starting with `push/`
- WHEN a push run starts
- THEN the run ends successfully without branch, service, or pass work

#### Scenario: Open pull request stops a scheduled run

- GIVEN an open pull request has a head branch starting with `schedule/`
- WHEN a scheduled run starts
- THEN the run ends successfully without branch, service, or pass work
