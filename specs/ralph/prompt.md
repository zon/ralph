# Prompt Command Specification

## Purpose

The `prompt` command runs one user-supplied prompt as a single non-interactive OpenCode pass. It is the companion of `run` and `loop` for work that comes from neither a project file nor a step list. Ralph asks the AI for a branch slug, runs the pass on the prompt branch, renames the branch from the work it did, and opens a pull request when the pass leaves commits. It has no project items, iteration cap, report contract, completion trailer, or iteration recovery.

Execution runs in one of three modes selected with `--mode`: `local` (default), `worktree`, or `remote`. The pass body is the same in every mode. The mode decides where it runs and how the result reaches the remote branch. Mode resolution, base branch synchronization, services, and workflow submission follow the shared contracts linked below.

## Requirements

### Requirement: Prompt input

The command SHALL accept the prompt from exactly one source:

- positional text,
- a file named by `--file <path>`, or
- standard input when the command has neither positional text nor `--file` and standard input is not a terminal.

Positional tokens SHALL be joined with single spaces. `--file -` SHALL read standard input even when standard input is a terminal. Using positional text and `--file` together SHALL fail. A missing, unreadable, empty, or whitespace-only prompt SHALL fail before any mode work starts. A prompt that begins with `-` SHALL require `--` before it.

#### Scenario: Positional text becomes the prompt

- GIVEN the user passes `ralph prompt update the readme`
- WHEN the command starts
- THEN the prompt is `update the readme`
- AND the pass receives that text unchanged

#### Scenario: Prompt read from a file

- GIVEN a readable file named `prompt.txt`
- WHEN the user passes `ralph prompt --file prompt.txt`
- THEN the file contents become the prompt
- AND standard input is not read

#### Scenario: Prompt read from standard input

- GIVEN the user passes no positional text and no `--file`
- AND standard input is not a terminal
- WHEN text is piped to `ralph prompt`
- THEN the piped text becomes the prompt

#### Scenario: Explicit standard input from a terminal

- GIVEN standard input is a terminal
- WHEN the user passes `ralph prompt --file -`
- THEN the command reads the prompt from standard input

#### Scenario: Positional text and file together fail

- GIVEN the user passes positional text and `--file`
- WHEN the command starts
- THEN the command fails with an error
- AND no mode work starts

#### Scenario: Missing prompt fails

- GIVEN the user passes no positional text and no `--file`
- AND standard input is a terminal
- WHEN the command starts
- THEN the command fails with an error
- AND no mode work starts

#### Scenario: Empty or whitespace-only prompt fails

- GIVEN the prompt source contains only whitespace
- WHEN the command starts
- THEN the command fails with an error
- AND no mode work starts

#### Scenario: Unreadable file fails

- GIVEN the user passes `--file` with a path that does not exist or cannot be read
- WHEN the command starts
- THEN the command fails with an error
- AND no mode work starts

#### Scenario: Leading dash needs a separator

- GIVEN the prompt text begins with `-`
- WHEN the user passes `ralph prompt -- --version`
- THEN `--version` is treated as the prompt text
- AND it is not treated as a command option

---

### Requirement: Command options

The command SHALL accept `--mode`, `--base` (`-B`), `--model`, `--variant`, `--agent`, `--context`, `--namespace` (`-n`), `--no-services`, `--no-follow`, `--no-notify`, `--debug <branch>`, and `--verbose`.

The command SHALL NOT accept project or item options, including `--items` and `--extra`.

`--context` and `--namespace` SHALL apply only in `remote` mode. `--no-follow` SHALL apply only in `remote` mode. `--debug` SHALL be rejected in `local` and `worktree` modes.

#### Scenario: Shared options are exposed

- GIVEN the user asks for the command usage
- WHEN the usage is shown
- THEN every option in this requirement is listed
- AND `--items` and `--extra` are not listed

#### Scenario: Project option rejected

- GIVEN the user passes `--items` with a prompt
- WHEN the command starts
- THEN the command fails with an unknown-option error
- AND no prompt pass runs

#### Scenario: Debug rejected outside remote mode

- GIVEN the mode resolves to `local` or `worktree`
- AND the user passes `--debug`
- WHEN the command starts
- THEN the command fails with an error
- AND no prompt pass runs

#### Scenario: Kubernetes options have no effect locally

- GIVEN the mode resolves to `local` or `worktree`
- AND the user passes `--context` or `--namespace`
- WHEN the command runs
- THEN no Kubernetes cluster is contacted

---

### Requirement: Mode resolution

