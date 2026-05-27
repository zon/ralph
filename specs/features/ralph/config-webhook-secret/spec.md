# Config Webhook Secret Specification

## Purpose

Generate per-repo webhook secrets, register GitHub webhooks, and store the secrets in Kubernetes for use by the ralph webhook service.

## Requirements

### Requirement: Webhook Secret Provisioning

The system SHALL generate and store per-repo webhook secrets and register GitHub webhooks via `ralph config webhook-secret`.

#### Scenario: Secret provisioning

- GIVEN the `webhook-config` ConfigMap exists in the target namespace with at least one repo
- WHEN the user runs `ralph config webhook-secret`
- THEN per-repo webhook secrets are generated
- AND GitHub webhooks are registered for each repo pointing at the ralph webhook ingress
- AND the secrets are stored in the `webhook-secrets` Kubernetes Secret in the `ralph-webhook` namespace

#### Scenario: Missing webhook config

- GIVEN the `webhook-config` ConfigMap does not exist
- WHEN the user runs `ralph config webhook-secret`
- THEN an error is returned instructing the user to run `ralph config webhook` first

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace.

#### Scenario: Context override

- GIVEN `--context production --namespace ralph-webhook` is passed
- WHEN `ralph config webhook-secret` runs
- THEN the specified Kubernetes context and namespace are used instead of defaults
