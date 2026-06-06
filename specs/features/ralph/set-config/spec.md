# Set Config Specification

## Purpose

One-shot setup of all Kubernetes credentials required for ralph remote execution on Argo Workflows.

## Requirements

### Requirement: Sequential Credential Setup

The system SHALL run credential setup steps in order via `ralph set config`: (1) resolve Kubernetes context, (2) validate and write GitHub App credentials, (3) read and write OpenCode credentials. If any step fails, the command SHALL exit immediately without proceeding to subsequent steps.

#### Scenario: All credentials configured successfully

- GIVEN a valid GitHub App private key `.pem` file and an OpenCode `auth.json` present on the local machine
- WHEN the user runs `ralph set config --github-key <key.pem>`
- THEN the GitHub App credentials are validated against the GitHub API
- AND a Kubernetes Secret is written with `app-id` and `private-key`
- AND the OpenCode `auth.json` is written as a Kubernetes Secret
- AND the command exits with success

#### Scenario: GitHub credential failure halts setup

- GIVEN an invalid or empty GitHub App private key file
- WHEN the user runs `ralph set config --github-key <key.pem>`
- THEN an error is returned after the GitHub credential step
- AND the OpenCode credential step is not attempted

#### Scenario: Missing OpenCode credentials

- GIVEN a valid GitHub App private key and no `auth.json` at `~/.local/share/opencode/auth.json`
- WHEN the user runs `ralph set config --github-key <key.pem>`
- THEN the GitHub credential step completes successfully
- AND an error is returned for the missing OpenCode credentials

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace, falling back to the current kubeconfig context and its default namespace.

#### Scenario: Context override

- GIVEN `--context staging --namespace argo` is passed
- WHEN `ralph set config` runs
- THEN both credential secrets are written to the `staging` context in the `argo` namespace

### Requirement: GitHub Key Flag

The command SHALL accept an optional `--github-key` flag pointing to an existing `.pem` file containing the GitHub App private key. If the flag is omitted, the command SHALL check whether the GitHub App credentials secret already exists in Kubernetes. If the secret exists, the existing key is reused. If the secret does not exist, an error is returned.

#### Scenario: Flag provided — secret written with new key

- GIVEN `--github-key <key.pem>` is provided
- WHEN the user runs `ralph set config`
- THEN the key is validated against the GitHub API
- AND the credentials secret is created or updated with the new key

#### Scenario: Flag omitted — existing secret reused

- GIVEN `--github-key` is not provided
- AND the GitHub App credentials secret already exists in the target namespace
- WHEN the user runs `ralph set config`
- THEN the GitHub credential step succeeds without reading a local key file

#### Scenario: Flag omitted — no existing secret

- GIVEN `--github-key` is not provided
- AND no GitHub App credentials secret exists in the target namespace
- WHEN the user runs `ralph set config`
- THEN an error is returned before any steps are attempted
