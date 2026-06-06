# Argo Specification

## Purpose

Convenience commands for inspecting and managing Argo Workflows created by ralph. Both commands scope to the ralph config namespace by default and support optional overrides for Kubernetes context and namespace.

## Requirements

### Requirement: `ralph list` shows ralph-owned workflows

The system SHALL list Argo Workflows in the active namespace filtered by the label `app.kubernetes.io/managed-by=ralph`, so only workflows created by ralph are shown.

#### Scenario: Default list

- GIVEN no flags are provided
- WHEN the user runs `ralph list`
- THEN ralph calls `argo list` filtered by `app.kubernetes.io/managed-by=ralph`
- AND scoped to the namespace from the ralph config

#### Scenario: Custom namespace

- GIVEN the user passes `--namespace staging` (or `-n staging`)
- WHEN the user runs `ralph list -n staging`
- THEN ralph lists ralph-owned workflows in the `staging` namespace

#### Scenario: Custom context

- GIVEN the user passes `--context prod-cluster`
- WHEN the user runs `ralph list --context prod-cluster`
- THEN ralph lists ralph-owned workflows using the `prod-cluster` Kubernetes context

#### Scenario: Custom context and namespace together

- GIVEN the user passes both `--context prod-cluster` and `-n staging`
- WHEN the user runs `ralph list --context prod-cluster -n staging`
- THEN ralph lists ralph-owned workflows in the `staging` namespace of the `prod-cluster` context

---

### Requirement: `ralph stop` stops a workflow by name

The system SHALL stop a named Argo Workflow in the active namespace.

#### Scenario: Stop by workflow name

- GIVEN a workflow name is provided
- WHEN the user runs `ralph stop <workflow-name>`
- THEN ralph calls `argo stop` for that workflow
- AND scoped to the namespace from the ralph config

#### Scenario: Missing workflow name

- GIVEN no workflow name is provided
- WHEN the user runs `ralph stop`
- THEN an error is returned with usage instructions

#### Scenario: Custom namespace

- GIVEN the user passes `--namespace staging` (or `-n staging`)
- WHEN the user runs `ralph stop -n staging <workflow-name>`
- THEN ralph stops the workflow in the `staging` namespace

#### Scenario: Custom context

- GIVEN the user passes `--context prod-cluster`
- WHEN the user runs `ralph stop --context prod-cluster <workflow-name>`
- THEN ralph stops the workflow using the `prod-cluster` Kubernetes context

---

### Requirement: Namespace resolution order

Both commands SHALL resolve the namespace using the following precedence (highest to lowest):

1. `--namespace` / `-n` flag value
2. Namespace from the ralph config
3. Default namespace of the active Kubernetes context

#### Scenario: Flag overrides config namespace

- GIVEN the ralph config specifies namespace `default`
- AND the user passes `-n staging`
- WHEN either command runs
- THEN the `staging` namespace is used

#### Scenario: Config namespace used when no flag given

- GIVEN the ralph config specifies namespace `platform`
- AND no `--namespace` flag is given
- WHEN either command runs
- THEN the `platform` namespace is used

