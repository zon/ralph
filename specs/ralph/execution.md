# Execution Specification

## Purpose

Define the behavior of a single project execution iteration: requirement selection, AI agent invocation, post-agent cleanup, commit, and blocking conditions.

## Requirements

### Requirement: Requirement Selection

Each iteration MUST use a picker AI agent to select the highest-priority failing requirement before running the development agent.

#### Scenario: Picker agent runs

- GIVEN a project with one or more requirements where `passing: false`
- WHEN an iteration begins
- THEN the picker agent is invoked with the full project content and recent commit log
- AND the agent writes its selection to `picked-requirement.yaml`

#### Scenario: Picker agent fails

- GIVEN the picker agent exits with an error
- WHEN the failure is detected
- THEN `blocked.md` is written at the repo root with the error details
- AND the iteration loop exits with an error

#### Scenario: All requirements already passing

- GIVEN all requirements have `passing: true`
- WHEN the iteration loop checks completion before starting an iteration
- THEN no picker agent is invoked and the loop exits cleanly

### Requirement: Development Agent

The system SHALL invoke a development AI agent with the selected requirement and full project context to implement changes.

#### Scenario: Agent runs successfully

- GIVEN `picked-requirement.yaml` was written by the picker agent
- WHEN the development agent runs
- THEN the agent implements the selected requirement
- AND updates `passing: true` on the requirement in the project file when complete

#### Scenario: Agent failure

- GIVEN the development agent exits with an error
- WHEN the failure is detected
- THEN `blocked.md` is written at the repo root
- AND the iteration loop exits with an error

#### Scenario: Fatal AI provider error

- GIVEN the AI provider returns a billing, account, or quota error
- WHEN the error is detected
- THEN the loop exits immediately with `ErrFatalOpenCodeError`
- AND no further iterations are attempted

### Requirement: Service Management

The system SHALL start configured services before the agent runs and stop them after each iteration completes.

#### Scenario: Services started successfully

- GIVEN services are configured in `.ralph/config.yaml` and `--no-services` is not set
- WHEN an iteration begins
- THEN all services are started in order before the picker and development agents run
- AND services are stopped when the iteration completes

#### Scenario: Service startup failure triggers AI fix

- GIVEN a configured service fails to start
- WHEN the failure is detected
- THEN the development agent is invoked with a prompt to diagnose and fix the startup failure
- AND the iteration proceeds after the fix attempt

#### Scenario: Port health check

- GIVEN a service has a `port` field configured
- WHEN the service is started
- THEN ralph waits for a TCP connection to that port to succeed before proceeding

### Requirement: Post-Agent Cleanup

The system SHALL normalize the project file and remove service logs after each agent run.

#### Scenario: Project file normalized

- GIVEN the agent may have added trailing newlines to the project file
- WHEN the agent finishes
- THEN excess trailing newlines are stripped from the project file
- AND the project file is staged if it has changes

#### Scenario: Service logs removed

- GIVEN services produced log files during the iteration
- WHEN the agent finishes
- THEN service log files are deleted from the working directory

### Requirement: Commit After Iteration

The system SHALL commit all changes after each iteration using `report.md` as the commit message.

#### Scenario: Report written by agent

- GIVEN the agent produced a `report.md` file
- WHEN changes are committed
- THEN the contents of `report.md` are used as the commit message
- AND `report.md` is deleted after the commit

#### Scenario: No report — changelog generated

- GIVEN the agent made file changes but did not write `report.md`
- WHEN changes are committed
- THEN the AI agent is called to generate a changelog and write `report.md`
- AND the generated contents are used as the commit message

#### Scenario: No changes

- GIVEN the agent made no file changes and `report.md` does not exist
- WHEN the commit step runs
- THEN no commit is made and the iteration loop continues

#### Scenario: Missing report after changes

- GIVEN the agent made changes but no `report.md` was produced and changelog generation fails
- WHEN the commit step runs
- THEN an error is returned and the loop exits

### Requirement: Completion Check

The system SHALL reload the project file after each commit and check whether all requirements are passing.

#### Scenario: All requirements pass

- GIVEN all requirements have `passing: true` after an iteration
- WHEN completion is checked
- THEN the loop exits successfully and newly-passing requirements are logged

#### Scenario: Requirements still failing

- GIVEN one or more requirements remain `passing: false`
- WHEN completion is checked
- THEN the loop continues to the next iteration

#### Scenario: Max iterations reached

- GIVEN the iteration count reaches the configured maximum before all requirements pass
- WHEN the loop exits
- THEN `ErrMaxIterationsReached` is returned with the count of still-failing requirements

### Requirement: Blocked File Detection

The system SHALL halt the iteration loop immediately if `blocked.md` exists at the repo root.

#### Scenario: Blocked before iteration

- GIVEN `blocked.md` is present at the start of an iteration
- WHEN the loop checks for the file
- THEN `ErrBlocked` is returned and no agent is invoked

#### Scenario: Blocked written by agent

- GIVEN the picker or development agent writes `blocked.md` due to an unresolvable error
- WHEN the next iteration begins
- THEN the loop detects the file and exits with `ErrBlocked`
