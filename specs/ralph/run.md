# Run Command Specification

## Purpose

The `run` command is Ralph's primary entry point. Given a project file, an orchestration document, or a spec document, it drives an AI coding agent through iterative development cycles until every item in the project is recorded complete, then opens a GitHub pull request. A project file is any YAML or JSON file containing an array of work items. The array is selected with the [item query](../../docs/glossary.md#item-query), and completion is recorded in the branch's commit messages, not in the file. When an orchestration or spec is provided instead of a project, Ralph generates the missing artifacts and commits them before running. Execution runs in one of three modes selected with `--mode`: `local` (default), which runs the loop in-process in the current checkout, `worktree`, which runs it in-process in a local Git worktree, or `remote`, which submits an Argo Workflow to Kubernetes.

Mode-specific behaviors are defined in:
- [run-local.md](run-local.md) — `local` mode: runs the development loop in-process in the current checkout
- [run-worktree.md](run-worktree.md) — `worktree` mode: runs the development loop in-process in a local Git worktree
- [run-remote.md](run-remote.md) — `remote` mode: submits an Argo Workflow to Kubernetes

## Requirements

### Requirement: Execution mode

The command SHALL accept `--mode` to select the execution mode. The option SHALL accept exactly one of `local`, `worktree`, or `remote`.

Mode resolution follows a three-level precedence: `--mode` at the command line takes priority. Otherwise the top-level `mode` field in `.ralph/config.yaml` is used. Otherwise the mode defaults to `local`.

- `local` runs the development loop in-process in the current checkout.
- `worktree` runs the development loop in-process in a Git worktree created for the project branch, leaving the current checkout untouched. See [run-worktree.md](run-worktree.md).
- `remote` submits an Argo Workflow to Kubernetes and returns after submission. The loop runs inside the workflow container. See [run-remote.md](run-remote.md).