The command SHALL resolve the execution mode in the same order as `run`:

1. `--mode` at the command line.
2. `mode` in `.ralph/config.yaml`.
3. `local`.

`--mode` SHALL accept exactly `local`, `worktree`, or `remote`. Any other value SHALL fail before any other work starts.

#### Scenario: Flag selects the mode

- GIVEN the user passes `--mode worktree`
- WHEN the command starts
- THEN execution runs in `worktree` mode

#### Scenario: Config supplies the mode

- GIVEN `mode: remote` is set in `.ralph/config.yaml`
- AND no `--mode` flag is passed
- WHEN the command starts
- THEN execution runs in `remote` mode

#### Scenario: Default mode is local

- GIVEN neither `--mode` nor `mode` in `.ralph/config.yaml` is set
- WHEN the command starts
- THEN execution runs in `local` mode

#### Scenario: Invalid mode rejected

- GIVEN the user passes `--mode sandbox`
- WHEN the command starts
- THEN the command fails with an error naming the valid modes
- AND no prompt pass runs

---

### Requirement: Base branch resolution

The command SHALL resolve the base branch in the same order as `run`:

1. `--base`.
2. The current branch, when it differs from the prompt branch.
3. `defaultBranch` in `.ralph/config.yaml`.

The resolved base branch SHALL be used for base branch synchronization and as the pull request target. It SHALL be resolved once and SHALL NOT be recomputed during the command.

#### Scenario: Explicit base branch wins

- GIVEN the user passes `--base release`
- WHEN the command starts
- THEN `release` is the base branch

#### Scenario: Current branch is the base

- GIVEN no `--base` flag is passed
- AND the current branch differs from the prompt branch
- WHEN the command starts
- THEN the current branch is the base branch

#### Scenario: Config default used when currently on the prompt branch

- GIVEN no `--base` flag is passed
- AND the current branch equals the prompt branch
- WHEN the command starts
- THEN `defaultBranch` from `.ralph/config.yaml` is the base branch

#### Scenario: Resolved base reaches synchronization and the pull request

- GIVEN the base branch resolves to `release`
- WHEN the command synchronizes and opens a pull request
- THEN both use `release`

---

### Requirement: AI options

The command SHALL accept `--model` and `--variant` following the shared [model-options.md](model-options.md) contract, and `--agent` following the agent precedence in [run.md](run.md).

The direct prompt, a service-startup fix, and base-conflict resolution SHALL use the configured agent, because they may write repository code. The prompt slug proposal and the pull request summary SHALL run with OpenCode's primary agent instead.

The prompt SHALL be used as written. Custom instructions from `.ralph/config.yaml` SHALL NOT apply.

#### Scenario: Model and variant flags are used

- GIVEN the user passes `--model claude-opus-4-8 --variant high`
- WHEN the pass runs
- THEN the direct prompt uses that model and variant

#### Scenario: Agent flag applies to the direct prompt

- GIVEN the user passes `--agent build`
- WHEN the direct prompt runs
- THEN it runs with the `build` agent

#### Scenario: Agent applies to service and conflict fixes

- GIVEN the configured agent is `build`
- WHEN a service-startup fix or a base-conflict resolution runs
- THEN it runs with the `build` agent

#### Scenario: Supporting prompts use the primary agent

- GIVEN the configured agent is `build`
- WHEN the prompt slug or the pull request summary runs
- THEN it runs without the `build` agent

#### Scenario: Custom instructions do not apply

- GIVEN `.ralph/config.yaml` defines custom instructions
- WHEN the direct prompt runs
- THEN the prompt text is sent as written
- AND the custom instructions are not appended

---

### Requirement: Prompt branch

