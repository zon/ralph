# Models Specification

## Purpose

List available AI models from providers that have API keys configured in `.ralph/auth.yaml`, so users can discover valid model identifiers to use with `--model`.

## Requirements

### Requirement: List models from configured providers

The command SHALL read provider API keys from `.ralph/auth.yaml`, query each configured provider's API for available models, and print them to stdout in `provider/model-id` format, one per line. Only providers with a key present in `.ralph/auth.yaml` are queried.

Supported providers:

| Provider | Key in auth.yaml |
|----------|-----------------|
| Anthropic | `anthropic` |
| Google | `google` |
| DeepSeek | `deepseek` |

#### Scenario: Models listed for configured providers

- GIVEN `.ralph/auth.yaml` contains keys for `anthropic` and `google`
- AND no key is present for `deepseek`
- WHEN the user runs `ralph models`
- THEN models from Anthropic and Google are printed in `provider/model-id` format
- AND no models from DeepSeek appear in the output

#### Scenario: No providers configured

- GIVEN `.ralph/auth.yaml` does not exist or contains no provider keys
- WHEN the user runs `ralph models`
- THEN an error is returned indicating no provider credentials are configured

#### Scenario: Provider API call fails

- GIVEN a provider key is present in `.ralph/auth.yaml` but the API call returns an error
- WHEN the user runs `ralph models`
- THEN a warning is printed for that provider
- AND models from other reachable providers are still listed

---

### Requirement: Optional provider filter

The command SHALL accept an optional positional argument to filter output to a single provider.

#### Scenario: Provider filter applied

- GIVEN `.ralph/auth.yaml` contains keys for `anthropic` and `google`
- WHEN the user runs `ralph models anthropic`
- THEN only Anthropic models are printed
- AND Google models are not included in the output

#### Scenario: Unknown provider name

- GIVEN the user runs `ralph models foobar`
- WHEN the command runs
- THEN an error is returned: `unknown provider: foobar`
