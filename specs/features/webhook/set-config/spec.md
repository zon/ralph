# Webhook Set Config Specification

## Purpose

One-shot setup of all Kubernetes resources required for the ralph-webhook service to receive and handle GitHub webhook events.

## Requirements

### Requirement: Sequential Resource Setup

The system SHALL run setup steps in order via `ralph-webhook set config`: (1) resolve Kubernetes context, (2) build and write the webhook-config ConfigMap, (3) generate per-repo HMAC webhook secrets, (4) register GitHub webhooks for each configured repo, (5) write the webhook-secrets Kubernetes Secret. If any step fails, the command SHALL exit immediately without proceeding to subsequent steps.

#### Scenario: Full setup completes successfully

- GIVEN a reachable Kubernetes cluster and valid GitHub credentials
- WHEN the user runs `ralph-webhook set config`
- THEN the webhook-config ConfigMap is written with repo list and app settings
- AND per-repo HMAC secrets are generated
- AND GitHub webhooks are registered for each repo pointing to the webhook service ingress
- AND the webhook-secrets Kubernetes Secret is written
- AND the command exits with success

#### Scenario: ConfigMap write failure halts setup

- GIVEN the Kubernetes API is unreachable
- WHEN the user runs `ralph-webhook set config`
- THEN an error is returned after the ConfigMap step
- AND no webhook secrets are generated or registered

#### Scenario: Webhook registration failure is non-fatal

- GIVEN one repo's GitHub webhook registration fails (e.g. insufficient permissions)
- WHEN `ralph-webhook set config` runs the registration step
- THEN a warning is emitted for the failing repo
- AND registration continues for remaining repos
- AND the webhook-secrets Secret is still written

### Requirement: Partial Config Seed

The command SHALL accept an optional `--partial-config` flag pointing to a partial AppConfig YAML file. When provided, its values SHALL be merged into the ConfigMap as a starting point before auto-detection fills remaining fields.

#### Scenario: Existing ConfigMap merged

- GIVEN a `webhook-config` ConfigMap already exists in the target namespace
- WHEN the user runs `ralph-webhook set config`
- THEN the existing ConfigMap values are read as the base
- AND auto-detected and provided values are merged on top
- AND the merged result is written back

#### Scenario: Collaborators auto-populated

- GIVEN the current directory is a git repository with a GitHub remote
- WHEN the user runs `ralph-webhook set config`
- THEN repository collaborators are fetched from GitHub
- AND the collaborator list is written into the webhook-config ConfigMap as allowed users

#### Scenario: Partial config provided

- GIVEN a YAML file specifying a subset of AppConfig fields
- WHEN the user runs `ralph-webhook set config --partial-config partial.yaml`
- THEN the provided values are merged on top of any existing ConfigMap values
- AND auto-detected values (repo owner, name, namespace) fill any unset fields

#### Scenario: Partial config file unreadable

- GIVEN `--partial-config` points to a file that does not exist or cannot be parsed
- WHEN the user runs `ralph-webhook set config --partial-config bad.yaml`
- THEN a warning is emitted and setup proceeds without the partial config

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags. The namespace SHALL default to `ralph-webhook`.

#### Scenario: Default namespace

- GIVEN no `--namespace` flag is provided
- WHEN `ralph-webhook set config` runs
- THEN all resources are written to the `ralph-webhook` namespace

#### Scenario: Namespace override

- GIVEN `--namespace staging-webhook` is passed
- WHEN `ralph-webhook set config` runs
- THEN all resources are written to the `staging-webhook` namespace
