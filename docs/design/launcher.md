# Launcher Design

`ralph-launcher` is one release per Ralph namespace. It listens to the repository subject published by [ralph-events](events.md) and starts a Ralph prompt workflow in its namespace.

The chart lives in `charts/ralph-launcher`. A push to `main` or `dev` publishes it to `oci://ghcr.io/zon/charts/ralph-launcher`.

## Flow

1. A NATS `EventSource` subscribes to `ralph.repo.<owner>.<name>` on the shared NATS server.
2. The event source republishes the message to the namespace event bus.
3. The sensor filters the event and starts a `Workflow`.
4. The `WorkflowTemplate` runs `ralph workflow prompt` in the Ralph image.
5. The container clones `repo.cloneBranch`, checks out `prompt.branch`, runs the prompt text, pushes the branch, and opens a pull request.

## Components

The launcher owns the event source, the sensor, the `WorkflowTemplate`, and the prompt ConfigMap. It consumes the namespace `EventBus`. Argo Workflows owns the pods the workflow creates.

## NATS Authentication

Its NATS event source authenticates with the username and password in `nats.auth.secret`. See [NATS Authentication](events.md#nats-authentication) for the `nats.auth` settings.

## Prompt ConfigMap

The chart writes `prompt.text` to a ConfigMap. The workflow argument `prompt` reads that key with `valueFrom.configMapKeyRef`. A ConfigMap edit changes the prompt for the next workflow without changing the sensor or template.

The prompt branch defaults to `ralph-prompt-<namespace>` so separate namespaces do not share a branch. A workflow mutex serializes runs on that branch.

## Filters

The default dependency filters accept `push` events on `refs/heads/<repo.baseBranch>` and drop pushes from `ralph-zon[bot]`. This prevents a Ralph push from starting another prompt run.

Set `sensor.events` to the GitHub event names that should start a workflow. The default is `[push]`. An empty list accepts every event:

```yaml
sensor:
  events:
    - push
    - pull_request
```

Set `sensor.defaultBranchOnly=false` to accept pushes to other branches. Add `sensor.filters` for extra Argo Events data filters. Fields are relative to the NATS event data, so the GitHub event name is `body.event` and the body is `body.payload`.

## Workflow

The `WorkflowTemplate` passes these values to the container:

```text
ralph workflow prompt --repo <owner/name> --clone <clone-branch>
  --branch <prompt-branch> --base <base-branch> --prompt <prompt>
```

It adds `--model`, `--variant`, `--agent`, `--no-services`, and `--verbose` when the values set them.

The workflow mounts `github-credentials` at `/secrets/github` and `opencode-credentials` at `/secrets/opencode`. `ralph setup` writes both Secrets. See [Setup Design](setup.md) and [Workflow Container Design](workflow.md).

## Event Bus

The launcher consumes an Argo Events event bus named `eventBus.name`. The [ralph-events](events.md) release creates the bus in each launcher namespace with `eventbus.instances`.

Set `eventBus.create=true` to have the launcher create the bus itself, with `jetstreamExotic` pointing at the existing NATS server. The launcher does not create a bus by default.

The EventBus reads its credentials from `eventBus.accessSecret`. The event source uses the username and password keys from `nats.auth.secret`. See [Event Bus Credentials](events.md#event-bus-credentials).

The sensor and event source names include the release namespace, so their NATS subjects and consumers do not collide with another launcher. Every `EventBus` that points at the [shared NATS server](events.md#prerequisites) uses the same stream settings.

## RBAC

The sensor uses a ServiceAccount with permission to create and inspect workflows and read workflow templates. The chart creates the ServiceAccount, Role, and RoleBinding. The workflow pods use the namespace default ServiceAccount unless `workflow.serviceAccountName` names another one.
