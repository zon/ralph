# Logs Command Specification

## Purpose

Define the behavior of `ralph logs`, the command that prints the pod logs of a ralph-owned Argo Workflow. It takes an optional workflow name. When the name is omitted, ralph logs the workflow at the top of the `ralph list` output, the most recently created ralph workflow. Logs are printed once by default and streamed with `--follow` (`-f`). Like `ralph list` and `ralph stop`, it scopes to the ralph config namespace by default and supports overrides for Kubernetes context and namespace. The shared `--context` and `--namespace` targeting contract is defined in [kubectl.md](kubectl.md).

## Requirements

### Requirement: Command Invocation

The system SHALL provide a `ralph logs` command that accepts an optional workflow name as a positional argument. It SHALL support the flags `--follow` (`-f`), `--namespace` (`-n`), and `--context`.

#### Scenario: Logs with an explicit workflow name

- GIVEN a workflow named `ralph-csv-export` exists in the active namespace
- WHEN the user runs `ralph logs ralph-csv-export`
- THEN the pod logs of that workflow are printed

#### Scenario: Logs with no workflow name

- GIVEN at least one ralph-owned workflow exists
- WHEN the user runs `ralph logs`
- THEN the pod logs of the workflow at the top of the `ralph list` output are printed

#### Scenario: Custom namespace

- GIVEN the user passes `--namespace staging` (or `-n staging`)
- WHEN the user runs `ralph logs -n staging ralph-csv-export`
- THEN the pod logs are fetched from the `staging` namespace

#### Scenario: Custom context

- GIVEN the user passes `--context prod-cluster`
- WHEN the user runs `ralph logs --context prod-cluster ralph-csv-export`
- THEN the pod logs are fetched using the `prod-cluster` Kubernetes context

---

### Requirement: Workflow Name Resolution

When no workflow name is provided, the system SHALL select the workflow at the top of the `ralph list` output and log that workflow. When the list is empty, the system SHALL return an error instead of logging nothing.

#### Scenario: Top of the list selected

- GIVEN `ralph list` shows `ralph-b` above `ralph-a`
- AND no workflow name argument is passed
- WHEN the user runs `ralph logs`
- THEN the pod logs of `ralph-b` are printed

#### Scenario: No workflows to log

- GIVEN no ralph-owned workflows exist in the resolved namespace
- WHEN the user runs `ralph logs` without a workflow name argument
- THEN an error is returned stating that no workflows were found
- AND no pod logs are fetched

---

### Requirement: Log Output

The system SHALL print the pod logs of the resolved workflow, targeting the pod Argo names after the workflow. By default the command SHALL print the current logs and exit. With `--follow` (`-f`) it SHALL stream log lines as they are produced until the pod terminates.

#### Scenario: Logs printed once by default

- GIVEN a workflow exists and `--follow` is not set
- WHEN `ralph logs ralph-csv-export` runs
- THEN the current pod logs are printed to stdout
- AND the command exits after printing

#### Scenario: Logs streamed with follow

- GIVEN a workflow is still running
- WHEN `ralph logs --follow ralph-csv-export` (or `ralph logs -f ralph-csv-export`) runs
- THEN pod log lines are printed as they appear
- AND the command exits when the pod terminates

#### Scenario: Workflow pod not found

- GIVEN a workflow name that has no pod in the resolved namespace
- WHEN `ralph logs ralph-csv-export` runs
- THEN an error is returned naming the missing pod

---

### Requirement: Namespace Resolution Order

The command SHALL resolve the namespace using the following precedence (highest to lowest):

1. `--namespace` / `-n` flag value
2. Namespace from the ralph config
3. Default namespace of the active Kubernetes context

#### Scenario: Flag overrides config namespace

- GIVEN the ralph config specifies namespace `default`
- AND the user passes `-n staging`
- WHEN `ralph logs` runs
- THEN the `staging` namespace is used

#### Scenario: Config namespace used when no flag given

- GIVEN the ralph config specifies namespace `platform`
- AND no `--namespace` flag is given
- WHEN `ralph logs` runs
- THEN the `platform` namespace is used
