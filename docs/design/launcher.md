# Launcher Design

`ralph-launcher` is one release per Ralph namespace. It listens to the repository subject published by [ralph-events](events.md) and starts Ralph prompt workflows in its namespace. The chart lists the GitHub event types the launcher handles and an optional schedule. The repository's `.ralph/config.yaml` supplies the prompt for each event. Each launcher workflow submits the agent run into a [target namespace](launcher-target.md), which is the launcher's own namespace unless `target.namespace` names another.

The chart lives in `charts/ralph-launcher`. A push to `main` or `dev` publishes it to `oci://ghcr.io/zon/charts/ralph-launcher`.

## Flow

1. A NATS `EventSource` subscribes to `ralph.repo.<owner>.<name>` on the shared NATS server.
2. The event source republishes the message to the namespace event bus.
3. The sensor dependency for the matching event passes the event to its trigger.
4. The trigger starts a `Workflow` and passes the event name, work-stream prefix, and target namespace.
5. The `WorkflowTemplate` runs `ralph workflow prompt` in the Ralph image.
6. The container clones `repo.cloneBranch` and reads `.ralph/config.yaml`.
7. The container normalizes the event name, selects the matching `prompts` key, reads the named file when the value is a file, and fails when the event has no prompt.
8. The container checks for an open pull request at the event prefix.
9. The container submits a remote run of the prompt text and the prefix into the target namespace and waits for it. The target workflow checks again for an open pull request, proposes a prompt slug, creates `<event>/<prompt-slug>`, runs the prompt, pushes the branch, and opens a pull request.

## Components

The launcher owns the event source, the sensor, and the `WorkflowTemplate`. It consumes the namespace `EventBus`. Argo Workflows owns the pods the workflow creates.

## Prompts

`prompts` is a top-level key in `.ralph/config.yaml`. Each key names a GitHub event, or `schedule` for a scheduled run. Each value is a prompt string or a mapping with a `file`:

```yaml
prompts:
  push: |
    Review the repository and make one small improvement.
  pullRequest:
    file: .ralph/prompts/pull-request.md
  schedule:
    file: .ralph/prompts/daily.md
```

The string form is the prompt text. The `file` form names a file relative to the repository root, and the launcher reads it after the clone. The chart holds no prompt text and writes no prompt ConfigMap.

Keys use lower camel case. To look up an event, the launcher lowercases its name, removes `_` and `-`, and compares that form with each key treated the same way. The GitHub event `pull_request` reads `pullRequest`, `issue_comment` reads `issueComment`, and `push` reads `push`. Two keys that normalize alike are a config error.

A missing key, an empty prompt, an empty file, or a missing file fails the run with an error that names the event. The launcher does not fall back to another prompt.

`schedule` is the key for a scheduled run.

## Chart Events

`prompt.events` lists the GitHub event names the launcher handles:

```yaml
prompt:
  events: [push, pull_request]
```

It defaults to `[push]` when no schedule is set. When a schedule is set and `prompt.events` is omitted, the launcher handles no events. An empty list accepts every event. Each event name becomes its own work stream, and the launcher passes it as the prefix.

