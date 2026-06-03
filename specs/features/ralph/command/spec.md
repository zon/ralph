# Command Specification

## Purpose

Submit an arbitrary command as an Argo Workflow on the current branch and stream its logs. Intended for testing ralph workflow configuration without AI iteration or PR creation.

## Requirements

### Requirement: Workflow Submission

The system SHALL generate and submit an Argo Workflow embedding the supplied command tokens when invoked as `ralph command -- <cmd>`.

#### Scenario: Command submitted

- GIVEN a command is provided and the current branch is in sync with the remote
- WHEN the user runs `ralph command -- <cmd>`
- THEN an Argo Workflow is generated embedding the command
- AND the workflow is submitted to the configured Kubernetes cluster

#### Scenario: Missing command

- GIVEN no command is provided after `--`
- WHEN the user runs `ralph command`
- THEN an error is returned with usage instructions

### Requirement: Workflow Monitoring

The system SHALL stream workflow logs by default after submission. The user MAY pass `--no-follow` to skip log streaming.

#### Scenario: Logs streamed by default

- GIVEN a command is submitted and `--no-follow` is not set
- WHEN the workflow starts
- THEN ralph streams the workflow logs until the workflow completes

#### Scenario: Follow suppressed

- GIVEN `--no-follow` is set
- WHEN the workflow is submitted
- THEN the workflow is submitted without streaming logs

### Requirement: Exit Code Propagation

The system SHALL reflect the workflow outcome as the process exit code.

#### Scenario: Workflow succeeds

- GIVEN the submitted workflow completes successfully
- WHEN log streaming finishes
- THEN ralph exits with code 0

#### Scenario: Workflow fails

- GIVEN the submitted workflow exits with a failure
- WHEN log streaming finishes
- THEN ralph exits with a non-zero code
