# Config Provider Specification

## Purpose

Collect an AI provider API key, write it to `.ralph/auth.yaml`, and store all current credentials as a Kubernetes Secret in the ralph namespace for remote execution on Argo Workflows.

## Requirements

### Requirement: Single Provider Credential Provisioning

The command SHALL accept a provider name as a required positional argument and prompt for that provider's API key. The key is written to `.ralph/auth.yaml` under the provider name, and the full contents of `.ralph/auth.yaml` are then stored as the `provider-credentials` Kubernetes Secret.

Supported providers:

| Provider | Key in auth.yaml |
|----------|-----------------|
| Anthropic | `anthropic` |
| Google | `google` |
| DeepSeek | `deepseek` |

#### Scenario: Key written and secret updated

- GIVEN the user runs `ralph config provider anthropic`
- AND enters a valid API key when prompted
- THEN the key is written to `.ralph/auth.yaml` under `anthropic`
- AND the full contents of `.ralph/auth.yaml` are stored as the `provider-credentials` Kubernetes Secret

#### Scenario: Other providers preserved

- GIVEN `.ralph/auth.yaml` already contains a `google` key
- WHEN the user runs `ralph config provider anthropic` and enters a key
- THEN the `anthropic` key is added and the existing `google` key is preserved in `.ralph/auth.yaml`
- AND both keys are included in the Kubernetes Secret

#### Scenario: Blank key rejected

- GIVEN the user runs `ralph config provider anthropic`
- AND leaves the prompt blank
- THEN an error is returned and no changes are written

#### Scenario: Unknown provider rejected

- GIVEN the user runs `ralph config provider foobar`
- WHEN the command starts
- THEN an error is returned: `unknown provider: foobar`

---

### Requirement: Kubernetes Context Targeting

The command SHALL accept `--context` and `--namespace` flags to target a specific cluster and namespace for the Kubernetes Secret.

#### Scenario: Context override

- GIVEN `--context production --namespace argo` is passed
- WHEN `ralph config provider anthropic` runs
- THEN the specified Kubernetes context and namespace are used instead of defaults
