# Launcher Target Specification

## Purpose

The launcher pod submits each run's prompt as a remote run in a target namespace and waits for it, so the agent runs with the repository's own config and the launcher pod holds no agent credentials. A launcher pod never runs the agent. Every run targets `target.namespace`, which is the launcher's own namespace by default.

## Requirements

### Requirement: Target namespace

`target.namespace` SHALL name the namespace every launcher run targets. When it is unset, the chart SHALL use the release namespace. The chart SHALL pass the result to the launcher workflow, and the workflow SHALL pass it to `ralph workflow prompt` as `--namespace`. The chart SHALL NOT render a launcher workflow that omits `--namespace`.

The target SHALL be a namespace of the launcher's own cluster. The chart SHALL NOT accept a target in another cluster.

#### Scenario: A configured target reaches the container

- GIVEN `target.namespace` is `team-a`
- WHEN a run starts
- THEN `ralph workflow prompt` receives `--namespace team-a`

#### Scenario: An unset target uses the release namespace

- GIVEN a launcher is released in namespace `ralph`
- AND `target.namespace` is unset
- WHEN a run starts
- THEN `ralph workflow prompt` receives `--namespace ralph`
- AND no prompt pass runs in the launcher pod

#### Scenario: Every event targets the same namespace

- GIVEN `target.namespace` is `team-a`
- AND the launcher handles `push` and `pull_request`
- WHEN either run starts
- THEN both target `team-a`

---

### Requirement: Targeted container run

`ralph workflow prompt --namespace <namespace>` SHALL run these steps in order:

