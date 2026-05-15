# Project Format Specification

## Purpose

Define the format of ralph project YAML files and the rules for writing them, and specify that the documentation describing this format is written as a Claude Code and opencode skill.

## Requirements

### Requirement: Slug

A project MUST have a `slug` field containing a lowercase, hyphen-separated string that uniquely identifies the project.

#### Scenario: Valid slug

- GIVEN a project with `slug: fix-pagination`
- WHEN the file is authored
- THEN the slug is accepted as the project identifier

#### Scenario: Invalid slug format

- GIVEN a project with a slug containing uppercase letters, spaces, or special characters
- WHEN the project is validated
- THEN an error is reported describing the invalid format

### Requirement: Title

A project MUST have a `title` field containing a brief one-line description of what the project is doing.

#### Scenario: Title describes the work

- GIVEN a project implementing a CSV export feature
- WHEN the author writes the title
- THEN the title reads something like `Add CSV export to the reports API`

### Requirement: Spec and Orchestration References

A project MAY include `spec` and `orchestration` fields containing relative paths to the spec and orchestration documents the project is implementing.

#### Scenario: Spec and orchestration provided

- GIVEN a project that implements a documented feature
- WHEN the author adds `spec` and `orchestration` fields
- THEN each field contains a relative path from the project file to the corresponding document

#### Scenario: Spec and orchestration omitted

- GIVEN a project with no associated spec or orchestration
- WHEN the project is authored without `spec` or `orchestration` fields
- THEN the project is valid and the fields are simply absent

### Requirement: Requirements

A project MUST have a `requirements` field containing a list of one or more requirements. Each requirement MUST have `description` and `passing` fields, and MUST have at least one of `items`, `scenarios`, or `orchestrations`.

#### Scenario: Requirement with failing work

- GIVEN a requirement describing work that has not been implemented
- WHEN the author sets `passing: false`
- THEN the agent will select and implement this requirement

#### Scenario: Requirement already complete

- GIVEN a requirement describing work that is already done
- WHEN the author sets `passing: true`
- THEN the agent skips this requirement

#### Scenario: Requirement items

- GIVEN a requirement with work that falls outside the associated spec and orchestration
- WHEN the author writes the `items` list
- THEN each item is a specific, observable outcome the agent must achieve, free of architecture decisions such as package names, struct names, or implementation strategies

#### Scenario: Items omitted when scenarios or orchestrations are present

- GIVEN a requirement with `scenarios` or `orchestrations` but no `items`
- WHEN the project is validated
- THEN the requirement is valid because at least one content field is present

#### Scenario: No content fields

- GIVEN a requirement with no `items`, `scenarios`, or `orchestrations`
- WHEN the project is validated
- THEN an error is reported requiring at least one content field

### Requirement: Requirement Scenarios

A requirement MAY include a `scenarios` field containing a list of scenarios. Each scenario MUST have a `title` and an `items` list. Scenarios are copied from the spec document when the project is based on one.

#### Scenario: Scenarios copied from spec

- GIVEN a project based on a spec document
- WHEN the author writes a requirement
- THEN relevant scenarios are copied from the spec into the requirement's `scenarios` field

#### Scenario: Scenarios omitted

- GIVEN a requirement with no associated spec scenarios
- WHEN the project is authored without a `scenarios` field on that requirement
- THEN the requirement is valid and the field is simply absent

### Requirement: Helper Requirements

Each helper function defined in an orchestration document's `helpers` list MUST have a corresponding requirement in the project. The helper requirement MUST include an `orchestrations` entry for the helper with `name` and optionally `module` and `description`, but MUST NOT include `code` or `helpers` on that orchestration. Scenarios from the spec that directly relate to the helper MUST be copied into the requirement. Items MUST be used to fill in any gaps not covered by scenarios or the orchestration.

#### Scenario: Helper gets its own requirement

- GIVEN an orchestration document that lists `buildCSV` as a helper of `ExportReport`
- WHEN the author writes the project
- THEN a separate requirement exists for `buildCSV` with an orchestration entry containing `name` and optionally `module` and `description`

#### Scenario: Helper orchestration omits code and helpers

- GIVEN a helper requirement with an `orchestrations` entry
- WHEN the author writes the orchestration
- THEN the orchestration does not include `code` or `helpers` properties

#### Scenario: Spec scenarios copied to helper requirement

- GIVEN a spec scenario that directly describes the behavior of a helper function
- WHEN the author writes the helper requirement
- THEN that scenario is copied into the requirement's `scenarios` field

#### Scenario: Items fill gaps for helper

- GIVEN a helper requirement where the spec and orchestration do not fully describe the expected behavior
- WHEN the author writes the helper requirement
- THEN `items` are added to cover the remaining behavioral expectations

### Requirement: Skill-Format Documentation

The documentation describing the project file format MUST be written as a skill file compatible with Claude Code and opencode. The [Agent Skills | OpenCode](https://opencode.ai/docs/skills/) page defines the format reference.

The skill file MUST begin with YAML frontmatter containing a `name` and `description` field. The body contains the documentation content in markdown.

#### Scenario: Documentation loaded as a skill

- GIVEN an agent that supports Claude Code or opencode skills
- WHEN it loads the project documentation skill
- THEN it can read the full project format and writing rules on demand

#### Scenario: Frontmatter is valid

- GIVEN the documentation skill file
- WHEN the frontmatter is parsed
- THEN `name` is a lowercase hyphen-separated identifier and `description` is a concise one-line summary

### Requirement: Requirement Orchestrations

A requirement MAY include an `orchestrations` field containing a list of orchestrations. Each orchestration MUST have a `name` field containing the method name. The `module`, `code`, `helpers`, and `description` fields are all optional. Orchestrations are optionally copied from the orchestration document when the project is based on one.

The `name` field identifies the method. The `module` field defines where the orchestration should be written. The `code` field contains the orchestration code itself — including package names, function signatures, struct names, and implementation strategies. The `helpers` field is a list of helper functions the orchestration requires, each with `name`, `module`, and `description` properties. The `description` field provides a short summary of the orchestration's purpose.

Orchestrations are the correct place to specify architecture. Items must not contain architecture decisions; orchestrations must.

#### Scenario: Orchestrations copied from orchestration document

- GIVEN a project based on an orchestration document
- WHEN the author writes a requirement
- THEN relevant orchestrations are copied from the orchestration document into the requirement's `orchestrations` field

#### Scenario: Orchestrations omitted

- GIVEN a requirement with no associated orchestration
- WHEN the project is authored without an `orchestrations` field on that requirement
- THEN the requirement is valid and the field is simply absent
