# Run --local Specification

## Purpose

Behavior of `ralph run --local`: runs the full development loop in-process on the local machine without submitting an Argo Workflow. This is the execution mode used inside workflow containers and for local development.

## Requirements

### Requirement: Pre-execution setup

Before starting the iteration loop, the command SHALL run any configured `before` commands and switch to the project branch.

#### Scenario: `before` commands run first

- GIVEN `.ralph/config.yaml` contains `before:` commands
- WHEN local execution begins
- THEN each `before` command is run before the iteration loop starts

#### Scenario: Branch switched before iteration

- GIVEN the project slug is `my-feature` and the current branch is `main`
- WHEN local execution starts
- THEN ralph switches to (or creates) the branch `my-feature` before running any iterations

---

### Requirement: Just-in-time artifact generation

When the input is an orchestration or spec document rather than a project file, the command SHALL use the AI agent to generate the missing artifacts and commit them after switching to the project branch, so that the generation commits and the coding work share the same branch.

When the input is an **orchestration document**, the command generates a project file and commits it, then proceeds using the generated project.

When the input is a **spec document**, the command generates an orchestration document in the same directory as the spec, then generates a project file, commits both, and proceeds using the generated project.

#### Scenario: Project generated and committed from orchestration

- GIVEN the input is an `orchestration.md` file
- AND the command has switched to the project branch
- WHEN just-in-time generation runs
- THEN the AI agent generates a project YAML file in `projects/` that implements the orchestration
- AND the generated project file is committed to the project branch
- AND execution proceeds using the generated project

#### Scenario: Orchestration and project generated and committed from spec

- GIVEN the input is a `spec.md` file
- AND the command has switched to the project branch
- WHEN just-in-time generation runs
- THEN the AI agent generates an `orchestration.md` file in the same directory as the spec
- AND the AI agent generates a project YAML file in `projects/` that implements the spec and orchestration
- AND both generated files are committed to the project branch
- AND execution proceeds using the generated project

#### Scenario: Project generation failure from orchestration aborts run

- GIVEN the input is an `orchestration.md` file
- AND the AI agent fails to generate a valid project
- WHEN the generation step runs
- THEN an error is returned and no further execution begins

#### Scenario: Orchestration generation failure from spec aborts run

- GIVEN the input is a `spec.md` file
- AND the AI agent fails to generate an orchestration
- WHEN the orchestration generation step runs
- THEN an error is returned and no further execution begins

#### Scenario: Project generation failure from spec aborts run after orchestration succeeds

- GIVEN the input is a `spec.md` file
- AND the orchestration is generated and committed successfully
- AND the AI agent fails to generate a valid project
- WHEN the project generation step runs
- THEN an error is returned and no further execution begins

---

### Requirement: Per-Iteration Service Management

Before each iteration the system SHALL start configured services and stop them after the iteration completes.

#### Scenario: Services started before each iteration

- GIVEN services are configured in `.ralph/config.yaml` and `--no-services` is not set
- WHEN an iteration begins
- THEN all services are started before the picker and development agents run
- AND services are stopped when the iteration completes

#### Scenario: Service startup failure triggers AI fix

- GIVEN a configured service fails to start at the start of an iteration
- WHEN the failure is detected
- THEN the development agent is invoked with a prompt to diagnose and fix the startup failure
- AND the iteration proceeds after the fix attempt

#### Scenario: Port health check

- GIVEN a service has a `port` field configured
- WHEN the service starts during an iteration
- THEN ralph waits for a TCP connection to that port to succeed before proceeding

---

### Requirement: Iteration loop

The iteration loop SHALL invoke the AI agent repeatedly until all requirements pass or the iteration limit is reached. Each iteration checks for a blocked state before invoking the AI.

#### Scenario: All requirements already passing — exits after one iteration

- GIVEN a project where every requirement has `passing: true`
- WHEN the iteration loop runs
- THEN the loop exits after exactly 1 iteration without invoking the AI

#### Scenario: Requirements pass mid-loop — exits early

- GIVEN a project with failing requirements and max iterations = 10
- WHEN the AI marks all requirements as passing during iteration 3
- THEN the loop exits after iteration 3
- AND does not consume additional iterations

#### Scenario: Max iterations reached with failures remaining

- GIVEN max iterations is 1 and requirements are still failing after iteration 1
- WHEN the iteration loop finishes
- THEN an error is returned indicating max iterations were reached
- AND the number of still-failing requirements is included in the error message

#### Scenario: `blocked.md` detected at loop start

- GIVEN `blocked.md` exists in the repository root at the start of an iteration
- WHEN the loop checks for the blocked state
- THEN the loop stops immediately with a blocked error
- AND the AI is not invoked

#### Scenario: Fatal AI error (billing/quota)

- GIVEN the AI agent returns an error containing a billing or quota keyword
- WHEN the iteration processes the error
- THEN a fatal error is returned
- AND the loop does not retry

