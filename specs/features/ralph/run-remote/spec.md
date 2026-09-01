# Run Remote Mode Specification

## Purpose

Behavior of the `remote` execution mode (`ralph run --mode remote`): submits an Argo Workflow to a Kubernetes cluster and returns after submission. The workflow runs ralph in a container, executing the development loop remotely. The shared `--context` and `--namespace` targeting contract is defined in [kubectl/spec.md](../kubectl/spec.md).

## Requirements

### Requirement: Branch must be in sync with remote before submission

The command SHALL verify that the current branch exists on the remote and that local and remote are at the same commit before submitting a workflow.

#### Scenario: Branch not pushed to remote

- GIVEN the current branch has no remote tracking ref
- WHEN the command checks branch sync
- THEN an error is returned indicating the branch has not been pushed

#### Scenario: Branch ahead of remote

- GIVEN the current branch is ahead of `origin/<branch>`
- WHEN the command checks branch sync
- THEN an error is returned indicating the branch is not in sync

#### Scenario: Branch behind remote

- GIVEN the current branch is behind `origin/<branch>`
- WHEN the command checks branch sync
- THEN an error is returned indicating the branch is not in sync

---

### Requirement: Workflow is submitted to Argo

The command SHALL generate an Argo Workflow for the project and submit it to the configured Kubernetes cluster.

#### Scenario: Successful workflow submission

- GIVEN the current branch is in sync with remote
- AND the Argo CLI is available
- WHEN the workflow is submitted
- THEN the workflow name is printed
- AND the command exits without waiting for the workflow to complete

#### Scenario: Log hint printed after submission

- GIVEN a workflow is submitted without `--follow`
- WHEN the workflow name is printed
- THEN ralph also prints the `argo logs` command the user can run to follow the workflow

---

### Requirement: Workflow Labeled as Ralph-Owned

The submitted workflow SHALL include the label `app.kubernetes.io/managed-by=ralph` in its metadata so that `ralph list` can filter for it.

#### Scenario: Label present on submitted workflow

- GIVEN `ralph run` generates a workflow for the project
- WHEN the workflow YAML is rendered
- THEN the workflow metadata contains the label `app.kubernetes.io/managed-by=ralph`

---

### Requirement: `--follow` streams logs after submission

With `--follow`, the command SHALL stream the workflow logs and wait for the workflow to finish before returning.

#### Scenario: `--follow` waits for completion

- GIVEN the user passes `--follow`
- AND the workflow is submitted successfully
- WHEN the workflow runs
- THEN ralph streams the Argo workflow logs and blocks until the workflow finishes

#### Scenario: Notification on followed workflow success

- GIVEN `--follow` is set and `--no-notify` is not set
- WHEN the followed workflow completes successfully
- THEN a success desktop notification is sent for the project slug

#### Scenario: Notification on followed workflow failure

- GIVEN `--follow` is set and `--no-notify` is not set
- WHEN the followed workflow fails
- THEN an error desktop notification is sent for the project slug

#### Scenario: Notifications suppressed

- GIVEN `--follow` is set and `--no-notify` is set
- WHEN the followed workflow completes or fails
- THEN no desktop notification is sent

---

### Requirement: `--debug` runs ralph from source inside the container

With `--debug <branch>`, the generated workflow SHALL check out the specified ralph source branch inside the container and invoke ralph via `go run` instead of the built binary.

#### Scenario: `--debug <branch>` selects a ralph source branch

- GIVEN the user passes `--debug my-fix`
- WHEN the workflow YAML is generated
- THEN the container checks out the `my-fix` branch of the ralph repository
- AND invokes ralph via `go run` instead of the pre-built binary

---

### Requirement: Base branch delivered to the workflow via `--base` argument

The base branch SHALL be resolved locally (see [run/spec.md](../run/spec.md)) and passed to the generated workflow as the `--base` CLI argument to `ralph workflow run`. The container SHALL NOT recompute the base branch.

#### Scenario: Resolved base branch passed as `--base` argument

- GIVEN a base branch has been resolved locally before workflow submission
- WHEN the workflow YAML is generated
- THEN the container args for `ralph workflow run` include `--base <resolved-base-branch>`

---

### Requirement: Item query delivered to the workflow via `--items` argument

The item query SHALL be resolved locally (see [run/spec.md](../run/spec.md)) and passed to the generated workflow as the `--items` CLI argument to `ralph workflow run`, so the query travels with the workflow rather than being re-read from `.ralph/config.yaml` inside the container. A remote run therefore indexes items consistently for its whole lifetime even if the repository's config changes underneath it.

#### Scenario: Resolved item query passed as `--items` argument

- GIVEN the item query resolved locally to `.requirements`
- WHEN the workflow YAML is generated
- THEN the container args for `ralph workflow run` include `--items .requirements`

#### Scenario: Default query passed explicitly

- GIVEN neither `--items` nor `items` in `.ralph/config.yaml` is set, so the query resolves to `.`
- WHEN the workflow YAML is generated
- THEN the container args include `--items .`, so the container does not re-resolve the query

#### Scenario: Config change after submission does not affect the run

- GIVEN a workflow has been submitted carrying `--items .requirements`
- AND `.ralph/config.yaml` is later changed on the branch
- WHEN the container runs
- THEN it resolves items with `.requirements` as submitted

---

### Requirement: Cleanup setting delivered to the workflow

The cleanup setting SHALL be resolved locally (see [run/spec.md](../run/spec.md)) and passed to the generated workflow as the `--cleanup` CLI argument to `ralph workflow run` when enabled.

#### Scenario: Cleanup enabled

- GIVEN cleanup resolved to enabled before workflow submission
- WHEN the workflow YAML is generated
- THEN the container args for `ralph workflow run` include `--cleanup`

#### Scenario: Cleanup disabled

- GIVEN cleanup resolved to disabled
- WHEN the workflow YAML is generated
- THEN the container args contain no `--cleanup` flag
