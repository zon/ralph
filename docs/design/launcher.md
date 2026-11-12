# Launcher Design

`ralph-launcher` is one release per Ralph namespace. It listens to the repository subject published by [ralph-events](events.md) and starts Ralph prompt workflows in its namespace. One launcher maps several GitHub event types to prompt entries, each with its own slug and prompt. An entry can also run on a [schedule](launcher-schedule.md) or in a [target namespace](launcher-target.md).

The chart lives in `charts/ralph-launcher`. A push to `main` or `dev` publishes it to `oci://ghcr.io/zon/charts/ralph-launcher`.

## Flow

1. A NATS `EventSource` subscribes to `ralph.repo.<owner>.<name>` on the shared NATS server.
2. The event source republishes the message to the namespace event bus.
3. The sensor dependency for the matching entry and event passes the event to its trigger.
4. The trigger starts a `Workflow`.
5. The `WorkflowTemplate` runs `ralph workflow prompt` in the Ralph image.
6. The container clones `repo.cloneBranch`, checks for an open pull request at the entry prefix, proposes a prompt slug, checks out `<prefix>/<prompt-slug>`, runs the prompt text, pushes the branch, and opens a pull request.

## Components

The launcher owns the event source, the sensor, the `WorkflowTemplate`, and the prompt ConfigMap. It consumes the namespace `EventBus`. Argo Workflows owns the pods the workflow creates.

## Prompt Entries

`prompt.entries` lists the work streams the launcher runs. Each entry maps one slug and prompt to one or more GitHub event types:

```yaml
prompt:
  entries:
    - slug: review
      events: [push]
      text: |
        Review the repository and make one small improvement.
    - slug: triage
      events: [pull_request]
      text: |
        Read the pull request and leave a review.
```

`slug` is required and is a simple, branch-safe string. The chart passes it unchanged as `--prefix` to the prompt workflow. The chart rejects duplicate slugs. Launchers that target the same repository should use distinct slugs, because they share branches and open pull request checks. Use the same slug only when separate launchers should share a work stream.

Because the chart adds nothing to the slug, every event for an entry shares one prefix, so the open pull request check applies to the whole entry.

`events` lists the GitHub event names the entry handles. When omitted it defaults to `[push]`, and an empty list accepts every event. An entry with a [schedule](launcher-schedule.md) and no `events` handles no events. More than one entry may list the same event, and every matching entry runs.

`text` is required and holds the prompt text. The chart writes one ConfigMap key per entry, so an entry has its own prompt.

## Prompt Branch

After it clones the repository and before the pass, the container asks the AI for a prompt slug from the prompt text. It creates the branch `<slug>/<prompt-slug>`, so the `review` entry starts on `review/<prompt-slug>` and the chart does not name the branch. The open pull request check runs before the prompt slug proposal. When a pull request with the prefix is open, the workflow ends successfully without creating a prompt branch, starting a service, or running the pass.

When the pass leaves commits, Ralph asks the AI for a commit slug from the commit log and renames the branch to `<slug>/<commit-slug>` before the push and pull request. The prefix stays fixed, so the open pull request check and the mutex keep using it.

## NATS Authentication

Its NATS event source authenticates with the username and password in `nats.auth.secret`. See [NATS Authentication](events.md#nats-authentication) for the `nats.auth` settings.

## Prompt ConfigMap

The chart writes one ConfigMap key per entry. The workflow argument `prompt` reads the key for its entry with `valueFrom.configMapKeyRef`. A ConfigMap edit changes the prompt for the next workflow without changing the sensor or template.

## Filters

Each entry and event pair becomes a sensor dependency and trigger. Each dependency filters by event name, so the default branch filter applies only to push dependencies. The global `sensor.defaultBranchOnly`, `sensor.branch`, `sensor.ignoreUsers`, and `sensor.filters` apply to every matching dependency.

Set `sensor.defaultBranchOnly=false` to accept pushes to other branches. Add `sensor.filters` for extra Argo Events data filters. Fields are relative to the NATS event data, so the GitHub event name is `body.event` and the body is `body.payload`.

## Workflow

The `WorkflowTemplate` passes these values to the container:

```text
ralph workflow prompt --repo <owner/name> --clone <clone-branch>
  --prefix <entry-slug> --base <base-branch> --prompt <prompt>
```

It adds `--model`, `--variant`, `--agent`, `--no-services`, and `--verbose` when the values set them.

Each workflow takes a mutex named from the entry prefix, normalized to a Kubernetes name. The mutex serializes runs of the same work stream, because the branch name is not known until the container proposes its slug. `workflow.mutex.name` overrides the name for every entry.

The workflow mounts `github-credentials` at `/secrets/github` and `opencode-credentials` at `/secrets/opencode`. `ralph setup` writes both Secrets. See [Setup Design](setup.md) and [Workflow Container Design](workflow.md).

`workflow.labels` apply to the workflow pods, as they do in a [remote run](remote.md), and to the `Workflow` itself. The `WorkflowTemplate` carries them as pod template labels, so a webhook that reads a pod label at creation, such as for identity injection, sees them.

## Event Bus

The launcher consumes an Argo Events event bus named `eventBus.name`. With `delivery: bus` it also reads the intake's events from the shared bus and renders no event source. See [Events Delivery Design](events-delivery.md). The [ralph-events](events.md) release creates the bus in each launcher namespace with `eventbus.instances`.

Set `eventBus.create=true` to have the launcher create the bus itself, with `jetstreamExotic` pointing at the existing NATS server. The launcher does not create a bus by default.

The EventBus reads its credentials from `eventBus.accessSecret`. The event source uses the username and password keys from `nats.auth.secret`. See [Event Bus Credentials](events.md#event-bus-credentials).

The sensor and event source names include the release namespace, so their NATS subjects and consumers do not collide with another launcher. Every `EventBus` that points at the [shared NATS server](events.md#prerequisites) uses the same stream settings.

## RBAC

The sensor uses a ServiceAccount with permission to create and inspect workflows and read workflow templates. The chart creates the ServiceAccount, Role, and RoleBinding. A target namespace adds a Role and RoleBinding there. See [Launcher Target Design](launcher-target.md#rights). The workflow pods use the namespace default ServiceAccount unless `workflow.serviceAccountName` names another one.
