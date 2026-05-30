# Run Remote Specification

## Purpose

Default behavior of `ralph run` (without `--local`): submits an Argo Workflow to a Kubernetes cluster and returns after submission. The workflow runs ralph in a container, executing the development loop remotely.

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
