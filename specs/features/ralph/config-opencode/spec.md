# Config OpenCode Specification

## Purpose

Store OpenCode AI provider credentials as a Kubernetes Secret for use by ralph's remote execution on Argo Workflows.

## Requirements

### Requirement: OpenCode Credential Provisioning

The system SHALL store OpenCode AI provider tokens as a Kubernetes Secret via `ralph config opencode`.

#### Scenario: Successful provisioning

- GIVEN `~/.local/share/opencode/auth.json` contains valid provider credentials
- WHEN the user runs `ralph config opencode`
- THEN the full `auth.json` content is stored as the `opencode` Kubernetes Secret

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace.

#### Scenario: Context override

- GIVEN `--context production --namespace argo` is passed
- WHEN `ralph config opencode` runs
- THEN the specified Kubernetes context and namespace are used instead of defaults
