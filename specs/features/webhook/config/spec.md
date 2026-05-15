# Webhook Config Specification

## Purpose

Configure the ralph-webhook service with per-repo settings, webhook secrets, and global defaults.

## Requirements

### Requirement: Configuration Loading

The service MUST load a YAML app config file and a separate YAML secrets file at startup.

#### Scenario: Paths from flags

- GIVEN `--config` and `--secrets` flags are provided
- WHEN the service starts
- THEN the specified files are loaded

#### Scenario: Paths from environment

- GIVEN `WEBHOOK_CONFIG` and `WEBHOOK_SECRETS` environment variables are set
- WHEN the service starts without flags
- THEN the environment variable paths are used

#### Scenario: Missing config path

- GIVEN neither `--config` nor `WEBHOOK_CONFIG` is provided
- WHEN the service starts
- THEN an error is returned and the service does not start

### Requirement: Config Validation

The service MUST validate that every configured repository has a namespace and a corresponding webhook secret.

#### Scenario: Valid config

- GIVEN all repos have `namespace` set and matching secrets in the secrets file
- WHEN the service starts
- THEN the config is accepted and the server begins listening

#### Scenario: Missing namespace

- GIVEN a repo in `repos` with no `namespace` field
- WHEN the service starts
- THEN an error is returned identifying the repo

#### Scenario: Missing webhook secret

- GIVEN a repo in `repos` with no matching entry in the secrets file
- WHEN the service starts
- THEN an error is returned identifying the repo

### Requirement: Per-Repo Configuration

The service SHALL support per-repo allowlists, ignorelists, and Kubernetes namespaces.

#### Scenario: Namespace routing

- GIVEN a repo with `namespace: argo-staging`
- WHEN a workflow is submitted for that repo
- THEN the workflow is submitted to the `argo-staging` namespace

#### Scenario: Allowed users

- GIVEN `allowedUsers: [alice, bob]` for a repo
- WHEN a comment arrives from `carol`
- THEN the event is dropped

#### Scenario: Ignored users

- GIVEN `ignoredUsers: [dependabot]` for a repo
- WHEN a comment arrives from `dependabot`
- THEN the event is dropped regardless of allowedUsers

### Requirement: Global Ralph User

The service SHALL globally ignore all events from the configured `ralphUser`, regardless of per-repo settings.

#### Scenario: Bot self-ignoring

- GIVEN `ralphUser: ralph-bot`
- WHEN a comment is received from `ralph-bot` on any repo
- THEN the event is always dropped

### Requirement: Custom Instructions

The service SHOULD allow overriding the default AI instructions for comment replies and merge operations.

#### Scenario: Comment instructions override

- GIVEN `commentInstructionsFile` points to a custom markdown file
- WHEN a comment event is dispatched
- THEN the custom file's content is used as the AI prompt template instead of the built-in default

#### Scenario: Merge instructions override

- GIVEN `mergeInstructionsFile` points to a custom markdown file
- WHEN a merge event is dispatched
- THEN the custom file's content is used as the AI merge prompt template

### Requirement: Container Image Configuration

The service SHOULD allow configuring the container image used for submitted Argo Workflows.

#### Scenario: Custom image

- GIVEN `imageRepository` and `imageTag` are set in the app config
- WHEN a workflow is generated
- THEN the specified image is used in the workflow spec

#### Scenario: Default image

- GIVEN `imageRepository` and `imageTag` are not set
- WHEN a workflow is generated
- THEN `ghcr.io/zon/ralph:latest` is used
