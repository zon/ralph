# Loop Command Specification

## Purpose

The `loop` command runs a bounded AI iteration loop over a list of steps. Given a slug, Ralph looks up the matching loop config in the `loops:` section of `.ralph/config.yaml`, embeds that config's `steps` in a prompt, and runs the prompt until the agent reports nothing left to do or the `--max` cap is reached. Before iterating, Ralph switches to the `loop-<slug>` branch, creating it from the current branch when it does not exist, so the agent works on the loop branch's own state. Every iteration that does real work is committed and pushed to `loop-<slug>`. When the loop ends with commits on that branch, Ralph opens a pull request. Steps can also be supplied directly with `--step` flags, which replace the config's `steps`. When steps are supplied without a slug, Ralph asks the AI to read the steps and propose a slug. Execution runs in one of three modes selected with `--mode`: `local` (default), which runs the loop in-process on the local machine, `worktree`, which runs it in-process in a local Git worktree, or `remote`, which submits an Argo Workflow to Kubernetes.

Mode-specific behaviors are defined in:
- [run-local.md](run-local.md) — `local` mode: runs the loop in-process in the current checkout
- [run-worktree.md](run-worktree.md) — `worktree` mode: runs the loop in-process in a local Git worktree
- [run-remote.md](run-remote.md) — `remote` mode: submits an Argo Workflow to Kubernetes, with the same remote defaults and config as `ralph run`

## Requirements

### Requirement: Execution mode selection

The command SHALL accept `--mode` to select the execution mode. The option SHALL accept exactly one of `local`, `worktree`, or `remote`.

Mode resolution follows a three-level precedence: `--mode` at the command line takes priority; otherwise the top-level `mode` field in `.ralph/config.yaml` is used; otherwise the mode defaults to `local`.

- `local` runs the loop in-process in the current checkout.
- `worktree` runs the loop in-process in a Git worktree created for the `loop-<slug>` branch, leaving the current checkout untouched; see [run-worktree.md](run-worktree.md).
- `remote` submits an Argo Workflow to Kubernetes, and the loop runs inside the workflow container; see [run-remote.md](run-remote.md).

