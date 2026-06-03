# Workflow Command Specification

## Purpose

`ralph workflow command` is the container entrypoint for arbitrary command execution: clone the current branch and run the supplied command tokens in the ralph container environment.

## Requirements

### Requirement: Workspace Setup

The system SHALL prepare the container workspace as defined in [workflow-workspace/spec.md](../workflow-workspace/spec.md) before running the command.

### Requirement: Command Execution

The system SHALL execute the supplied command tokens after the repository is cloned.

#### Scenario: Command run

- GIVEN one or more command tokens are provided to `ralph workflow command`
- WHEN the container runs
- THEN the supplied tokens are executed as a command in the cloned repository

#### Scenario: Missing command tokens

- GIVEN no command tokens are provided to `ralph workflow command`
- WHEN the command starts
- THEN an error is returned before any work is done

