# Config Specification

## Purpose

Provision Kubernetes Secrets required for ralph's remote execution on Argo Workflows, and manage workflow state via list and stop commands.

## Requirements

### Requirement: GitHub App Credential Provisioning

The system SHALL store GitHub App credentials as a Kubernetes Secret via `ralph config github`.

#### Scenario: Successful provisioning

- GIVEN a valid GitHub App private key `.pem` file
- WHEN the user runs `ralph config github <key.pem>`
- THEN the private key and app ID are stored in the `github` Kubernetes Secret in the configured namespace
- AND the credentials are validated against the GitHub API before storage

#### Scenario: Invalid key

- GIVEN an empty or invalid private key file
- WHEN the user runs `ralph config github <key.pem>`
- THEN an error is returned and no secret is written

### Requirement: OpenCode Credential Provisioning

The system SHALL store OpenCode AI provider tokens as a Kubernetes Secret via `ralph config opencode`.

#### Scenario: Successful provisioning

- GIVEN `~/.local/share/opencode/auth.json` contains valid provider credentials
- WHEN the user runs `ralph config opencode`
- THEN all configured AI provider tokens are stored in the `opencode` Kubernetes Secret

### Requirement: Pulumi Credential Provisioning

The system SHALL store a Pulumi access token as a Kubernetes Secret via `ralph config pulumi`.

#### Scenario: Token from argument

- GIVEN a token is passed as a positional argument
- WHEN the user runs `ralph config pulumi <token>`
- THEN the token is stored in the `pulumi` Kubernetes Secret without prompting

#### Scenario: Token from environment

- GIVEN `PULUMI_ACCESS_TOKEN` is set in the environment
- WHEN the user runs `ralph config pulumi`
- THEN the environment variable value is used without prompting

#### Scenario: Interactive prompt

- GIVEN no argument and no environment variable
- WHEN the user runs `ralph config pulumi`
- THEN the user is prompted to enter the token interactively

### Requirement: Webhook Config Provisioning

The system SHALL provision webhook configuration and secrets into Kubernetes via `ralph config webhook` and `ralph config webhook-secret`.

#### Scenario: Config provisioning

- GIVEN a webhook config YAML file
- WHEN `ralph config webhook <file>` is run
- THEN the config is stored as a Kubernetes Secret in the target namespace

### Requirement: Kubernetes Context Targeting

All `config` subcommands SHOULD accept `--context` and `--namespace` flags to target a specific cluster and namespace.

#### Scenario: Context override

- GIVEN `--context production --namespace argo` is set
- WHEN any `ralph config` command runs
- THEN the specified Kubernetes context and namespace are used instead of defaults

### Requirement: Workflow Listing

The system SHALL list active Argo Workflows via `ralph list`.

#### Scenario: List workflows

- GIVEN Argo Workflows are running in the configured namespace
- WHEN the user runs `ralph list`
- THEN the active workflow names and statuses are printed

### Requirement: Workflow Stopping

The system SHALL stop a named Argo Workflow via `ralph stop <workflow-name>`.

#### Scenario: Stop workflow

- GIVEN an active workflow named `my-workflow`
- WHEN the user runs `ralph stop my-workflow`
- THEN the workflow is terminated in the configured namespace
