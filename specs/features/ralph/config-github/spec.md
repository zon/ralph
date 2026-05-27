# Config GitHub Specification

## Purpose

Store GitHub App credentials as a Kubernetes Secret for use by ralph's remote execution on Argo Workflows.

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

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace.

#### Scenario: Context override

- GIVEN `--context production --namespace argo` is passed
- WHEN `ralph config github` runs
- THEN the specified Kubernetes context and namespace are used instead of defaults