`prompt.events` uses the GitHub event names verbatim, because the sensor filters on the delivered names. The container normalizes an event name for the config lookup, as described in [Prompts](#prompts).

The chart rejects an empty event name and duplicate event names. The chart cannot read the repository config when it renders, so it does not reject an event that the config does not name. The run reports the missing prompt instead.

A launcher must set at least one event or a [schedule](launcher-schedule.md).

## Prompt Branch

After it clones the repository and before the pass, the target workflow's container asks the AI for a prompt slug from the prompt text. The launcher pod proposes no slug, so it needs no model. The target creates the branch `<event>/<prompt-slug>`, so a push run starts on `push/<prompt-slug>` and the chart does not name the branch. The open pull request check runs in the launcher pod before it submits and again in the target before the prompt slug proposal. When a pull request with the prefix is open, the workflow ends successfully without creating a prompt branch, starting a service, or running the pass.

Because the event name is the prefix, two launchers that handle the same event for one repository share branches and open pull request checks. Give a repository one launcher per event.

When the pass leaves commits, Ralph asks the AI for a commit slug from the commit log and renames the branch to `<event>/<commit-slug>` before the push and pull request. The prefix stays fixed, so the open pull request check and the mutex keep using it.

## NATS Authentication

Its NATS event source authenticates with the username and password in `nats.auth.secret`. See [NATS Authentication](events.md#nats-authentication) for the `nats.auth` settings.

## Filters

Each event in `prompt.events` becomes a sensor dependency and trigger. Each dependency filters by event name, so the default branch filter applies only to push dependencies. The global `sensor.defaultBranchOnly`, `sensor.branch`, `sensor.ignoreUsers`, and `sensor.filters` apply to every matching dependency.

Set `sensor.defaultBranchOnly=false` to accept pushes to other branches. Add `sensor.filters` for extra Argo Events data filters. Paths are relative to the data of the chosen [delivery path](events-delivery.md#filters), so switching paths rewrites them.

An empty `prompt.events` list renders one dependency with no event filter. Its trigger reads the event name from the delivery data and passes it as the event and prefix. The default branch filter does not apply, because the chart cannot tell a wildcard push from another event.

## Workflow Names

The `prefix` value names every workflow the launcher creates and defaults to `ralph`. Each name includes the event name and a generated suffix where the Kubernetes API supports one:

| Workflow | Name |
| --- | --- |
| Event-started launcher workflow | `<prefix>-<event>-launcher-<random>` |
| Scheduled launcher workflow | `<prefix>-schedule-cron-launcher` |
| Scheduled run | `<prefix>-schedule-cron-launcher-<unix-seconds>` |
| Target agent workflow | `<prefix>-<event>-<random>` |

The event is the GitHub event name, or `schedule` for a scheduled run. The sensor trigger uses `generateName`, so Kubernetes appends a random suffix. A `CronWorkflow` cannot itself use `generateName`, so it takes the fixed `<prefix>-schedule-cron-launcher` name and Argo names each run `<cronworkflow-name>-<unix-seconds>`. The launcher passes its `prefix` to the target as `--name-prefix`, which names the agent workflow `<prefix>-<event>-<random>`.

The prefix and event are lowercased, and `.`, `_`, and `/` become `-`, so the names are valid Kubernetes names. The branch prefix keeps the event name as written.

## Workflow

The `WorkflowTemplate` passes these values to the container:

```text
ralph workflow prompt --repo <owner/name> --clone <clone-branch>
  --prefix <event> --name-prefix <prefix> --base <base-branch>
  --event <event> --namespace <target-namespace>
```

It adds `--model`, `--variant`, `--agent`, `--no-services`, and `--verbose` when the values set them.

The launcher passes no prompt text. The container normalizes the event name and reads the matching `prompts` key from the cloned repository. A scheduled run passes `--event schedule`.

Each workflow takes a mutex named from the event, normalized to a Kubernetes name. The mutex serializes runs of the same work stream, because the branch name is not known until the target workflow proposes its slug. `workflow.mutex.name` overrides the name for every event.

The workflow mounts `github-credentials` at `/secrets/github`, and `opencode-credentials` at `/secrets/opencode` when `workflow.credentials.opencodeSecret` is set. The launcher pod never runs the agent, so it can leave the OpenCode Secret out. `ralph setup` writes both Secrets. See [Setup Design](setup.md) and [Workflow Container Design](workflow.md).

`workflow.labels` apply to the workflow pods, as they do in a [remote run](remote.md), and to the `Workflow` itself. The `WorkflowTemplate` carries them as pod template labels, so a webhook that reads a pod label at creation, such as for identity injection, sees them.

## Event Bus

The launcher consumes an Argo Events event bus named `eventBus.name`. With `delivery: bus` it also reads the intake's events from the shared bus and renders no event source. See [Events Delivery Design](events-delivery.md). The [ralph-events](events.md) release creates the bus in each launcher namespace with `eventbus.instances`.

Set `eventBus.create=true` to have the launcher create the bus itself, with `jetstreamExotic` pointing at the existing NATS server. The launcher does not create a bus by default.

The EventBus reads its credentials from `eventBus.accessSecret`. The event source uses the username and password keys from `nats.auth.secret`. See [Event Bus Credentials](events.md#event-bus-credentials).

The sensor and event source names include the release namespace, so their NATS subjects and consumers do not collide with another launcher. Every `EventBus` that points at the [shared NATS server](events.md#prerequisites) uses the same stream settings.

## RBAC

The sensor uses a ServiceAccount with permission to create and inspect workflows and read workflow templates. The chart creates the ServiceAccount, Role, and RoleBinding. The target namespace, including the release namespace by default, gets one Role and RoleBinding for the workflow ServiceAccount. See [Launcher Target Design](launcher-target.md#rights). The workflow pods use the namespace default ServiceAccount unless `workflow.serviceAccountName` names another one.
