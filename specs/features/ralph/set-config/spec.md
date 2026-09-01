# Set Config Specification

## Purpose

One-shot setup of all Kubernetes credentials required for ralph remote execution on Argo Workflows. The shared `--context` and `--namespace` targeting contract is defined in [kubectl/spec.md](../kubectl/spec.md).

## Requirements

### Requirement: Sequential Credential Setup

The system SHALL run credential setup steps in order via `ralph set config`: (1) resolve Kubernetes context, (2) validate and write GitHub credentials, (3) read and write OpenCode credentials. If any step fails, the command SHALL exit immediately without proceeding to subsequent steps.

#### Scenario: App credentials configured successfully

- GIVEN a valid GitHub App private key `.pem` file and an OpenCode `auth.json` present on the local machine
- WHEN the user runs `ralph set config --github-key <key.pem>`
- THEN the GitHub App credentials are validated against the GitHub API
- AND a Kubernetes Secret is written with `app-id` and `private-key`
- AND the OpenCode `auth.json` is written as a Kubernetes Secret
- AND the command exits with success

#### Scenario: Token credentials configured successfully

- GIVEN a GitHub personal access token and an OpenCode `auth.json` present on the local machine
- WHEN the user runs `ralph set config --github-token <token>`
- THEN a Kubernetes Secret is written with `token`
- AND the OpenCode `auth.json` is written as a Kubernetes Secret
- AND the command exits with success

#### Scenario: GitHub credential failure halts setup

- GIVEN an invalid or empty GitHub App private key file
- WHEN the user runs `ralph set config --github-key <key.pem>`
- THEN an error is returned after the GitHub credential step
- AND the OpenCode credential step is not attempted

#### Scenario: Missing OpenCode credentials

- GIVEN valid GitHub credentials and no `auth.json` at `~/.local/share/opencode/auth.json`
- WHEN the user runs `ralph set config --github-key <key.pem>`
- THEN the GitHub credential step completes successfully
- AND an error is returned for the missing OpenCode credentials

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace, falling back to the current kubeconfig context and its default namespace. See [kubectl/spec.md](../kubectl/spec.md) for the shared targeting contract.

#### Scenario: Context override

- GIVEN `--context staging --namespace argo` is passed
- WHEN `ralph set config` runs
- THEN both credential secrets are written to the `staging` context in the `argo` namespace

### Requirement: GitHub Key Flag

The command SHALL accept an optional `--github-key` flag pointing to an existing `.pem` file containing the GitHub App private key. When provided, the key SHALL be validated against the GitHub API and written to the credentials Secret as `app-id` and `private-key`. The flag SHALL be mutually exclusive with `--github-token`.

#### Scenario: Flag provided writes the secret with a new key

- GIVEN `--github-key <key.pem>` is provided
- WHEN the user runs `ralph set config`
- THEN the key is validated against the GitHub API
- AND the credentials secret is created or updated with the new key

#### Scenario: Flag omitted reuses the existing secret

- GIVEN `--github-key` is not provided
- AND the GitHub credentials secret already exists in the target namespace
- WHEN the user runs `ralph set config`
- THEN the GitHub credential step succeeds without reading a local key file

### Requirement: GitHub Token Flag

The command SHALL accept an optional `--github-token` flag containing a GitHub personal access token. When provided, the token SHALL be written to the `token` key of the credentials Secret without validation. The flag SHALL be mutually exclusive with `--github-key`.

When neither `--github-key` nor `--github-token` is provided and no credentials Secret exists, the command SHALL fall back to the token stored by the `gh` CLI login, then to the `GITHUB_TOKEN` environment variable. If a fallback token is found, it is written to the Secret. If none is found, an error is returned.

#### Scenario: Flag provided writes the token secret

- GIVEN `--github-token <token>` is provided
- WHEN the user runs `ralph set config`
- THEN the credentials secret is created or updated with the `token` key set to the token

#### Scenario: Both flags are mutually exclusive

- GIVEN both `--github-key <key.pem>` and `--github-token <token>` are provided
- WHEN the user runs `ralph set config`
- THEN an error is returned before any step is attempted

#### Scenario: No flags falls back to gh login token

- GIVEN neither flag is provided
- AND no GitHub credentials secret exists in the target namespace
- AND `gh` is authenticated
- WHEN the user runs `ralph set config`
- THEN the token is read from the `gh` CLI login
- AND the credentials secret is created with the `token` key set to that token

#### Scenario: No flags falls back to environment token

- GIVEN neither flag is provided
- AND no GitHub credentials secret exists in the target namespace
- AND `gh` is not authenticated
- AND `GITHUB_TOKEN` is set
- WHEN the user runs `ralph set config`
- THEN the credentials secret is created with the `token` key set to the environment token

#### Scenario: No flags and no credentials available

- GIVEN neither flag is provided
- AND no GitHub credentials secret exists in the target namespace
- AND no `gh` token or `GITHUB_TOKEN` is available
- WHEN the user runs `ralph set config`
- THEN an error is returned before any steps are attempted
