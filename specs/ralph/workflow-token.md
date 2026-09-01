# Workflow Token Specification

## Purpose

Configure git HTTPS authentication inside Argo Workflow containers from GitHub App credentials or a stored personal access token, preferring the App credentials when both are present.

## Requirements

### Requirement: Git Authentication

The system SHALL configure git HTTPS authentication via `ralph workflow token` from credentials in a mounted secrets directory. The system SHALL prefer GitHub App credentials over a stored token when both are present. If `app-id` and `private-key` are present, the system SHALL exchange them for a short-lived installation token via the GitHub API. If only a `token` file is present, the system SHALL configure git with the stored token directly.

#### Scenario: App credentials present

- GIVEN GitHub App credentials (`app-id` and `private-key`) are present at `--secrets-dir` (default: `/secrets/github`)
- AND the App is installed on the target repository
- WHEN the user runs `ralph workflow token`
- THEN a GitHub App installation token is generated
- AND git HTTPS authentication is configured so subsequent git operations authenticate as the App

#### Scenario: App credentials preferred over token

- GIVEN both App credentials and a `token` file are present at the secrets directory
- WHEN the user runs `ralph workflow token`
- THEN the App installation token flow is used

#### Scenario: Token credentials present

- GIVEN only a `token` file is present at the secrets directory
- WHEN the user runs `ralph workflow token`
- THEN git HTTPS authentication is configured with the stored token

#### Scenario: Missing credentials

- GIVEN the secrets directory does not exist or is missing required files
- WHEN the user runs `ralph workflow token`
- THEN an error is returned and no git configuration is written

#### Scenario: Invalid credentials

- GIVEN the private key or app ID in the secrets directory is malformed
- WHEN the user runs `ralph workflow token`
- THEN an error is returned after the GitHub API rejects the JWT

### Requirement: Repository Targeting

The command SHALL accept `--owner` and `--repo` flags to identify the target repository. If omitted, owner and repo SHALL be auto-detected from the git remote of the current working directory.

#### Scenario: Flags provided

- GIVEN `--owner myorg --repo myrepo` is passed
- WHEN `ralph workflow token` runs
- THEN the target repository is resolved to `myorg/myrepo`
- AND git HTTPS authentication is configured for that repository

#### Scenario: Auto-detection from git remote

- GIVEN `--owner` and `--repo` are not provided
- AND the current directory is a git repository with a GitHub remote
- WHEN `ralph workflow token` runs
- THEN the owner and repo are inferred from the remote URL