The `--follow` flag is workflow-only and is rejected for `local` and `worktree` modes; see [Incompatible flags are rejected](#requirement-incompatible-flags-are-rejected).

The loop body (slug and step resolution, prompt construction, iteration, commit and push, and pull request opening) SHALL behave identically across all three modes.

#### Scenario: `--mode local` runs in-process

- GIVEN the user passes `--mode local`
- WHEN the command starts
- THEN the loop runs in-process on the local machine
- AND no workflow is submitted

#### Scenario: `--mode worktree` runs in a Git worktree

- GIVEN the user passes `--mode worktree`
- WHEN the command starts
- THEN a Git worktree is created for the `loop-<slug>` branch
- AND the loop runs in-process inside that worktree

#### Scenario: `--mode remote` submits a workflow

- GIVEN the user passes `--mode remote`
- WHEN the command starts
- THEN an Argo Workflow is submitted to Kubernetes
- AND the loop runs inside the workflow container

#### Scenario: Default mode runs locally

- GIVEN neither `--mode` nor `mode` in `.ralph/config.yaml` is set
- WHEN the command starts
- THEN execution runs in `local` mode

#### Scenario: Config mode used when no flag is passed

- GIVEN `mode: local` is set in `.ralph/config.yaml`
- AND no `--mode` flag is passed
- WHEN the command starts
- THEN execution runs in `local` mode

#### Scenario: `--mode` overrides the configured mode

- GIVEN `mode: remote` is set in `.ralph/config.yaml`
- AND the user passes `--mode worktree`
- WHEN the command starts
- THEN execution runs in `worktree` mode instead of the config value

#### Scenario: Loop body runs in all three modes

- GIVEN the mode resolves to `local`, `worktree`, or `remote`
- WHEN the loop executes
- THEN the requirements below for slug and step resolution, prompt construction, iteration, commit and push, and pull request opening apply unchanged

---

### Requirement: AI model and variant overrides

The command SHALL accept `--model` and `--variant` to override the AI model and its provider-specific reasoning-effort variant from config, per the shared [model-options.md](model-options.md) contract. The overrides SHALL apply to every prompt the loop runs: the slug proposal and each iteration prompt. For `remote` mode, the same resolution SHALL be carried into the container.

#### Scenario: Overrides reach the remote container

- GIVEN the user passes `--model gpt-4 --variant high`
- WHEN a remote workflow is submitted
- THEN the container runs the loop with `--model gpt-4 --variant high`

---

### Requirement: Incompatible flags are rejected

The command SHALL reject flag combinations that have no valid meaning before any execution begins.

#### Scenario: `--follow` with `--mode local`

- GIVEN the user passes both `--follow` and `--mode local`
- WHEN the command validates flag combinations
- THEN an error is returned: `--follow flag is not applicable with --mode local`

#### Scenario: `--follow` with `--mode worktree`

- GIVEN the user passes both `--follow` and `--mode worktree`
- WHEN the command validates flag combinations
- THEN an error is returned: `--follow flag is not applicable with --mode worktree`

---

### Requirement: Remote behavior matches `ralph run`

The command SHALL reuse the remote defaults and config of `ralph run` as defined in [run.md](run.md) and [run-remote.md](run-remote.md). The resolved model and variant, Kubernetes context, base branch, branch-sync check, Ralph-owned workflow label, and notification behavior SHALL be the same as for `ralph run`.

`--context` SHALL override the Kubernetes context used for workflow submission, per the shared [kube-options.md](kube-options.md) contract. Model and variant resolution SHALL follow the shared contract in [model-options.md](model-options.md). Before submission the command SHALL verify, exactly as `ralph run` does, that the current branch exists on the remote and that local and remote are at the same commit.

#### Scenario: `--context` overrides the Kubernetes context

- GIVEN the user passes `--context my-cluster`
- WHEN a remote workflow is submitted
- THEN `my-cluster` is used as the Kubernetes context instead of the default

#### Scenario: Branch must be in sync with remote before submission

- GIVEN the current branch has no remote tracking ref, or local and remote differ at the current commit
- WHEN the command checks branch sync
- THEN an error is returned
- AND no workflow is submitted

#### Scenario: Workflow carries the slug and steps into the container

- GIVEN a loop with slug `fmt` and resolved steps
- WHEN the workflow YAML is generated
- THEN the container runs the loop with the slug `fmt` and the resolved steps

---

### Requirement: `--follow` streams logs after submission

The command SHALL accept `--follow`. With `--follow`, the command SHALL stream the workflow logs and wait for the workflow to finish before returning. Without `--follow`, the command SHALL print the `argo logs` command the user can run to follow the workflow and return after submission. Notification behavior on followed workflows SHALL match [run-remote.md](run-remote.md).

#### Scenario: Log hint printed after submission

- GIVEN a workflow is submitted without `--follow`
- WHEN the workflow name is printed
- THEN Ralph also prints the `argo logs` command the user can run to follow the workflow
- AND the command returns without waiting

#### Scenario: `--follow` waits for completion

- GIVEN the user passes `--follow`
- AND the workflow is submitted successfully
- WHEN the workflow runs
- THEN Ralph streams the Argo workflow logs and blocks until the workflow finishes

---

### Requirement: Slug or steps required

The command SHALL accept an optional positional slug argument and any number of `--step` flags. Exactly one of a slug argument or at least one `--step` flag SHALL be provided. When neither is given, the command SHALL return an error with usage before any execution begins.

#### Scenario: Slug argument provided

- GIVEN the user runs `ralph loop fmt`
- WHEN the command validates its input
- THEN the slug is `fmt` and execution proceeds

#### Scenario: Steps provided without a slug

- GIVEN the user runs `ralph loop --step "run gofmt" --step "run go vet"`
- WHEN the command validates its input
- THEN execution proceeds with steps `run gofmt` and `run go vet`

#### Scenario: Neither slug nor steps provided

- GIVEN the user runs `ralph loop` with no slug argument and no `--step` flags
- WHEN the command validates its input
- THEN an error is returned indicating that a slug or at least one `--step` is required
- AND no execution begins

---

### Requirement: Loop config resolution

When a slug argument is provided, the command SHALL load `.ralph/config.yaml` and find the entry in its `loops:` section whose `slug` matches the argument. The matching entry's `steps` SHALL be the resolved steps, unless `--step` flags override them (see [Steps override](#requirement-steps-override)). When no entry matches the slug, the command SHALL return an error and no execution begins.

#### Scenario: Matching loop config found

- GIVEN `.ralph/config.yaml` has a `loops:` entry with `slug: fmt` and two steps
- WHEN the user runs `ralph loop fmt`
- THEN the entry's `steps` are resolved for the prompt
- AND the branch is derived from the slug `fmt`

#### Scenario: Loop config not found

- GIVEN `.ralph/config.yaml` has no `loops:` entry whose `slug` is `fmt`
- WHEN the user runs `ralph loop fmt`
- THEN an error is returned: `loop config not found: fmt`
- AND no prompt runs

#### Scenario: Slug lookup still required when steps override

- GIVEN the user runs `ralph loop fmt --step "run tests"`
- AND no `loops:` entry has `slug: fmt`
- WHEN the command resolves the loop config
- THEN an error is returned: `loop config not found: fmt`
- AND no prompt runs

---

### Requirement: Steps override

The command SHALL accept one or more `--step` flags. When at least one `--step` flag is passed, the resolved steps SHALL be the flag values, in the order given, replacing the loop config's `steps` property entirely.

#### Scenario: Single `--step` replaces config steps

- GIVEN a `loops:` entry with `slug: fmt` whose `steps` list has two entries
- AND the user runs `ralph loop fmt --step "run tests"`
- WHEN the prompt is built
- THEN the prompt embeds only the step `run tests`

#### Scenario: Multiple `--step` flags in order

- GIVEN the user runs `ralph loop --step "step one" --step "step two" --step "step three"`
- WHEN the prompt is built
- THEN the prompt embeds the three steps in the order given

#### Scenario: No `--step` flags uses config steps

- GIVEN a `loops:` entry with `slug: fmt` whose `steps` list has two entries
- AND the user runs `ralph loop fmt` with no `--step` flags
- WHEN the prompt is built
- THEN the prompt embeds the config's two steps

---

### Requirement: Slug derived from steps

When `--step` flags are provided without a slug argument, the command SHALL invoke the AI with the steps and ask it to read them and propose a slug. The proposed slug SHALL be used to derive the branch name, and no loop config lookup SHALL occur.

#### Scenario: Slug proposed from steps

- GIVEN the user runs `ralph loop --step "run gofmt" --step "run go vet"`
- WHEN the AI is asked to propose a slug
- THEN the AI reads the steps and proposes a slug, e.g. `format-code`
- AND the branch is derived from that proposed slug

#### Scenario: AI returns no usable slug

- GIVEN the AI is asked to propose a slug from the steps
- AND the AI returns an empty or blank proposal
- WHEN the slug proposal is processed
- THEN an error is returned
- AND no prompt runs and no branch is created

---

### Requirement: Prompt construction

The command SHALL build a prompt that embeds the resolved steps, in order. The prompt SHALL instruct the AI to follow the steps and to write a brief and simple summary of what was done to `report.md`. The prompt SHALL instruct the AI that the summary describes only what was done in response to the steps and SHALL NOT restate the steps themselves. The prompt SHALL also instruct the AI that when nothing was necessary, it MUST write exactly the constant string `NOTHING_TO_DO` to `report.md` instead of a summary.

#### Scenario: Config steps embedded

- GIVEN a `loops:` entry with `slug: fmt` and steps `step one` and `step two`
- WHEN the prompt is built
- THEN the prompt embeds both steps in order

#### Scenario: Steps from flags embedded

- GIVEN the user passes `--step "run tests"`
- WHEN the prompt is built
- THEN the prompt embeds the step `run tests`

#### Scenario: Prompt requires a `report.md` summary

- GIVEN the prompt is built
- WHEN the AI reads it
- THEN the prompt instructs the AI to write a brief and simple summary of what it did to `report.md`

#### Scenario: Prompt summary excludes the loop steps

- GIVEN the prompt is built
- WHEN the AI reads it
- THEN the prompt instructs the AI that the summary describes only what was done in response
- AND the summary does not restate the loop steps

#### Scenario: Prompt names the nothing-to-do constant

- GIVEN the prompt is built
- WHEN the AI reads it
- THEN the prompt instructs the AI to write exactly `NOTHING_TO_DO` to `report.md` when nothing was necessary

---

### Requirement: Iteration loop

Before running the prompt, the command SHALL switch to the branch `loop-<slug>`, creating it from the current branch when it does not already exist. The command SHALL then run the prompt repeatedly as an iteration loop. Each iteration SHALL invoke the AI with the prompt and then read `report.md`. The loop SHALL stop when the report content equals the constant string `NOTHING_TO_DO` (trimmed of surrounding whitespace) or when the number of iterations reaches the `--max` cap, whichever comes first. The `--max` flag SHALL default to `20` and SHALL be a positive integer.

#### Scenario: Switches to the loop branch before the first iteration

- GIVEN no `loop-<slug>` branch exists and the current branch is `main`
- WHEN the loop starts
- THEN the branch `loop-<slug>` is created from `main` and checked out
- AND the prompt runs on the `loop-<slug>` branch

#### Scenario: Switches to an existing loop branch

- GIVEN a `loop-<slug>` branch already exists
- WHEN the loop starts
- THEN the `loop-<slug>` branch is checked out
- AND the prompt runs on the `loop-<slug>` branch's own state

#### Scenario: Stops on the nothing-to-do report

- GIVEN the AI writes exactly `NOTHING_TO_DO` to `report.md` during iteration 3
- WHEN the report is read
- THEN the loop stops immediately
- AND no further iteration runs

#### Scenario: Stops at the `--max` cap

- GIVEN `--max 3` and the AI never writes `NOTHING_TO_DO`
- WHEN the loop reaches the third iteration
- THEN the loop stops after iteration 3
- AND no fourth iteration runs

#### Scenario: Default `--max` is 20

- GIVEN no `--max` flag is passed
- AND the AI never writes `NOTHING_TO_DO`
- WHEN the loop runs
- THEN the loop runs at most 20 iterations

#### Scenario: Custom `--max` overrides the default

- GIVEN the user passes `--max 5`
- WHEN the loop runs
- THEN the loop runs at most 5 iterations

#### Scenario: Non-positive `--max` rejected

- GIVEN the user passes `--max 0`
- WHEN the command validates the flag
- THEN an error is returned
- AND no loop runs

---

### Requirement: Commit and push each iteration

After each iteration whose report is not the nothing-to-do constant, the command SHALL commit the AI's changes and push them to the branch `loop-<slug>`. The commit message SHALL be the report content. `report.md` SHALL be deleted after the commit. An iteration whose report equals `NOTHING_TO_DO` SHALL NOT be committed.

#### Scenario: Changes committed and pushed

- GIVEN the AI made changes and wrote a summary to `report.md`
- WHEN the iteration completes
- THEN the changes are committed with the report content as the message
- AND the commit is pushed to `loop-<slug>`

#### Scenario: Commits land on the checked-out loop branch

- GIVEN the loop switched to `loop-<slug>` before iterating
- AND the AI made changes on that branch and wrote a summary to `report.md`
- WHEN the iteration completes
- THEN the changes are committed on `loop-<slug>`
- AND the working tree stays clean after the commit

#### Scenario: `report.md` deleted after the commit

- GIVEN the AI wrote a summary to `report.md`
- WHEN the changes are committed
- THEN `report.md` is removed from the working tree after the commit

#### Scenario: Nothing-to-do iteration is not committed

- GIVEN the AI wrote exactly `NOTHING_TO_DO` to `report.md`
- WHEN the iteration completes
- THEN no commit is created and nothing is pushed for that iteration

---

### Requirement: Pull request on completion

When the loop ends, the command SHALL open a pull request from the branch `loop-<slug>` to the branch it was created from, if and only if at least one commit was made on `loop-<slug>`. The pull request body SHALL be an AI-generated summary of the commit log, generated the same way `ralph run` generates its PR summary. When no commits were made, the command SHALL NOT open a pull request and SHALL exit successfully.

#### Scenario: Pull request opened when commits exist

- GIVEN the loop ended after committing changes on `loop-<slug>`
- WHEN the loop finishes
- THEN a pull request is opened from `loop-<slug>` to the branch it was created from

#### Scenario: Pull request body is an AI-generated summary

- GIVEN the loop ended after committing changes on `loop-<slug>`
- WHEN a pull request is opened
- THEN the pull request body is an AI-generated summary of the commits on `loop-<slug>`

#### Scenario: No pull request when nothing was committed

- GIVEN the loop stopped on the nothing-to-do report in its first iteration
- AND no commit was made on `loop-<slug>`
- WHEN the loop finishes
- THEN no pull request is opened
- AND the command exits successfully

---

### Requirement: Token usage and cost reporting

When running inside a workflow container the command SHALL print accumulated AI token usage and cost statistics at the end of execution, regardless of whether the loop succeeded or failed, matching the behavior of `ralph run` described in [run-local.md](run-local.md).

#### Scenario: Stats reported on successful workflow loop

- GIVEN Ralph is executing inside a workflow container
- AND the loop completes successfully
- WHEN execution finishes
- THEN input tokens, output tokens, and total cost across the entire loop are printed to the log

#### Scenario: Stats reported on failed workflow loop

- GIVEN Ralph is executing inside a workflow container
- AND the loop exits with an error (iteration limit reached, fatal AI error, or any other failure)
- WHEN execution finishes
- THEN input tokens, output tokens, and total cost across the entire loop are printed to the log before the error is surfaced

#### Scenario: Stats not printed outside a workflow

- GIVEN Ralph is executing locally (not inside a workflow container)
- WHEN the loop completes or fails
- THEN no token usage or cost statistics are printed