The `--follow` and `--debug` flags are workflow-only and are rejected for `local` and `worktree` modes. See [Incompatible flags are rejected](#requirement-incompatible-flags-are-rejected).

#### Scenario: `--mode local` runs in the current checkout

- GIVEN the user passes `--mode local`
- WHEN the command starts
- THEN the development loop runs in-process in the current checkout
- AND no workflow is submitted

#### Scenario: `--mode worktree` runs in a Git worktree

- GIVEN the user passes `--mode worktree`
- WHEN the command starts
- THEN a Git worktree is created for the project branch
- AND the development loop runs in-process inside that worktree

#### Scenario: `--mode remote` submits a workflow

- GIVEN the user passes `--mode remote`
- WHEN the command starts
- THEN an Argo Workflow is submitted to Kubernetes
- AND the loop runs inside the workflow container

#### Scenario: Default mode is `local`

- GIVEN neither `--mode` nor `mode` in `.ralph/config.yaml` is set
- WHEN the command starts
- THEN execution runs in `local` mode

#### Scenario: Config mode used when no flag is passed

- GIVEN `mode: remote` is set in `.ralph/config.yaml`
- AND no `--mode` flag is passed
- WHEN the command starts
- THEN execution runs in `remote` mode

#### Scenario: `--mode` overrides the configured mode

- GIVEN `mode: remote` is set in `.ralph/config.yaml`
- AND the user passes `--mode local`
- WHEN the command starts
- THEN execution runs in `local` mode instead of the config value

#### Scenario: Invalid mode value rejected

- GIVEN the user passes `--mode sandbox`
- WHEN the command starts
- THEN an error is returned: `invalid mode: sandbox (expected local, worktree, or remote)`
- AND no execution begins

---

### Requirement: Input file is required

The command SHALL require a positional argument that is a path to one of: a project file (`.yaml`, `.yml`, or `.json`), an orchestration document (`orchestration.md`), or a spec document (`spec.md`). The file must exist on disk before execution proceeds. When an orchestration or spec is provided, the actual project generation and artifact commits happen inside the execution mode. See [run-local.md](run-local.md) and [run-worktree.md](run-worktree.md).

#### Scenario: Project file provided

- GIVEN the user provides a path to a project file that parses and yields items
- WHEN the command starts
- THEN the project file is loaded and execution proceeds

#### Scenario: Orchestration file provided

- GIVEN the user provides a path to a file named `orchestration.md`
- WHEN the command starts
- THEN the input is forwarded to the execution mode for just-in-time project generation

#### Scenario: Spec file provided

- GIVEN the user provides a path to a file named `spec.md`
- WHEN the command starts
- THEN the input is forwarded to the execution mode for just-in-time orchestration and project generation

#### Scenario: Input file not found

- GIVEN the user provides a path to a file that does not exist on disk
- WHEN the command starts
- THEN an error is returned: `input file not found: <path>`
- AND no execution begins

#### Scenario: Unrecognized file type

- GIVEN the user provides a path to a file that is not a `.yaml`/`.yml`/`.json` file, `orchestration.md`, or `spec.md`
- WHEN the command starts
- THEN an error is returned: `unrecognized input file type: <path>`
- AND no execution begins

---

### Requirement: Working directory override

The command SHALL change its working directory to the path given by `--working-dir` (`-C`) before any other setup occurs, allowing the command to be invoked against a project in a different directory.

#### Scenario: `--working-dir` changes the working directory

- GIVEN the user passes `--working-dir /path/to/project`
- WHEN the command starts
- THEN the working directory is changed to `/path/to/project` before the project file is loaded

---

### Requirement: AI model, variant, and Kubernetes context overrides

The command SHALL accept `--model` and `--variant` to override the AI model and its provider-specific reasoning-effort variant from config, per the shared [model-options.md](model-options.md) contract. The command SHALL also accept `--context` and `--namespace` to target the Kubernetes cluster used for remote workflow submission, per the shared [kube-options.md](kube-options.md) contract.

#### Scenario: `--context` overrides the Kubernetes context

- GIVEN the user passes `--context my-cluster`
- WHEN a remote workflow is submitted
- THEN `my-cluster` is used as the Kubernetes context instead of the default

---

### Requirement: opencode agent override

The command SHALL accept `--agent` to select which opencode agent runs the AI prompts that write repository code. The agent SHALL apply only to prompts that change code: item development, merge-conflict resolution, PR comment implementation, and service-startup fixes. Prompts that produce supporting artifacts without touching repository code (item selection, orchestration and project generation, changelogs, PR summaries, and PR review bodies) SHALL run with opencode's primary agent and SHALL NOT receive the configured agent.

Agent resolution follows a two-level precedence: `--agent` at the command line takes priority. Otherwise the top-level `agent` field in `.ralph/config.yaml` is used. When both are unset, no agent is passed to any prompt.

#### Scenario: `--agent` flag applies to the item development prompt

- GIVEN the user passes `--agent code-reviewer`
- WHEN an item development prompt runs
- THEN `--agent code-reviewer` is included in its opencode invocation

#### Scenario: Config agent used when no flag is passed

- GIVEN `agent: build` is set in `.ralph/config.yaml`
- AND no `--agent` flag is passed
- WHEN a prompt that writes code runs
- THEN `--agent build` is included in its opencode invocation

#### Scenario: Agent applies to merge-conflict resolution

- GIVEN the agent resolves to `build`
- WHEN a merge-conflict resolution prompt runs
- THEN `--agent build` is included in its opencode invocation

#### Scenario: Agent applies to PR comment implementation

- GIVEN the agent resolves to `build`
- WHEN a PR comment prompt runs
- THEN `--agent build` is included in its opencode invocation

#### Scenario: Agent applies to service-startup fixes

- GIVEN the agent resolves to `build`
- WHEN a service-startup fix prompt runs
- THEN `--agent build` is included in its opencode invocation

#### Scenario: Item picker runs without the agent

- GIVEN the agent resolves to `build`
- WHEN the item picker prompt runs
- THEN the `--agent` option is omitted from its opencode invocation, and opencode's primary agent is used

#### Scenario: Artifact generation runs without the agent

- GIVEN the agent resolves to `build`
- WHEN an orchestration or project generation prompt runs
- THEN the `--agent` option is omitted from its opencode invocation, and opencode's primary agent is used

#### Scenario: Changelog, PR summary, and review prompts run without the agent

- GIVEN the agent resolves to `build`
- WHEN a changelog, PR summary, or PR review body prompt runs
- THEN the `--agent` option is omitted from its opencode invocation, and opencode's primary agent is used

#### Scenario: Agent omitted when both flag and config are unset

- GIVEN `agent` is not set in `.ralph/config.yaml`
- AND no `--agent` flag is passed
- WHEN any prompt runs
- THEN the `--agent` option is omitted from every opencode invocation, and opencode's primary agent is used

#### Scenario: `--context` overrides the Kubernetes context

- GIVEN the user passes `--context my-cluster`
- WHEN a remote workflow is submitted
- THEN `my-cluster` is used as the Kubernetes context instead of the default

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

#### Scenario: `--debug` with `--mode local`

- GIVEN the user passes `--debug <branch>` and `--mode local`
- WHEN the command validates flag combinations
- THEN an error is returned: `--debug flag is not applicable with --mode local`

#### Scenario: `--debug` with `--mode worktree`

- GIVEN the user passes `--debug <branch>` and `--mode worktree`
- WHEN the command validates flag combinations
- THEN an error is returned: `--debug flag is not applicable with --mode worktree`

---

### Requirement: Base branch resolution

The command SHALL determine the base branch for PR creation by the following priority: explicit `--base` flag > current branch (when different from project branch) > config default branch. This resolution SHALL happen once, locally, before dispatching to run-local, run-worktree, or run-remote, and the resolved value SHALL be passed down as a parameter rather than recomputed by the runner.

#### Scenario: Explicit `--base` flag

- GIVEN the user passes `--base develop`
- WHEN the base branch is resolved
- THEN `develop` is used regardless of other state

#### Scenario: Current branch differs from project branch

- GIVEN the current branch is `feature-x` and the project branch would be `my-project`
- AND no `--base` flag is provided
- WHEN the base branch is resolved
- THEN `feature-x` is used as the base branch

#### Scenario: Already on the project branch

- GIVEN the current branch is `my-project` and the project branch is also `my-project`
- AND no `--base` flag is provided
- WHEN the base branch is resolved
- THEN the config default branch (e.g. `main`) is used

#### Scenario: Resolved base branch passed to run-local

- GIVEN the base branch has been resolved locally
- AND the command runs in `local` mode
- WHEN execution is dispatched
- THEN the resolved base branch is passed to the run-local behavior described in [run-local.md](run-local.md)
- AND run-local does not recompute the base branch

#### Scenario: Resolved base branch passed to run-worktree

- GIVEN the base branch has been resolved locally
- AND the command runs in `worktree` mode
- WHEN execution is dispatched
- THEN the resolved base branch is passed to the run-worktree behavior described in [run-worktree.md](run-worktree.md)
- AND run-worktree does not recompute the base branch

#### Scenario: Resolved base branch passed to run-remote

- GIVEN the base branch has been resolved locally
- AND the command runs in `remote` mode
- WHEN execution is dispatched
- THEN the resolved base branch is passed to the run-remote behavior described in [run-remote.md](run-remote.md)
- AND run-remote does not recompute the base branch

---

### Requirement: Item query resolution

The command SHALL accept `--items` to set the jq query that selects the item array from the project file. The query SHALL be resolved once, locally, before dispatching to the selected execution mode, and the resolved value SHALL be passed down so that the whole run, whichever mode, indexes items against the same query.

Item query resolution follows a three-level precedence: `--items` at the command line takes priority. Otherwise the `items` field in `.ralph/config.yaml` is used. Otherwise the query defaults to `.`.

#### Scenario: `--items` overrides the configured query

- GIVEN `items: .requirements` is set in `.ralph/config.yaml`
- AND the user passes `--items '.spec.tasks'`
- WHEN the item query is resolved
- THEN the resolved query is `.spec.tasks`

#### Scenario: Config query used when no flag is passed

- GIVEN `items: .requirements` is set in `.ralph/config.yaml`
- AND no `--items` flag is passed
- WHEN the item query is resolved
- THEN the resolved query is `.requirements`

#### Scenario: Default query when flag and config are unset

- GIVEN `items` is not set in `.ralph/config.yaml`
- AND no `--items` flag is passed
- WHEN the item query is resolved
- THEN the resolved query is `.`

#### Scenario: Resolved query passed to the execution mode

- GIVEN the item query has been resolved locally
- WHEN execution is dispatched to the selected mode (run-local, run-worktree, or run-remote)
- THEN the resolved query is passed down as a parameter
- AND the execution mode does not re-resolve it from config

---

### Requirement: Cleanup resolution

The command SHALL accept `--cleanup` to request that the project file be deleted in its own commit once every item is complete. The resolved value SHALL be passed down to the execution mode, which performs the deletion. See [run-local.md](run-local.md).

Cleanup resolution follows a three-level precedence: `--cleanup` at the command line takes priority. Otherwise the `cleanup` field in `.ralph/config.yaml` is used. Otherwise cleanup is disabled.

#### Scenario: `--cleanup` enables cleanup for one run

- GIVEN `cleanup` is not set in `.ralph/config.yaml`
- AND the user passes `--cleanup`
- WHEN cleanup is resolved
- THEN cleanup is enabled for this run

#### Scenario: Config value used when no flag is passed

- GIVEN `cleanup: true` is set in `.ralph/config.yaml`
- AND no `--cleanup` flag is passed
- WHEN cleanup is resolved
- THEN cleanup is enabled

#### Scenario: Cleanup disabled by default

- GIVEN `cleanup` is not set in `.ralph/config.yaml`
- AND no `--cleanup` flag is passed
- WHEN cleanup is resolved
- THEN cleanup is disabled and the project file survives the run

---

### Requirement: Extra iterations resolution

The command SHALL accept `--extra` to set a finite extra iteration count. The resolved extra iteration value SHALL be passed down to the execution mode, which determines the default when the value is unset.

Extra iterations resolution follows a two-level precedence: `--extra` at the command line takes priority. Otherwise the `extraIterations` field in `.ralph/config.yaml` is used. The resolved value SHALL be passed to the execution mode as-is (nil/unset propagates to the runner).

#### Scenario: Flag takes precedence over config

- GIVEN `extraIterations: 5` in `.ralph/config.yaml`
- AND the user passes `--extra 2`
- WHEN the extra iteration count is resolved
- THEN the resolved value is 2

#### Scenario: Config value used when flag is absent

- GIVEN `extraIterations: 3` in `.ralph/config.yaml`
- AND no `--extra` flag is passed
- WHEN the extra iteration count is resolved
- THEN the resolved value is 3

#### Scenario: Both flag and config are unset

- GIVEN neither `extraIterations` in config nor `--extra` flag is set
- WHEN the extra iteration count is resolved
- THEN the resolved value is nil/unset
- AND the execution mode applies its default behavior

---

### Requirement: Project slug resolution

The project slug SHALL be taken from the project file's top-level `slug` field when the file's top level is a mapping carrying one, and SHALL otherwise fall back to the project file's base name without its extension. A project file whose top level is an array therefore always takes its slug from the file name.

#### Scenario: Slug field present

- GIVEN a project file whose top level is a mapping with `slug: csv-export`
- WHEN the slug is resolved
- THEN the slug is `csv-export`

#### Scenario: Top-level array has no slug field

- GIVEN a project file at `projects/csv-export.yaml` whose top level is an array
- WHEN the slug is resolved
- THEN the slug is `csv-export`, from the file's base name

#### Scenario: Mapping without a slug field

- GIVEN a project file at `projects/tasks.json` whose top level is a mapping with no `slug` field
- WHEN the slug is resolved
- THEN the slug is `tasks`, from the file's base name

---

### Requirement: Branch name derived from project slug

The project branch name SHALL be derived from the resolved project slug: lowercased, with spaces, underscores, and dots converted to hyphens, non-alphanumeric characters stripped, and consecutive or leading/trailing hyphens collapsed.

#### Scenario: Slug with spaces and capitals

- GIVEN a project slug `My Feature Work`
- WHEN the branch name is derived
- THEN the branch name is `my-feature-work`

#### Scenario: Slug with special characters

- GIVEN a project slug `fix: auth/bug`
- WHEN the branch name is derived
- THEN the branch name is `fix-authbug`

#### Scenario: Empty or all-invalid slug

- GIVEN a project slug that produces an empty string after sanitization
- WHEN the branch name is derived
- THEN the branch name is `unnamed-project`
