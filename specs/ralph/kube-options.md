# Kubectl Commands Specification

## Purpose

Shared contract for every Ralph CLI command that interacts with a Kubernetes cluster through kubectl. These commands SHALL accept `--context` and `--namespace` options to target a specific cluster and namespace, overriding the values in `.ralph/config.yaml`. Individual command specs link here for the shared behavior.

## Requirements

### Requirement: Covered commands

The commands that interact with kubectl are: `ralph list`, `ralph stop`, `ralph logs`, `ralph run` (in `remote` mode), `ralph loop` (in `remote` mode), `ralph command`, and `ralph setup`. Each SHALL accept the `--context` and `--namespace` options described below.

#### Scenario: Every covered command exposes both options

- GIVEN any covered command
- WHEN its usage is shown
- THEN `--context` and `--namespace` (alias `-n`) are listed among its options

#### Scenario: Flag values reach the cluster interaction

- GIVEN the user passes `--context staging -n argo`
- WHEN any covered command interacts with the cluster
- THEN the `staging` context and `argo` namespace are used for every cluster call the command makes

---

### Requirement: `--context` selects the cluster

The system SHALL accept `--context <name>` on every covered command to select the Kubernetes context used for cluster interaction.

Context resolution SHALL follow this precedence (highest to lowest):

1. `--context` flag value
2. `context` in the `workflow:` section of `.ralph/config.yaml`
3. Current context of the active kubeconfig

#### Scenario: Flag overrides the configured context

- GIVEN `workflow.context: staging` is set in `.ralph/config.yaml`
- AND the user passes `--context canary`
- WHEN the command interacts with the cluster
- THEN the `canary` context is used instead of the config value

#### Scenario: Config context used when no flag is passed

- GIVEN `workflow.context: staging` is set in `.ralph/config.yaml`
- AND no `--context` flag is passed
- WHEN the command interacts with the cluster
- THEN the `staging` context is used

#### Scenario: Current context used when flag and config are unset

- GIVEN neither `--context` nor `workflow.context` is set
- AND the active kubeconfig has a current context `prod`
- WHEN the command interacts with the cluster
- THEN the `prod` context is used

---

### Requirement: `--namespace` selects the namespace

The system SHALL accept `--namespace <name>` (alias `-n`) on every covered command to select the Kubernetes namespace used for cluster interaction.

Namespace resolution SHALL follow this precedence (highest to lowest):

1. `--namespace` / `-n` flag value
2. `namespace` in the `workflow:` section of `.ralph/config.yaml`
3. Default namespace of the resolved Kubernetes context

#### Scenario: Flag overrides the configured namespace

- GIVEN `workflow.namespace: default` is set in `.ralph/config.yaml`
- AND the user passes `-n staging`
- WHEN the command interacts with the cluster
- THEN the `staging` namespace is used instead of the config value

#### Scenario: Config namespace used when no flag is passed

- GIVEN `workflow.namespace: platform` is set in `.ralph/config.yaml`
- AND no `--namespace` flag is passed
- WHEN the command interacts with the cluster
- THEN the `platform` namespace is used

#### Scenario: Context namespace used when flag and config are unset

- GIVEN neither `--namespace` nor `workflow.namespace` is set
- AND the resolved context carries a default namespace `argo`
- WHEN the command interacts with the cluster
- THEN the `argo` namespace is used

---

### Requirement: kubectl must be installed and configured

When a covered command must resolve a Kubernetes context and kubectl is missing, or the kubeconfig has no current context, the command SHALL fail with a clear error.

#### Scenario: kubectl not installed

- GIVEN kubectl is not on the PATH
- AND no `--context` or config context is set
- WHEN a covered command resolves the cluster
- THEN an error is returned telling the user kubectl is not installed
- AND no cluster interaction is attempted

#### Scenario: No current context in kubeconfig

- GIVEN kubectl is installed
- AND the kubeconfig has no current context
- AND no `--context` or config context is set
- WHEN a covered command resolves the cluster
- THEN an error is returned telling the user the current context could not be determined

---

### Requirement: Commands without a cluster interaction do not require the options

The `--context` and `--namespace` options SHALL be meaningful only where a command interacts with the cluster. Commands that run entirely in-process (`ralph complete`, `ralph incomplete`, `ralph validate`, and `ralph run`/`ralph loop` in `local` or `worktree` mode) SHALL NOT require these options and SHALL accept them without effect.

#### Scenario: Local run ignores the options

- GIVEN the user runs `ralph run --mode local --context staging -n argo`
- WHEN the run executes
- THEN no cluster interaction occurs
- AND the options do not cause an error
