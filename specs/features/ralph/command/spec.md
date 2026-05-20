# Command Specification

## Purpose

Run an arbitrary command through the ralph workflow infrastructure (before-commands, services) on the current branch, without branch creation, AI iteration, or PR creation. Intended for testing ralph workflow configuration.

## Requirements

### Requirement: Command Execution via Workflow

The system SHALL run a user-supplied command through the full ralph workflow (excluding AI iteration) when invoked as `ralph command -- <cmd>`.

#### Scenario: Remote submission (default)

- GIVEN a command is provided and the current branch is in sync with the remote
- WHEN the user runs `ralph command -- <cmd>`
- THEN an Argo Workflow is generated embedding the command
- AND the workflow is submitted to the configured Kubernetes cluster
- AND the user is shown a command to follow logs

#### Scenario: Local execution

- GIVEN a command is provided and the `--local` flag is set
- WHEN the user runs `ralph command --local -- <cmd>`
- THEN ralph runs the before-commands, starts any configured services, and executes the supplied command on the current branch

#### Scenario: Missing command

- GIVEN no command is provided after `--`
- WHEN the user runs `ralph command`
- THEN an error is returned with usage instructions

### Requirement: Before Commands

The system SHALL run before-commands from `.ralph/config.yaml` before executing the supplied command, matching the behavior in `ralph run`.

#### Scenario: Before commands configured

- GIVEN `before` commands are defined in `.ralph/config.yaml`
- WHEN local execution begins
- THEN all before commands run sequentially before the supplied command
- AND a non-zero exit from a non-optional before command aborts execution

#### Scenario: No before commands

- GIVEN no `before` commands are defined
- WHEN local execution begins
- THEN the supplied command runs directly

### Requirement: Command Exit Code Propagation

The system SHALL treat the exit code of the supplied command as the run result.

#### Scenario: Command succeeds

- GIVEN the supplied command exits with code 0
- WHEN the command finishes
- THEN execution is considered successful

#### Scenario: Command fails

- GIVEN the supplied command exits with a non-zero code
- WHEN the command finishes
- THEN execution is considered failed and the non-zero exit is reported

### Requirement: Workflow Monitoring

The system SHOULD allow the user to follow Argo Workflow logs in real time via `--follow`.

#### Scenario: Follow logs

- GIVEN `--follow` (or `-f`) is set and `--local` is not set
- WHEN the workflow is submitted
- THEN ralph streams the workflow logs until the workflow completes

#### Scenario: Incompatible flags

- GIVEN both `--follow` and `--local` are set
- WHEN ralph starts
- THEN an error is returned

### Requirement: Desktop Notifications

The system SHOULD send a desktop notification when a local command run completes or fails.

#### Scenario: Success notification

- GIVEN `--no-notify` is not set and the command succeeds
- WHEN local execution finishes
- THEN a success desktop notification is shown

#### Scenario: Failure notification

- GIVEN `--no-notify` is not set and the command fails
- WHEN local execution fails
- THEN an error desktop notification is shown
