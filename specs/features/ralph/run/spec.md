# Run Command Specification

## Purpose

The `run` command is ralph's primary entry point. Given a project YAML file, it drives an AI coding agent through iterative development cycles until all project requirements pass, then opens a GitHub pull request. Execution can be delegated to an Argo Workflow (default) or run directly on the local machine (`--local`).

Mode-specific behaviors are defined in:
- [run-local/spec.md](../run-local/spec.md) — `--local` flag: runs the development loop in-process
- [run-remote/spec.md](../run-remote/spec.md) — default: submits an Argo Workflow to Kubernetes

## Requirements

### Requirement: Project file is required

The command SHALL require a project YAML file as a positional argument. Validation of the file's contents is handled by the validate feature.

#### Scenario: Project file provided

- GIVEN the user provides a path to a valid project YAML file
- WHEN the command starts
- THEN the project is loaded and execution proceeds

---

### Requirement: Incompatible flags are rejected

The command SHALL reject flag combinations that have no valid meaning before any execution begins.

#### Scenario: `--follow` with `--local`

- GIVEN the user passes both `--follow` and `--local`
- WHEN the command validates flag combinations
- THEN an error is returned: `--follow flag is not applicable with --local flag`

#### Scenario: `--debug` with `--local`

- GIVEN the user passes `--debug <branch>` and `--local`
- WHEN the command validates flag combinations
- THEN an error is returned: `--debug flag is not applicable with --local flag`

---

### Requirement: Base branch resolution

The command SHALL determine the base branch for PR creation by the following priority: explicit `--base` flag > current branch (when different from project branch) > config default branch.

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

---

### Requirement: Max iterations resolution

The iteration limit SHALL come from the `--max-iterations` flag when provided and non-zero; otherwise it falls back to `maxIterations` in `.ralph/config.yaml` (default: 10).

#### Scenario: Flag takes precedence over config

- GIVEN `maxIterations: 5` in `.ralph/config.yaml`
- AND the user passes `--max-iterations 2`
- WHEN the iteration limit is resolved
- THEN the limit is 2

#### Scenario: Config default used when flag is absent

- GIVEN `maxIterations: 7` in `.ralph/config.yaml`
- AND no `--max-iterations` flag is passed
- WHEN the iteration limit is resolved
- THEN the limit is 7

---

### Requirement: Branch name derived from project slug

The project branch name SHALL be derived from the project slug: lowercased, with spaces, underscores, and dots converted to hyphens, non-alphanumeric characters stripped, and consecutive or leading/trailing hyphens collapsed.

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
