# Run Specification

## Purpose

Execute an AI-driven development workflow for a given project file, from branch creation through pull request submission.

## Requirements

### Requirement: Project Execution

The system SHALL run the full development workflow when given a project YAML file.

#### Scenario: Remote submission (default)

- GIVEN a valid project file and the current branch is in sync with the remote
- WHEN the user runs `ralph <project-file>`
- THEN an Argo Workflow is generated embedding the project file
- AND the workflow is submitted to the configured Kubernetes cluster
- AND the user is shown a command to follow logs

#### Scenario: Local execution

- GIVEN a valid project file and the `--local` flag is set
- WHEN the user runs `ralph <project-file> --local`
- THEN ralph creates the project branch, runs the iteration loop, and opens a PR when done

#### Scenario: Missing project file

- GIVEN no project file argument is provided
- WHEN the user runs `ralph`
- THEN an error is returned with usage instructions

#### Scenario: Project file not found

- GIVEN a project file path that does not exist on disk
- WHEN the user runs `ralph <missing-file>`
- THEN an error is returned indicating the file was not found

### Requirement: Branch Management

The system SHALL create and switch to a branch named `ralph/<project-name>` before executing.

#### Scenario: New branch

- GIVEN the project branch does not exist
- WHEN local execution begins
- THEN a branch `ralph/<project-name>` is created from the current branch
- AND the working tree is switched to that branch

#### Scenario: Base branch resolution

- GIVEN the user is on branch `feature/foo` and the project branch is `ralph/my-project`
- WHEN ralph starts
- THEN the base branch for the PR is set to `feature/foo`
- AND `--base` overrides this detection when provided

### Requirement: Iteration Loop

The system SHALL iterate over failing requirements until all pass or the iteration limit is reached. See [execution.md](execution.md) for the full behavior of each iteration: requirement selection, agent invocation, service management, committing, and blocking conditions.

#### Scenario: Before commands configured

- GIVEN `before` commands are defined in `.ralph/config.yaml`
- WHEN local execution begins
- THEN all before commands run sequentially before the iteration loop starts
- AND a non-zero exit from a non-optional command aborts execution

### Requirement: Pull Request Creation

For local execution (`--local`), the system SHALL create a pull request after the iteration loop completes. For remote execution, PR creation happens inside the Argo Workflow container — see [workflow.md](workflow.md).

#### Scenario: PR created (local)

- GIVEN `--local` is set and at least one commit was made ahead of the base branch
- WHEN the iteration loop finishes
- THEN an AI-generated PR summary is produced from the commit log
- AND a pull request is opened against the base branch
- AND the PR URL is printed

#### Scenario: No commits (local)

- GIVEN `--local` is set and all requirements were already passing before any iteration ran
- WHEN execution completes
- THEN no PR is created
- AND the user is notified of success

### Requirement: Single Iteration Mode

The system SHALL support running one iteration without branch creation or PR submission via `--once`.

#### Scenario: Once mode

- GIVEN the `--once` flag is set
- WHEN ralph runs
- THEN one development iteration executes on the current branch
- AND no branch is created and no PR is opened

#### Scenario: Incompatible flags

- GIVEN both `--once` and `--local` are set
- WHEN ralph starts
- THEN an error is returned before any work is done

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

The system SHOULD send a desktop notification when a local run completes or fails.

#### Scenario: Success notification

- GIVEN `--no-notify` is not set and the run succeeds
- WHEN local execution finishes
- THEN a success desktop notification is shown for the project name

#### Scenario: Failure notification

- GIVEN `--no-notify` is not set and the run fails
- WHEN local execution fails
- THEN an error desktop notification is shown for the project name
