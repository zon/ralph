# Config Webhook Specification

## Purpose

Provision webhook configuration into a Kubernetes ConfigMap for use by the ralph webhook service.

## Requirements

### Requirement: Webhook Config Provisioning

The system SHALL provision webhook configuration into Kubernetes via `ralph config webhook`.

#### Scenario: Config provisioning

- GIVEN the current directory is a git repository with a GitHub remote
- WHEN the user runs `ralph config webhook`
- THEN the repo owner, name, and namespace are auto-detected
- AND repository collaborators are fetched from GitHub to populate allowed users
- AND any existing `webhook-config` ConfigMap in the target namespace is read and merged with the new values
- AND the merged config is written to the `webhook-config` ConfigMap in the `ralph-webhook` namespace

#### Scenario: Config provisioning with partial config override

- GIVEN a partial AppConfig YAML file is provided via `--config <path>`
- WHEN the user runs `ralph config webhook --config <path>`
- THEN the partial config is merged on top of the existing ConfigMap values before writing

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace.

#### Scenario: Context override

- GIVEN `--context production --namespace ralph-webhook` is passed
- WHEN `ralph config webhook` runs
- THEN the specified Kubernetes context and namespace are used instead of defaults
