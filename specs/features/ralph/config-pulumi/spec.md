# Config Pulumi Specification

## Purpose

Store a Pulumi access token as a Kubernetes Secret for use by ralph's remote execution on Argo Workflows.

## Requirements

### Requirement: Pulumi Credential Provisioning

The system SHALL store a Pulumi access token as a Kubernetes Secret via `ralph config pulumi`.

#### Scenario: Token from argument

- GIVEN a token is passed as a positional argument
- WHEN the user runs `ralph config pulumi <token>`
- THEN the token is stored in the `pulumi` Kubernetes Secret without prompting

#### Scenario: Token from environment or interactive prompt

- GIVEN no token argument is passed
- WHEN the user runs `ralph config pulumi`
- THEN if `PULUMI_ACCESS_TOKEN` is set in the environment it is used without further prompting
- AND if it is not set the user is prompted to enter the token interactively

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace.

#### Scenario: Context override

- GIVEN `--context production --namespace argo` is passed
- WHEN `ralph config pulumi` runs
- THEN the specified Kubernetes context and namespace are used instead of defaults