1. Prepare the workspace as defined in [workflow-workspace.md](workflow-workspace.md): GitHub authentication, then the clone of the `--clone` branch.
2. Read `.ralph/config.yaml` and resolve the prompt for the event, following [prompts.md](prompts.md). A missing key or an empty prompt SHALL fail the command before any branch, service, or pass work.
3. Check for an open pull request at the prefix. A match SHALL end the command successfully.
4. Run the prompt as a remote run from the clone, as defined in [prompt.md](prompt.md#requirement-remote-submission). The command SHALL resolve the base branch, check base branch synchronization, and submit a `<prefix>-<event>-` workflow into the target namespace with the event prefix and no prompt branch. The launcher SHALL pass its chart `prefix` as `--name-prefix`, so the agent workflow is named `<prefix>-<event>-<random>`. The command SHALL NOT propose a prompt slug or call a model. The target workflow SHALL propose the prompt slug, as defined in [prompt.md](prompt.md#requirement-container-entrypoint).
5. Follow the logs and wait for the target workflow to finish, with notifications off.
6. Exit with the target workflow's result.

The command SHALL NOT run the prompt pass in its own pod.

A failed target workflow SHALL fail the command. Stopping the launcher workflow SHALL leave the target workflow running.

#### Scenario: The prompt runs in the target namespace

- GIVEN `ralph workflow prompt --prefix pull_request --event pull_request --name-prefix ralph --namespace team-a`
- WHEN the container runs
- THEN a `ralph-pull-request-` workflow is submitted into `team-a`
- AND no prompt pass runs in the container

#### Scenario: A missing prompt stops before the submission

- GIVEN the repository config has no prompt for the event
- WHEN the container runs
- THEN the command exits non-zero
- AND no workflow is submitted

#### Scenario: Open pull request skips the submission

- GIVEN an open pull request has a head branch starting with `pull_request/`
- WHEN the container runs with `--prefix pull_request --namespace team-a`
- THEN the command exits successfully
- AND no workflow is submitted

#### Scenario: Logs are followed without a notification

- GIVEN the target workflow is submitted
- WHEN the container waits for it
- THEN its logs stream to the container output
- AND no completion notification is sent

#### Scenario: Target success ends the command successfully

- GIVEN the target workflow succeeds
- WHEN the command ends
- THEN it exits zero

#### Scenario: Target failure fails the command

- GIVEN the target workflow fails
- WHEN the command ends
- THEN it exits non-zero

#### Scenario: Stopping the launcher leaves the target running

- GIVEN the target workflow is running
- WHEN the launcher workflow is stopped
- THEN the target workflow continues

---

### Requirement: Repository config governs the agent

The target workflow SHALL take its image, mounts, environment, labels, and resources from the `.ralph/config.yaml` of the cloned `repo.cloneBranch`, as any remote `ralph prompt` run does. A config change on that branch SHALL reach the next run with no launcher release.

The `--namespace` flag SHALL win over `workflow.namespace` in the checkout's config. The checkout's `workflow.context` SHALL NOT apply, as defined in [kube-options.md](kube-options.md#requirement-in-cluster-commands).

#### Scenario: Config change reaches the next run

- GIVEN the clone branch's `.ralph/config.yaml` sets a new `workflow.image`
- WHEN the next launcher run is submitted
- THEN the target workflow uses that image
- AND the launcher release is unchanged

#### Scenario: The flag wins over the configured namespace

- GIVEN the checkout's config sets `workflow.namespace: other`
- AND the container receives `--namespace team-a`
- WHEN the target workflow is submitted
- THEN it is submitted into `team-a`

#### Scenario: The configured context is ignored

- GIVEN the checkout's config sets `workflow.context: staging`
- WHEN the container submits the target workflow
- THEN the submission reaches the launcher's own cluster
- AND the `staging` context is not used

---

### Requirement: Target rights

The chart SHALL render one Role and one RoleBinding in the target namespace for the workflow ServiceAccount. The Role SHALL allow `create`, `get`, `list`, and `watch` on workflows, and `get`, `list`, and `watch` on pods and pod logs. It SHALL allow nothing else.

`target.rbac.create: false` SHALL skip the Role and RoleBinding.

#### Scenario: The target namespace gets a Role and RoleBinding

- GIVEN `target.namespace` is `team-a`
- WHEN the chart renders
- THEN `team-a` holds a Role and a RoleBinding for the workflow ServiceAccount
- AND the Role allows the workflow, pod, and pod log verbs above

#### Scenario: The release namespace is the target when unset

- GIVEN a launcher is released in namespace `ralph`
- AND `target.namespace` is unset
- WHEN the chart renders
- THEN `ralph` holds the target Role and RoleBinding

#### Scenario: Target RBAC can be skipped

- GIVEN `target.rbac.create` is `false`
- AND `target.namespace` is `team-a`
- WHEN the chart renders
- THEN no target Role or RoleBinding is rendered

#### Scenario: The release namespace is not rendered twice

- GIVEN a launcher is released in namespace `ralph`
- AND `target.namespace` is `ralph`
- WHEN the chart renders
- THEN `ralph` holds one target Role and one target RoleBinding

---

### Requirement: No agent credentials in the launcher pod

The launcher pod never runs the agent and never calls a model, so a release MAY leave `workflow.credentials.opencodeSecret` empty, and the launcher pod SHALL then mount no OpenCode Secret. When `workflow.credentials.opencodeSecret` names a Secret, the launcher pod SHALL mount it. A launcher pod with no OpenCode mount SHALL start and run any event, because the OpenCode credentials step loads nothing when the mount is absent.

#### Scenario: No OpenCode Secret is set

- GIVEN `workflow.credentials.opencodeSecret` is empty
- WHEN the chart renders
- THEN the launcher pod mounts no OpenCode Secret

#### Scenario: A run starts without the mount

- GIVEN the launcher pod has no OpenCode mount
- AND the repository's `.ralph/config.yaml` names a model
- WHEN a run starts
- THEN workspace preparation succeeds
- AND the launcher pod makes no model call
- AND the target workflow is submitted

#### Scenario: The Secret is mounted when set

- GIVEN `workflow.credentials.opencodeSecret` names a Secret
- WHEN the chart renders
- THEN the launcher pod mounts it at `/secrets/opencode`

---

### Requirement: Mutex and lock

The launcher mutex SHALL serialize runs of one event. The schedule SHALL use the `schedule` mutex. A scheduled run and an event run SHALL NOT share a mutex, so they may work at the same time.

The target workflow SHALL take no lock, because the launcher sends a prefix and no branch, and a lock named from the prefix would collide with the launcher mutex while the launcher workflow waits for the target. The launcher workflow SHALL wait for the target workflow to finish.

#### Scenario: Runs of one event queue

- GIVEN two events start the same work stream
- WHEN both workflows are submitted
- THEN the second waits for the first launcher workflow
- AND the second target workflow starts only after the first has finished

#### Scenario: A scheduled run does not wait for an event run

- GIVEN an event run is in progress
- WHEN the schedule starts a run
- THEN the scheduled run starts without waiting

#### Scenario: The target takes no lock

- GIVEN a launcher run submits its workflow with a prefix and no branch
- WHEN the target workflow starts
- THEN it holds no mutex
- AND it does not wait on the launcher mutex