---

### Requirement: Commit after each iteration

After each iteration the command SHALL commit any changes the AI produced. The commit message comes from `report.md` if present; otherwise the AI generates a changelog.

#### Scenario: AI produces `report.md`

- GIVEN the AI wrote `report.md` during the iteration
- WHEN changes are committed
- THEN `report.md` is used as the commit message
- AND `report.md` is deleted after the commit

#### Scenario: Changes without `report.md`

- GIVEN the working tree has uncommitted changes and no `report.md`
- WHEN changes are committed
- THEN the AI is called to generate a changelog, producing `report.md`
- AND that content is used as the commit message

#### Scenario: No changes and no `report.md`

- GIVEN the working tree is clean and no `report.md` exists
- WHEN the commit step runs
- THEN no commit is created and no error is returned

#### Scenario: `blocked.md` written on AI agent failure

- GIVEN the AI agent exits with a non-fatal error
- WHEN the iteration processes the failure
- THEN `blocked.md` is written to the repository root containing the failure reason
- AND subsequent iterations detect it and stop

---

### Requirement: Post-Agent Cleanup

After each agent run the command SHALL normalize the project file.

#### Scenario: Project file normalized after agent run

- GIVEN the agent may have added trailing newlines to the project file
- WHEN the agent finishes
- THEN excess trailing newlines are stripped from the project file
- AND the project file is staged if it has changes

---

### Requirement: Orchestration cleanup before PR

Before submitting a pull request the command SHALL check whether the project's spec has an orchestration document, and if so, delete it and commit the deletion.

#### Scenario: Project has a spec with orchestration — orchestration removed

- GIVEN the project references a spec that contains an orchestration document
- WHEN all requirements are passing and the command is about to create a PR
- THEN the orchestration document is deleted from the repository
- AND the deletion is committed before the pull request is opened

#### Scenario: Project has no spec — cleanup skipped

- GIVEN the project does not reference a spec
- WHEN the command is about to create a PR
- THEN no orchestration cleanup is performed

#### Scenario: Project spec has no orchestration — cleanup skipped

- GIVEN the project references a spec that does not contain an orchestration document
- WHEN the command is about to create a PR
- THEN no orchestration cleanup is performed

---

### Requirement: PR creation when all requirements pass

When all requirements are found to be passing — whether they were already passing before the first iteration or became passing during the loop — the command SHALL generate an AI PR summary and open a GitHub pull request from the project branch to the base branch.

#### Scenario: All requirements pass after iterations — commits exist

- GIVEN requirements become passing during the iteration loop
- AND the project branch has commits not on the base branch
- WHEN the PR creation step runs
- THEN a pull request is created
- AND the PR title matches the project title

#### Scenario: All requirements already passing at start — commits exist

- GIVEN all requirements are passing before any iteration runs
- AND the project branch has commits not on the base branch
- WHEN the PR creation step runs
- THEN a pull request is created

#### Scenario: No commits ahead of base branch

- GIVEN all requirements are passing
- AND no commits were added to the project branch
- WHEN the PR creation step runs
- THEN PR creation is skipped
- AND the command exits successfully

#### Scenario: Max iterations reached with failing requirements — PR skipped

- GIVEN the iteration loop exits because max iterations were reached
- AND one or more requirements are still failing
- WHEN the loop ends
- THEN PR creation is skipped
- AND an error is returned

---

### Requirement: Token usage and cost reporting

When running inside a workflow container the command SHALL print accumulated AI token usage and cost statistics at the end of execution, regardless of whether the run succeeded or failed.

#### Scenario: Stats reported on successful workflow run

- GIVEN ralph is executing inside a workflow container
- AND the run completes successfully
- WHEN execution finishes
- THEN input tokens, output tokens, and total cost across the entire run are printed to the log

#### Scenario: Stats reported on failed workflow run

- GIVEN ralph is executing inside a workflow container
- AND the run exits with an error (max iterations, blocked, fatal AI error, or any other failure)
- WHEN execution finishes
- THEN input tokens, output tokens, and total cost across the entire run are printed to the log before the error is surfaced

#### Scenario: Stats not printed outside a workflow

- GIVEN ralph is executing locally (not inside a workflow container)
- WHEN the run completes or fails
- THEN no token usage or cost statistics are printed

---

### Requirement: Desktop notifications

The command SHALL send a desktop notification on completion unless `--no-notify` is set.

#### Scenario: Notification on success

- GIVEN `--no-notify` is not set
- WHEN the run completes successfully
- THEN a success desktop notification is sent for the project slug

#### Scenario: Notification on failure

- GIVEN `--no-notify` is not set
- WHEN the iteration loop fails
- THEN an error desktop notification is sent for the project slug

#### Scenario: Notifications suppressed

- GIVEN `--no-notify` is set
- WHEN the run completes or fails
- THEN no desktop notification is sent
