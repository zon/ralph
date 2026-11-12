# Launcher Target Specification

## Purpose

The launcher pod submits each entry's prompt as a remote run in a target namespace and waits for it, so the agent runs with the repository's own config and the launcher pod holds no agent credentials. A launcher pod never runs the agent. An entry that sets a `namespace` targets it, and an entry with no `namespace` targets the launcher's own namespace, as defined in [launcher.md](launcher.md).

## Requirements

### Requirement: Entry namespace

Each prompt entry MAY have a `namespace`. When it is set, the chart SHALL pass the namespace to the launcher workflow. When it is unset, the chart SHALL pass the release namespace. In both cases the workflow SHALL pass the namespace to `ralph workflow prompt` as `--namespace`. The chart SHALL NOT render a launcher workflow that omits `--namespace`.

The target SHALL be a namespace of the launcher's own cluster. The chart SHALL NOT accept a target in another cluster.

#### Scenario: An entry with a namespace passes it to the container

- GIVEN an entry has `namespace: team-a`
- WHEN its workflow is submitted
- THEN `ralph workflow prompt` receives `--namespace team-a`

#### Scenario: An entry without a namespace targets the release namespace

- GIVEN a launcher is released in namespace `ralph`
- AND an entry has no `namespace`
- WHEN its workflow is submitted
- THEN `ralph workflow prompt` receives `--namespace ralph`
- AND no prompt pass runs in the launcher pod

#### Scenario: Entries target different namespaces

- GIVEN one entry has `namespace: team-a` and another has `namespace: team-b`
- WHEN both workflows are submitted
- THEN each passes its own namespace

---

### Requirement: Targeted container run

`ralph workflow prompt --namespace <namespace>` SHALL run these steps in order:

1. Prepare the workspace as defined in [workflow-workspace.md](workflow-workspace.md): GitHub authentication, then the clone of the `--clone` branch.
2. Check for an open pull request at the prefix. A match SHALL end the command successfully.
3. Run the prompt as a remote run from the clone, as defined in [prompt.md](prompt.md#requirement-remote-submission). The command SHALL resolve the base branch, check base branch synchronization, and submit a `ralph-prompt-` workflow into the target namespace with the prefix and no prompt branch. The command SHALL NOT propose a prompt slug or call a model. The target workflow SHALL propose the prompt slug, as defined in [prompt.md](prompt.md#requirement-container-entrypoint).
4. Follow the logs and wait for the target workflow to finish, with notifications off.
5. Exit with the target workflow's result.

The command SHALL NOT run the prompt pass in its own pod.

A failed target workflow SHALL fail the command. Stopping the launcher workflow SHALL leave the target workflow running.

#### Scenario: The prompt runs in the target namespace

- GIVEN `ralph workflow prompt --prefix review --namespace team-a`
- WHEN the container runs
- THEN a `ralph-prompt-` workflow is submitted into `team-a`
- AND no prompt pass runs in the container

#### Scenario: Open pull request skips the submission

- GIVEN an open pull request has a head branch starting with `review/`
- WHEN the container runs with `--prefix review --namespace team-a`
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

For each distinct target namespace, the chart SHALL render a Role and a RoleBinding in that namespace for the workflow ServiceAccount. The release namespace is a target namespace whenever an entry sets no `namespace` or names it. The Role SHALL allow `create`, `get`, `list`, and `watch` on workflows, and `get`, `list`, and `watch` on pods and pod logs. It SHALL allow nothing else.

`target.rbac.create: false` SHALL skip the Role and RoleBinding.

#### Scenario: A target namespace gets a Role and RoleBinding

- GIVEN an entry has `namespace: team-a`
- WHEN the chart renders
- THEN `team-a` holds a Role and a RoleBinding for the workflow ServiceAccount
- AND the Role allows the workflow, pod, and pod log verbs above

#### Scenario: Entries sharing a namespace share one Role

- GIVEN two entries have `namespace: team-a`
- WHEN the chart renders
- THEN `team-a` holds one Role and one RoleBinding

#### Scenario: Each target namespace gets its own Role

- GIVEN entries target `team-a` and `team-b`
- WHEN the chart renders
- THEN each namespace holds its own Role and RoleBinding

#### Scenario: Target RBAC can be skipped

- GIVEN `target.rbac.create` is `false`
- AND an entry has `namespace: team-a`
- WHEN the chart renders
- THEN no target Role or RoleBinding is rendered

#### Scenario: An entry without a namespace gets a Role in the release namespace

- GIVEN a launcher is released in namespace `ralph`
- AND no entry has a `namespace`
- WHEN the chart renders
- THEN `ralph` holds a Role and a RoleBinding for the workflow ServiceAccount

#### Scenario: The release namespace is not rendered twice

- GIVEN a launcher is released in namespace `ralph`
- AND one entry has no `namespace` and another has `namespace: ralph`
- WHEN the chart renders
- THEN `ralph` holds one target Role and one target RoleBinding

---

### Requirement: No agent credentials in the launcher pod

The launcher pod never runs the agent and never calls a model, so a release MAY leave `workflow.credentials.opencodeSecret` empty, and the launcher pod SHALL then mount no OpenCode Secret. When `workflow.credentials.opencodeSecret` names a Secret, the launcher pod SHALL mount it. A launcher pod with no OpenCode mount SHALL start and run any entry, because the OpenCode credentials step loads nothing when the mount is absent.

#### Scenario: No OpenCode Secret is set

- GIVEN `workflow.credentials.opencodeSecret` is empty
- AND an entry has no `namespace`
- WHEN the chart renders
- THEN the launcher pod mounts no OpenCode Secret

#### Scenario: A run starts without the mount

- GIVEN the launcher pod has no OpenCode mount
- AND the repository's `.ralph/config.yaml` names a model
- WHEN an entry runs
- THEN workspace preparation succeeds
- AND the launcher pod makes no model call
- AND the target workflow is submitted

#### Scenario: The Secret is mounted when set

- GIVEN `workflow.credentials.opencodeSecret` names a Secret
- WHEN the chart renders
- THEN the launcher pod mounts it at `/secrets/opencode`

---

### Requirement: Mutex and branch lock

The launcher mutex SHALL keep serializing runs of one entry. The target workflow SHALL take no lock, because the launcher sends a prefix and no branch, and a lock named from the prefix would collide with the launcher mutex while the launcher workflow waits for the target. The launcher workflow SHALL wait for the target workflow to finish.

#### Scenario: Runs of one entry queue

- GIVEN two events start the same entry
- WHEN both workflows are submitted
- THEN the second waits for the first launcher workflow
- AND the second target workflow starts only after the first has finished

#### Scenario: The target takes no lock

- GIVEN a launcher run submits its workflow with a prefix and no branch
- WHEN the target workflow starts
- THEN it holds no mutex
- AND it does not wait on the launcher mutex