Ralph SHALL ask the AI for a short slug from the prompt and use the cleaned slug as the prompt branch name. Branch name cleanup SHALL match a [project branch](run.md#requirement-branch-name-derived-from-project-slug).

Before the pull request, Ralph SHALL ask the AI for a short slug from the commits the pass made and rename the prompt branch to the cleaned slug. The pull request SHALL use the renamed branch as its head.

A failed slug proposal SHALL fail the command before any branch, service, or pass work starts. A failed branch rename SHALL fail the command before the push or pull request.

The prompt branch SHALL be created and checked out according to the execution mode:

- `local` checks out the prompt branch in the current checkout, creating it from the current branch when it does not exist.
- `worktree` creates a worktree on the prompt branch with the same origin and local rules as a run, and refuses a branch already checked out in another worktree.
- `remote` clones the current branch, then checks out the prompt branch in the container, creating it from the clone branch when it does not exist.

#### Scenario: Slug proposed from the prompt

- GIVEN a valid prompt
- WHEN the command starts
- THEN Ralph asks the AI for a short slug
- AND the prompt branch name comes from that slug

#### Scenario: Slug is cleaned

- GIVEN the AI proposes a slug with spaces, capitals, or special characters
- WHEN the prompt branch is named
- THEN the name is cleaned up like a project branch

#### Scenario: Failed slug proposal stops the command

- GIVEN the AI fails to propose a slug
- WHEN the command starts
- THEN the command fails
- AND no branch is created
- AND no services start
- AND no pass runs

#### Scenario: Local mode creates the branch from the current branch

- GIVEN the prompt branch does not exist
- AND the mode is `local`
- WHEN the command starts
- THEN the prompt branch is created from the current branch
- AND it is checked out in the current checkout

#### Scenario: Worktree refuses a branch checked out elsewhere

- GIVEN the prompt branch is already checked out in another worktree
- AND the mode is `worktree`
- WHEN the command starts
- THEN the command fails with an error
- AND no pass runs

#### Scenario: Remote mode creates the branch from the clone branch

- GIVEN the prompt branch does not exist on the remote
- AND the mode is `remote`
- WHEN the container prepares the workspace
- THEN it clones the current branch
- AND it creates the prompt branch from that clone branch

#### Scenario: Branch is renamed from the work before the pull request

- GIVEN the pass leaves commits
- AND the AI proposes a slug from the commit log that differs from the prompt branch
- WHEN Ralph prepares the pull request
- THEN the prompt branch is renamed to the cleaned slug
- AND the push and pull request use the renamed branch

#### Scenario: A slug that matches the branch leaves the name alone

- GIVEN the pass leaves commits
- AND the AI proposes a slug from the commit log that cleans to the prompt branch name
- WHEN Ralph prepares the pull request
- THEN the prompt branch keeps its name

#### Scenario: Failed branch proposal or rename stops the pull request

- GIVEN the pass leaves commits
- AND the AI fails to propose a slug from the commit log, or the rename fails
- WHEN Ralph prepares the pull request
- THEN the command fails
- AND no branch is pushed
- AND no pull request opens

---

### Requirement: Setup before the pass

A prompt SHALL run its setup in this order:

1. Run the configured before commands.
2. Check out the prompt branch.
3. Synchronize the base branch, following [base-branch-sync.md](base-branch-sync.md).
4. Start the configured services.
5. Run the pass.

Before commands SHALL run once. They SHALL run before the branch switch in `local` and `remote` modes, and inside the new worktree in `worktree` mode. A required before command that fails SHALL fail the command. An optional before command that fails SHALL log a warning and continue.

The configuration SHALL come from `.ralph/config.yaml` in the checkout.

When a service fails to start or become healthy, Ralph SHALL run the service-fix prompt before the pass. If the fix succeeds, the pass SHALL continue without services. If the fix fails, the command SHALL fail.

`--no-services` SHALL skip service startup and the service-fix prompt. Before commands SHALL still run.

#### Scenario: Setup runs in order

- GIVEN the mode is `local`
- WHEN the command starts
- THEN the before commands run
- AND the prompt branch is checked out
- AND the base branch is synchronized
- AND the services start
- AND then the pass runs

#### Scenario: Required before command fails

- GIVEN a before command without `optional` exits non-zero
- WHEN the command starts
- THEN the command fails
- AND no pass runs

#### Scenario: Optional before command failure continues

- GIVEN a before command marked `optional` exits non-zero
- WHEN the command starts
- THEN a warning is logged
- AND the command continues

#### Scenario: Before commands run inside a worktree

- GIVEN the mode is `worktree`
- WHEN the command starts
- THEN the before commands run inside the new worktree

#### Scenario: Service failure runs the fix prompt

- GIVEN a configured service fails to start or become healthy
- WHEN setup reaches the services step
- THEN the service-fix prompt runs before the pass

#### Scenario: Successful service fix continues without services

- GIVEN the service-fix prompt succeeds
- WHEN the pass runs
- THEN the pass runs without the failed services

#### Scenario: Failed service fix fails the command

- GIVEN the service-fix prompt fails
- WHEN setup completes
- THEN the command fails
- AND the pass does not run

#### Scenario: Services skipped

- GIVEN the user passes `--no-services`
- WHEN the command runs
- THEN no services start
- AND the service-fix prompt does not run
- AND the before commands still run

---

### Requirement: Single pass

The prompt SHALL run as one non-interactive OpenCode pass. A prompt slug proposal, a service fix, a base-conflict resolution, or a pull request summary MAY run as a supporting pass. The command SHALL have no item picker, development prompt, changelog, report contract, block detection, completion trailer, or iteration recovery. It SHALL have no iteration cap.

Each invocation SHALL start a fresh OpenCode session. Ralph SHALL read no report. It SHALL NOT restrict the tools the prompt may use, so the prompt may run Git commands itself. The transcript SHALL be OpenCode's standard output and standard error. A non-zero OpenCode exit SHALL fail the command.

Cancel a `local` or `worktree` pass with Ctrl-C. Stop a `remote` pass with `ralph stop`.

#### Scenario: One pass runs

- GIVEN a valid prompt
- WHEN the command runs
- THEN the prompt text is sent as one non-interactive OpenCode pass
- AND no project items are read
- AND no report file is read

#### Scenario: Supporting passes are limited

- GIVEN a service failure, a base conflict, or commits after the pass
- WHEN the matching step runs
- THEN only the service fix, conflict resolution, or pull request summary runs in addition to the direct prompt
- AND no picker, development, or changelog pass runs

#### Scenario: Prompt runs tools itself

- GIVEN the prompt asks the agent to commit
- WHEN the pass runs
- THEN the agent may run Git commands
- AND Ralph does not block them

#### Scenario: Non-zero exit fails the command

- GIVEN OpenCode exits non-zero
- WHEN the pass ends
- THEN the command fails

#### Scenario: Remote stop does not stop the workflow

- GIVEN a followed remote prompt is running
- WHEN the user stops the local command
- THEN the local command returns
- AND the workflow continues

---

### Requirement: Commit detection and pull request

Ralph SHALL NOT stage or commit the prompt's working tree and SHALL NOT choose the prompt's commit messages. The prompt commits according to its own instructions. A prompt that leaves changes uncommitted SHALL be a valid outcome.

Ralph SHALL record the prompt branch tip after base branch synchronization. After a successful pass it SHALL compare the tip:

- If the tip did not advance, the prompt made no commits. The command SHALL succeed without a push or pull request. Uncommitted changes SHALL remain in a `local` checkout. `worktree` and `remote` modes SHALL discard them.
- If the tip advanced, the prompt made commits. Ralph SHALL rename the prompt branch from the commit log, synchronize the base branch again, push the renamed branch, generate a pull request description from the commit log with the pull request summary prompt, and open a pull request to the base branch.

The pull request summary prompt SHALL receive the prompt text in place of a project description. The pull request title SHALL be the first H1 in the summary, falling back to the renamed branch name.

A failed pass SHALL make no push and open no pull request. If the summary or pull request step fails, the command SHALL fail and the pushed prompt branch SHALL remain.

#### Scenario: No commits opens no pull request

- GIVEN the pass leaves the branch tip unchanged
- WHEN the pass succeeds
- THEN the command succeeds
- AND no branch is pushed
- AND no pull request opens

#### Scenario: Local uncommitted changes remain

- GIVEN the mode is `local`
- AND the pass leaves uncommitted changes and no commits
- WHEN the command ends
- THEN the uncommitted changes remain in the checkout

#### Scenario: Worktree and remote discard uncommitted changes

- GIVEN the mode is `worktree` or `remote`
- AND the pass leaves uncommitted changes and no commits
- WHEN the command ends
- THEN the uncommitted changes are discarded

#### Scenario: Commits are pushed and a pull request opens

- GIVEN the pass advances the branch tip
- WHEN the pass succeeds
- THEN the prompt branch is renamed from the commit log
- AND the base branch is synchronized again
- AND the renamed branch is pushed
- AND a pull request opens to the base branch

#### Scenario: Summary receives the prompt text

- GIVEN the pass leaves commits
- WHEN the pull request summary runs
- THEN it receives the prompt text in place of a project description

#### Scenario: Pull request title comes from the summary

- GIVEN the summary contains an H1 heading
- WHEN the pull request opens
- THEN its title is that H1 text

#### Scenario: Pull request title falls back to the branch name

- GIVEN the summary contains no H1 heading
- WHEN the pull request opens
- THEN its title is the renamed branch name

#### Scenario: Failed pass makes no push

- GIVEN the pass fails
- WHEN the command ends
- THEN no branch is pushed
- AND no pull request opens

#### Scenario: Summary failure leaves the pushed branch

- GIVEN the branch was pushed
- AND the summary or pull request step fails
- WHEN the command ends
- THEN the command fails
- AND the pushed prompt branch remains

---

### Requirement: Branch restoration

A `local` prompt SHALL restore the starting branch after it opens a pull request. A `worktree` prompt SHALL leave the starting checkout untouched.

#### Scenario: Local branch restored after the pull request

- GIVEN the mode is `local`
- AND the prompt opened a pull request
- WHEN the command ends
- THEN the starting branch is checked out again

#### Scenario: Worktree leaves the starting checkout alone

- GIVEN the mode is `worktree`
- WHEN the command ends
- THEN the starting checkout is still on its original branch

---

### Requirement: Remote submission

In `remote` mode the command SHALL require the current branch to exist on the remote at the same commit, because the workflow container clones that branch. Uncommitted changes SHALL NOT be sent.

The command SHALL resolve the prompt slug, prompt branch, and base branch locally. A prompt read from a file or standard input SHALL be read locally before submission and sent as text.

The workflow SHALL use the `ralph-prompt-` name prefix, carry the Ralph ownership label, and use the configured image, mounts, environment, labels, and resources. It SHALL target the resolved Kubernetes context and namespace, and use the same TTL and pod garbage collection as the other workflows. It SHALL take the prompt branch lock, as a run does.

Ralph SHALL embed the prompt in the workflow manifest as one container argument. The prompt is therefore visible to anyone who can read workflows in the namespace.

Remote mode SHALL follow logs by default and wait for the workflow to finish. `--no-follow` SHALL submit and return with the workflow name and an `argo logs` hint. A followed workflow SHALL send a completion notification unless `--no-notify` is set. Stopping the local command SHALL NOT stop the workflow. A followed failure SHALL exit non-zero. A submission without follow SHALL exit zero once the workflow is accepted.

`--debug <branch>` SHALL run Ralph from that source branch inside the container.

#### Scenario: Branch must be in sync with the remote

- GIVEN the current branch is missing on the remote, ahead of it, or behind it
- WHEN the user runs a remote prompt
- THEN submission fails before a workflow is created

#### Scenario: Uncommitted changes are not sent

- GIVEN the checkout has uncommitted changes
- WHEN the workflow is submitted
- THEN only the current branch's remote commits are sent

#### Scenario: Workflow is named and labeled for Ralph

- GIVEN a remote prompt is submitted
- WHEN the workflow is created
- THEN its name starts with `ralph-prompt-`
- AND it carries the Ralph ownership label

#### Scenario: Prompt is one container argument

- GIVEN a prompt is submitted
- WHEN the workflow manifest is created
- THEN the prompt is a single container argument

#### Scenario: Logs follow by default

- GIVEN a remote prompt is submitted without `--no-follow`
- WHEN the workflow starts
- THEN its logs stream locally
- AND the command waits for completion

#### Scenario: No-follow returns after submission

- GIVEN the user passes `--no-follow`
- WHEN the workflow is accepted
- THEN the command prints the workflow name and an `argo logs` hint
- AND it exits zero

#### Scenario: Followed failure exits non-zero

- GIVEN a followed workflow fails
- WHEN the command ends
- THEN it exits non-zero

#### Scenario: Notification on completion

- GIVEN a followed workflow finishes
- AND `--no-notify` is not set
- WHEN the command ends
- THEN a completion notification is sent

#### Scenario: Debug runs from a source branch

- GIVEN the mode is `remote`
- AND the user passes `--debug feature-branch`
- WHEN the workflow runs
- THEN the container runs Ralph from `feature-branch` instead of the installed binary

---

### Requirement: Container entrypoint

`ralph workflow prompt` SHALL be the container entrypoint. It SHALL validate the prompt, prepare the workspace, and run the flow defined in this specification in the cloned checkout.

It SHALL accept the branch, base, prompt, model, variant, agent, and `--no-services` from the workflow. It SHALL validate the prompt before workspace setup.

#### Scenario: Container runs the prompt flow

- GIVEN a workflow invokes `ralph workflow prompt` with a valid prompt
- WHEN the entrypoint runs
- THEN the workspace is prepared
- AND the flow runs in the cloned checkout

#### Scenario: Invalid prompt fails before workspace setup

- GIVEN the workflow supplies a missing or empty prompt
- WHEN the entrypoint starts
- THEN it fails
- AND no workspace setup runs

#### Scenario: Service choice reaches the container

- GIVEN the workflow was submitted with `--no-services`
- WHEN the entrypoint runs
- THEN it skips service startup and the service-fix prompt
