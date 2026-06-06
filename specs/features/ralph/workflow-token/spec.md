# Workflow Token Specification

## Purpose

Generate a GitHub App installation token and configure git HTTPS authentication for use inside Argo Workflow containers.

## Requirements

### Requirement: Installation Token Generation

The system SHALL generate a GitHub App installation token via `ralph workflow token` by reading App credentials from a mounted secrets directory, exchanging them for a short-lived installation token via the GitHub API, and configuring git HTTPS authentication with that token.

#### Scenario: Successful token generation

- GIVEN GitHub App credentials (`app-id` and `private-key`) are present at `--secrets-dir` (default: `/secrets/github`)
- AND the App is installed on the target repository
- WHEN the user runs `ralph workflow token`
- THEN a GitHub App installation token is generated
- AND git HTTPS authentication is configured so subsequent git operations authenticate as the App

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
- THEN the installation token is scoped to that repository

#### Scenario: Auto-detection from git remote

- GIVEN `--owner` and `--repo` are not provided
- AND the current directory is a git repository with a GitHub remote
- WHEN `ralph workflow token` runs
- THEN the owner and repo are inferred from the remote URL
