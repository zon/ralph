# Workflow Workspace Specification

## Purpose

Shared container bootstrap for all `ralph workflow` subcommands: authenticate to GitHub, place AI credentials, configure git identity, clone the repository, check out the target branch, and symlink mounted files into the working directory.

## Requirements

### Requirement: GitHub Authentication

The system SHALL configure git HTTPS authentication from credentials mounted at `/secrets/github` before any git operations are performed. The system SHALL prefer GitHub App credentials over a stored token when both are available.

#### Scenario: App credentials preferred over token

- GIVEN GitHub App credentials (`app-id` and `private-key`) mounted at `/secrets/github`
- AND a `token` file is also present
- WHEN the workflow container starts
- THEN a GitHub App installation token is generated and git HTTPS authentication is configured for the target repo

#### Scenario: Token credentials

- GIVEN only a `token` file mounted at `/secrets/github`
- WHEN the workflow container starts
- THEN git HTTPS authentication is configured for the target repo with the stored token

### Requirement: AI Credentials

The system SHALL place OpenCode credentials in the expected location before any subcommand logic runs.

#### Scenario: OpenCode credential setup

- GIVEN OpenCode provider credentials mounted at `/secrets/opencode`
- WHEN the workflow container starts
- THEN the credentials are placed in the expected location for the AI agent to use

### Requirement: Git Identity

The system SHALL configure a git user identity before performing any commits.

#### Scenario: Git user configuration

- GIVEN `--bot-name` and `--bot-email` are provided (defaults: `ralph-zon[bot]` and `ralph-zon[bot]@users.noreply.github.com`)
- WHEN the workflow container starts
- THEN git is configured with those identity values for all subsequent commits

### Requirement: Repository Cloning

The system SHALL clone the target repository into the container workspace before any subcommand logic runs.

#### Scenario: Repository cloning

- GIVEN `GIT_BRANCH` environment variable is set to the branch to clone
- WHEN the workspace is prepared
- THEN the repository is cloned at that branch into `/workspace`

### Requirement: Branch Checkout

After cloning, the system SHALL check out the target branch for the subcommand. The calling subcommand provides the target branch. If none is specified, the clone branch remains checked out.

#### Scenario: Checkout existing branch

- GIVEN a target branch is provided and exists on the remote
- WHEN the workspace prepares the repository
- THEN the branch is fetched and checked out

#### Scenario: Create and checkout new branch

- GIVEN a target branch is provided but does not yet exist on the remote
- WHEN the workspace prepares the repository
- THEN a new local branch is created and checked out

#### Scenario: No separate branch requested

- GIVEN no target branch is provided beyond the clone branch
- WHEN the workspace prepares the repository
- THEN the clone branch remains checked out and no additional checkout is performed

### Requirement: Workspace Symlink Setup

The system SHALL symlink mounted ConfigMap and Secret files into the working directory after the branch is checked out, unless symlink setup is explicitly disabled by the calling subcommand.

#### Scenario: Linked ConfigMap

- GIVEN a ConfigMap mount with `link: true` in `.ralph/config.yaml`
- WHEN workspace setup runs inside the container
- THEN a symlink is created from the repo working directory path to the mounted path under `/workspace`

#### Scenario: Already-linked path

- GIVEN a symlink target already exists at the destination
- WHEN workspace setup runs
- THEN the existing symlink is left untouched and no error is raised

#### Scenario: Symlinks skipped

- GIVEN workspace is configured with symlink setup disabled
- WHEN workspace setup runs
- THEN no symlinks are created and no error is raised
