# Workflow Run Specification

## Purpose

`ralph workflow run` executes the project loop after the workspace is ready: synchronize the base branch, then delegate to the run-local behavior.

## Requirements

### Requirement: Workspace Setup

The system SHALL prepare the container workspace as defined in [workflow-workspace/spec.md](../workflow-workspace/spec.md) before doing any work.

### Requirement: Execution Parameters

The system SHALL apply run-specific flags to the execution context before starting the project loop.

#### Scenario: Instructions injection

- GIVEN `--instructions-md` is provided with inline instructions content
- WHEN `ralph workflow run` starts
- THEN the instructions are passed into the execution context for the AI agent

#### Scenario: Extra iterations

- GIVEN `--extra-iterations` is provided
- WHEN the project loop executes
- THEN the provided extra iteration count is passed to the run-local behavior
- AND when `--extra-iterations` is absent, no extra iteration value is passed and run-local applies its default

#### Scenario: Item query

- GIVEN `--items` is provided with the query resolved at submission time
- WHEN the project loop executes
- THEN that query is used to resolve the item array
- AND the `items` field in `.ralph/config.yaml` is not consulted

#### Scenario: Cleanup

- GIVEN `--cleanup` is provided
- WHEN every item is complete
- THEN the project file is deleted and the deletion committed on its own before the pull request is opened
- AND when `--cleanup` is absent, the project file is left in place

#### Scenario: Model override

- GIVEN `--model` is provided
- WHEN `ralph workflow run` starts
- THEN the specified model is used instead of the model in `.ralph/config.yaml`

#### Scenario: Agent override

- GIVEN `--agent` is provided
- WHEN `ralph workflow run` starts
- THEN the specified opencode agent is used for the prompts that write code instead of the agent in `.ralph/config.yaml`

#### Scenario: Service startup skipped

- GIVEN `--no-services` is set
- WHEN `ralph workflow run` starts
- THEN dependent service startup is skipped before executing the project

#### Scenario: Base branch bounds the completion log

- GIVEN `--base develop` is provided
- WHEN `ralph workflow run` starts
- THEN the run-local behavior bounds the commit log completion is read from by `develop`
- AND completion is never read from commits already on `develop`

### Requirement: Input and Configuration Validation

The system SHALL validate all required inputs and load configuration before performing any base-branch synchronization or AI agent invocations.

#### Scenario: Missing project path

- GIVEN no project path argument is provided to `ralph workflow run`
- WHEN the command starts
- THEN an error is returned before any work is done

#### Scenario: Missing config proceeds with defaults

- GIVEN `.ralph/config.yaml` does not exist in the repository
- WHEN validation runs after the workspace is ready
- THEN execution continues with default values for all config-backed settings

#### Scenario: Malformed config fails

- GIVEN `.ralph/config.yaml` exists but cannot be parsed
- WHEN validation runs after the workspace is ready
- THEN an error is returned before base-branch synchronization begins

#### Scenario: Project file load failure

- GIVEN the project file at the provided path is missing, does not parse, or yields no non-empty items under the supplied item query
- WHEN validation runs after the workspace is ready
- THEN an error is returned before base-branch synchronization begins

### Requirement: Base Branch Synchronization

The system SHALL attempt to merge the base branch into the project branch before running the project.

#### Scenario: Branch is up-to-date

- GIVEN the project branch's merge-base with the base branch equals the base branch tip
- WHEN synchronization runs
- THEN no merge is performed and execution continues

#### Scenario: Clean merge

- GIVEN the project branch is behind the base branch with no conflicts
- WHEN synchronization runs
- THEN the base branch is merged into the project branch (fast-forward or auto-merge)
- AND execution continues

#### Scenario: Merge conflicts resolved by AI

- GIVEN a merge attempt produces conflicts
- WHEN the merge fails
- THEN the merge is aborted
- AND an AI agent is invoked with instructions to resolve all conflicts, run tests, and stage the resolved files
- AND the invocation receives the configured agent, because resolving conflicts writes repository code
- AND execution continues after resolution

#### Scenario: Base branch fetch failure

- GIVEN the base branch cannot be fetched (e.g., network error)
- WHEN synchronization runs
- THEN a warning is logged and execution continues without merging

### Requirement: Debug Mode

The system SHOULD support a debug mode that clones a specific ralph branch and invokes ralph via `go run` instead of the built binary.

#### Scenario: Debug branch set

- GIVEN `--debug <branch>` is provided
- WHEN `ralph workflow run` starts
- THEN the specified ralph source branch is checked out into `/workspace/ralph`
- AND a wrapper script is written to `/usr/local/bin/ralph` that invokes `go run ./cmd/ralph/main.go` from the cloned source
- AND subsequent ralph invocations use that wrapper instead of the installed binary

### Requirement: Project Execution

After base-branch synchronization, the system SHALL execute the project by invoking the run-local behavior defined in [run-local/spec.md](../run-local/spec.md).
