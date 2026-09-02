# Model Options Specification

## Purpose

Shared contract for every Ralph CLI command that prompts an AI model through opencode. These commands SHALL accept `--model` and `--variant` options to override the model and its provider-specific reasoning-effort variant from `.ralph/config.yaml`. Individual command specs link here for the shared behavior.

## Requirements

### Requirement: Covered commands

The commands that prompt an AI model are: `ralph run`, `ralph loop`, and `ralph workflow run`. Each SHALL accept the `--model` and `--variant` options described below.

#### Scenario: Every covered command exposes both options

- GIVEN any covered command
- WHEN its usage is shown
- THEN `--model` and `--variant` are listed among its options

---

### Requirement: `--model` selects the AI model

The system SHALL accept `--model <name>` on every covered command to select the AI model used for every prompt the command runs.

Model resolution SHALL follow this precedence (highest to lowest):

1. `--model` flag value
2. `model` in the top level of `.ralph/config.yaml`
3. `deepseek/deepseek-chat`

#### Scenario: Flag overrides the configured model

- GIVEN `model: anthropic/claude-sonnet-4-6` is set in `.ralph/config.yaml`
- AND the user passes `--model claude-opus-4-8`
- WHEN the command runs a prompt
- THEN `claude-opus-4-8` is used instead of the config value

#### Scenario: Config model used when no flag is passed

- GIVEN `model: anthropic/claude-sonnet-4-6` is set in `.ralph/config.yaml`
- AND no `--model` flag is passed
- WHEN the command runs a prompt
- THEN `anthropic/claude-sonnet-4-6` is used

#### Scenario: Default model used when flag and config are unset

- GIVEN neither `--model` nor `model` in `.ralph/config.yaml` is set
- WHEN the command runs a prompt
- THEN `deepseek/deepseek-chat` is used

---

### Requirement: `--variant` sets the reasoning effort

The system SHALL accept `--variant <hint>` on every covered command to pass a provider-specific reasoning effort hint (e.g., `high`, `max`, `minimal`) to opencode. When neither the flag nor the `variant` field in `.ralph/config.yaml` is set, the `--variant` option is omitted entirely from the opencode invocation.

Variant resolution SHALL follow this precedence (highest to lowest):

1. `--variant` flag value
2. `variant` in the top level of `.ralph/config.yaml`
3. Omitted from the opencode invocation

#### Scenario: Flag passes the variant to opencode

- GIVEN the user passes `--variant high`
- WHEN the command runs a prompt
- THEN `--variant high` is included in the opencode invocation

#### Scenario: Config variant used when no flag is passed

- GIVEN `variant: max` is set in `.ralph/config.yaml`
- AND no `--variant` flag is passed
- WHEN the command runs a prompt
- THEN `--variant max` is included in the opencode invocation

#### Scenario: Variant omitted when flag and config are unset

- GIVEN `variant` is not set in `.ralph/config.yaml`
- AND no `--variant` flag is passed
- WHEN the command runs a prompt
- THEN the `--variant` option is omitted from the opencode invocation

---

### Requirement: Commands that prompt a model without the options

`ralph validate` also prompts an AI model to repair a malformed project file, but it does not expose the `--model` and `--variant` options. It resolves its model from config alone, as defined in [validate.md](validate.md).

#### Scenario: Validate runs without the options

- GIVEN the user runs `ralph validate <file>`
- WHEN a repair prompt runs
- THEN no `--model` or `--variant` flag is required, and the model comes from config
