# Run --local Specification

## Purpose

Behavior of `ralph run --local`: runs the full development loop in-process on the local machine without submitting an Argo Workflow. This is the execution mode used inside workflow containers and for local development.

Each iteration works on exactly one [item](../../../../docs/glossary.md#item) of the project's resolved item array, and the branch's commit log — not the project file — records which items are done. See [Iterations](../../../../docs/iterations.md).

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
- THEN the AI agent generates a project file in `projects/` that implements the orchestration
- AND the generated project file is committed to the project branch
- AND execution proceeds using the generated project

#### Scenario: Orchestration and project generated and committed from spec

- GIVEN the input is a `spec.md` file
- AND the command has switched to the project branch
- WHEN just-in-time generation runs
- THEN the AI agent generates an `orchestration.md` file in the same directory as the spec
- AND the AI agent generates a project file in `projects/` that implements the spec and orchestration
- AND both generated files are committed to the project branch
- AND execution proceeds using the generated project

#### Scenario: Generated project resolves under the run's item query

- GIVEN the input is an `orchestration.md` or `spec.md` file
- WHEN the project file is generated
- THEN it is written in a shape that the run's resolved item query selects an item array from
- AND generation fails if the resolved query yields no non-empty items from the generated file

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

### Requirement: Item array resolved once per run

The command SHALL resolve the item array by evaluating the item query supplied by the caller (see [run/spec.md](../run/spec.md)) against the parsed project file, and SHALL do so exactly once, before the first iteration. Every iteration SHALL use that same resolved array, so an item's index means the same thing from the first iteration to the last.

Resolution discards empty outputs, so the resolved array is either empty or made entirely of non-empty items; see [write-project/spec.md](../write-project/spec.md). An empty resolved array SHALL abort the run before the first iteration, because a run with nothing to do MUST NOT reach the pull request step as though the project had completed.

#### Scenario: Query resolved before the first iteration

- GIVEN a project file and a resolved item query
- WHEN local execution reaches the iteration loop
- THEN the item array has already been resolved
- AND the same array is used for every iteration of the run

#### Scenario: Query yields no items

- GIVEN a project file against which the item query produces no output
- WHEN the item array is resolved
- THEN an error is returned: `item query yielded no items: <query>`
- AND no iteration runs

#### Scenario: Query yields only empty items

- GIVEN a project file whose item list holds nothing but nulls, blank strings, and empty mappings
- WHEN the item array is resolved
- THEN the resolved array is empty
- AND an error is returned: `item query yielded no items: <query>`
- AND no iteration runs and no pull request is opened

#### Scenario: Empty items dropped before indexing

- GIVEN a project file whose item list holds two work items with a null entry between them
- WHEN the item array is resolved
- THEN the loop iterates two items, indexed 0 and 1
- AND the completion trailers record those indices

#### Scenario: Query resolution does not depend on the file's shape

- GIVEN a project file whose items are plain strings, mappings, or nested structures
- WHEN the item array is resolved
- THEN each element becomes one item regardless of its shape
- AND no field of any item is required

---

### Requirement: Project file is not written during the loop

The command SHALL NOT write, normalize, reformat, or stage the project file at any point between the first and last iteration, and the AI agent SHALL be instructed not to edit it. The only permitted write to the project file is the optional cleanup deletion after every item is complete.

#### Scenario: Project file untouched across iterations

- GIVEN a run that completes several iterations
- WHEN the iterations finish
- THEN the project file's contents are byte-identical to what they were before the first iteration

#### Scenario: No normalization after an agent run

- GIVEN the AI agent has finished an iteration
- WHEN the iteration completes
- THEN no normalization or staging is applied to the project file

#### Scenario: Item indices stay stable

- GIVEN an item at index 3 in the first iteration
- WHEN the ninth iteration resolves incomplete items
- THEN index 3 still refers to that same item

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

### Requirement: Completion read from the commit log

At the start of every iteration the command SHALL determine which items are complete by reading the [completion trailers](../../../../docs/iterations.md#recording-completion) in the commit messages on the project branch that are not on the base branch, exactly as `ralph get complete` does (see [get/spec.md](../get/spec.md)). The project file SHALL NOT be consulted for completion state.

#### Scenario: Completion recomputed each iteration

- GIVEN an iteration is about to begin
- WHEN completion is determined
- THEN the branch's commit log is read and every completion trailer on it is collected
- AND items at those indices are treated as complete

#### Scenario: Trailer written in a previous iteration is honored

- GIVEN iteration 1 committed a message ending with `Ralph item 0 (csv-serializer) completed`
- WHEN iteration 2 determines completion
- THEN item 0 is complete and is not offered to the picker

#### Scenario: Out-of-range trailer ignored with a warning

- GIVEN a completion trailer whose index is outside the resolved item array
- WHEN completion is determined
- THEN a warning is emitted and the trailer is ignored
- AND the run continues

#### Scenario: Interrupted run resumes from the log

- GIVEN a run that was stopped after completing several items
- WHEN local execution is started again against the same branch and base
- THEN the previously completed items are read from the commit log
- AND the loop continues with the items that are left

---

### Requirement: Iteration loop

The iteration loop SHALL invoke the AI agent repeatedly until every item is complete or the iteration limit is reached. The iteration limit SHALL be the resolved item count plus the extra iteration count. When the extra iteration count is unset (nil), it SHALL default to 20% of the item count, rounded up. Each iteration checks for a blocked state before invoking the AI.

#### Scenario: All items already complete — exits after one iteration

- GIVEN a branch whose commit log already records every item complete
- WHEN the iteration loop runs
- THEN the loop exits after exactly 1 iteration without invoking the AI

#### Scenario: Items complete mid-loop — exits early

- GIVEN a project with 5 items and `--extra-iterations 3` (limit = 8)
- WHEN the last incomplete item is recorded complete during iteration 5
- THEN the loop exits after iteration 5
- AND does not consume additional iterations

#### Scenario: Default extra iterations is 20% when unset

- GIVEN neither `extraIterations` in config nor `--extra-iterations` flag is set
- AND the project resolves to 10 items
- WHEN the iteration loop starts
- THEN the iteration limit is 12 (10 items + 20% of 10)

#### Scenario: Default extra iterations rounds up

- GIVEN neither `extraIterations` in config nor `--extra-iterations` flag is set
- AND the project resolves to 3 items
- WHEN the iteration loop starts
- THEN the iteration limit is 4 (3 items + 20% of 3 rounded up from 0.6 to 1)

#### Scenario: Extra iterations exhausted with items incomplete

- GIVEN a project with 1 item and `--extra-iterations 0` (limit = 1)
- AND the item is still incomplete after iteration 1
- WHEN the iteration loop finishes
- THEN an error is returned indicating the iteration limit was reached
- AND the error message names the incomplete items

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

### Requirement: Item selection

Each iteration SHALL invoke a picker agent to select exactly one incomplete item. The picker SHALL receive the full project file, the incomplete items with their indices and keys, and the recent commit log, and SHALL choose based on dependencies between items, logical ordering, and impact. Selection SHALL NOT be constrained to array order.

#### Scenario: Picker chooses from incomplete items only

- GIVEN items 0 and 2 are recorded complete
- WHEN the picker runs
- THEN it is given only the remaining items, each with its index and key

#### Scenario: Picker receives the full project file

- GIVEN a project file with content outside the item array
- WHEN the picker runs
- THEN the whole file is included in the prompt as context

#### Scenario: Selection is not array order

- GIVEN incomplete items at indices 1, 2, and 3 where item 3 is a prerequisite of item 1
- WHEN the picker runs
- THEN it may select index 3 first

---

### Requirement: Item development

After selection, the command SHALL invoke the development agent with the selected item verbatim, its index, its key when it has one, and the full project file. The agent SHALL be instructed to write its commit message to `report.md` and, when the item is finished, to end that message with the completion trailer for the supplied index and key.

#### Scenario: Selected item passed verbatim

- GIVEN the picker selected the item at index 2
- WHEN the development agent is invoked
- THEN the item's value is included in the prompt exactly as it appears in the resolved array

#### Scenario: Index and key supplied to the agent

- GIVEN the selected item at index 2 has the key `export-endpoint`
- WHEN the development agent is invoked
- THEN the prompt supplies index 2 and key `export-endpoint`
- AND instructs the agent to end its commit message with `Ralph item 2 (export-endpoint) completed` when the item is done

#### Scenario: Item without a key

- GIVEN the selected item at index 2 is a plain string with no `slug`, `id`, or `name`
- WHEN the development agent is invoked
- THEN the prompt supplies the index-only trailer form `Ralph item 2 completed`

#### Scenario: Agent instructed not to edit the project file

- GIVEN the development agent is invoked for any item
- WHEN the prompt is built
- THEN it instructs the agent not to modify the project file

---

### Requirement: Commit after each iteration

After each iteration the command SHALL commit any changes the AI produced. The commit message comes from `report.md` if present; otherwise the AI generates a changelog. The command SHALL commit `report.md` verbatim and SHALL NOT append, rewrite, or remove a completion trailer.

#### Scenario: AI produces `report.md`

- GIVEN the AI wrote `report.md` during the iteration
- WHEN changes are committed
- THEN `report.md` is used as the commit message unmodified
- AND `report.md` is deleted after the commit

#### Scenario: Trailer preserved in the commit message

- GIVEN `report.md` ends with `Ralph item 1 (export-endpoint) completed`
- WHEN changes are committed
- THEN the commit message ends with that line
- AND the next iteration reads item 1 as complete

#### Scenario: Report without a trailer leaves the item incomplete

- GIVEN the AI wrote `report.md` with no completion trailer
- WHEN changes are committed
- THEN the commit is created and no item is marked complete
- AND the picker may select the same item again in a later iteration

#### Scenario: Changes without `report.md`

- GIVEN the working tree has uncommitted changes and no `report.md`
- WHEN changes are committed
- THEN the AI is called to generate a changelog, producing `report.md`
- AND that content is used as the commit message
- AND the generated changelog contains no completion trailer, so no item is marked complete

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

### Requirement: Orchestration cleanup before PR

Before submitting a pull request the command SHALL check whether the project's spec has an orchestration document, and if so, delete it and commit the deletion.

#### Scenario: Project has a spec with orchestration — orchestration removed

- GIVEN the project references a spec that contains an orchestration document
- WHEN all items are complete and the command is about to create a PR
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

### Requirement: Project file cleanup before PR

When cleanup is enabled by the caller (see [run/spec.md](../run/spec.md)) and every item is complete, the command SHALL delete the project file and commit the deletion on its own, before the pull request is opened. The cleanup commit SHALL carry no completion trailer and SHALL contain no other changes. Cleanup SHALL be skipped when it is not enabled.

#### Scenario: Project file deleted in its own commit

- GIVEN cleanup is enabled and every item is complete
- WHEN the command is about to create a PR
- THEN the project file is deleted and the deletion is committed as `chore: clean up completed project <path>`
- AND that commit contains no other file changes

#### Scenario: Cleanup commit carries no trailer

- GIVEN the cleanup commit is created
- WHEN the commit message is written
- THEN it contains no completion trailer
- AND the completion record already in the branch's history is unchanged

#### Scenario: Cleanup disabled leaves the file in place

- GIVEN cleanup is not enabled
- WHEN all items are complete
- THEN the project file is left in the repository and no cleanup commit is created

#### Scenario: Cleanup skipped when the run did not complete

- GIVEN cleanup is enabled
- AND the iteration loop ended with items still incomplete
- WHEN the loop exits
- THEN the project file is not deleted

#### Scenario: Completion still readable after cleanup

- GIVEN cleanup deleted the project file on this branch
- WHEN completion is read again from the branch
- THEN the trailers in the branch's history still report every item complete

---

### Requirement: Base branch for PR creation

The base branch used for PR creation SHALL be the value passed in by the caller, resolved according to [run/spec.md](../run/spec.md). The command SHALL NOT recompute or override this value. The same base branch SHALL bound the commit log that completion is read from.

#### Scenario: PR opened against the supplied base branch

- GIVEN the caller has resolved a base branch and passed it to run-local
- WHEN the PR creation step runs
- THEN the pull request base is the supplied base branch

#### Scenario: Completion read against the supplied base branch

- GIVEN the caller supplied `develop` as the base branch
- WHEN completion is read at the start of an iteration
- THEN only commits on the project branch that are not on `develop` are scanned for trailers

---

### Requirement: PR creation when all items are complete

When every item is found to be complete — whether the branch already recorded them all before the first iteration or they were recorded during the loop — the command SHALL generate an AI PR summary from the branch's commit log and open a GitHub pull request from the project branch to the base branch.

#### Scenario: All items complete after iterations — commits exist

- GIVEN items are recorded complete during the iteration loop
- AND the project branch has commits not on the base branch
- WHEN the PR creation step runs
- THEN a pull request is created
- AND the PR title is the project's `title` field, falling back to its slug

#### Scenario: All items already complete at start — commits exist

- GIVEN the branch already records every item complete before any iteration runs
- AND the project branch has commits not on the base branch
- WHEN the PR creation step runs
- THEN a pull request is created

#### Scenario: No commits ahead of base branch

- GIVEN all items are complete
- AND no commits were added to the project branch
- WHEN the PR creation step runs
- THEN PR creation is skipped
- AND the command exits successfully

#### Scenario: Iteration limit reached with items incomplete — PR skipped

- GIVEN the iteration loop exits because the iteration limit was reached
- AND one or more items are still incomplete
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
- AND the run exits with an error (iteration limit reached, blocked, fatal AI error, or any other failure)
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
