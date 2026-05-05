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

### Requirement: Spec and Flow References

A project MAY include `spec` and `flow` fields containing relative paths to the spec and flow documents the project is implementing.

#### Scenario: Spec and flow provided

- GIVEN a project that implements a documented feature
- WHEN the author adds `spec` and `flow` fields
- THEN each field contains a relative path from the project file to the corresponding document

#### Scenario: Spec and flow omitted

- GIVEN a project with no associated spec or flow
- WHEN the project is authored without `spec` or `flow` fields
- THEN the project is valid and the fields are simply absent

### Requirement: Requirements

A project MUST have a `requirements` field containing a list of one or more requirements. Each requirement MUST have `description`, `items`, and `passing` fields.

#### Scenario: Requirement with failing work

- GIVEN a requirement describing work that has not been implemented
- WHEN the author sets `passing: false`
- THEN the agent will select and implement this requirement

#### Scenario: Requirement already complete

- GIVEN a requirement describing work that is already done
- WHEN the author sets `passing: true`
- THEN the agent skips this requirement

#### Scenario: Requirement items

- GIVEN a requirement that describes a behavioral outcome
- WHEN the author writes the `items` list
- THEN each item is a specific, observable outcome the agent must achieve

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

### Requirement: Requirement Flows

A requirement MAY include a `flows` field containing a list of flows. Each flow MUST have a `module`, `code`, and `helpers` field. Flows are optionally copied from the flow document when the project is based on one.

The `module` field defines where the flow should be written. The `code` field contains the flow code itself. The `helpers` field is a list of helper functions the flow requires, each with `name`, `module`, and `description` properties.

#### Scenario: Flows copied from flow document

- GIVEN a project based on a flow document
- WHEN the author writes a requirement
- THEN relevant flows are copied from the flow document into the requirement's `flows` field

#### Scenario: Flows omitted

- GIVEN a requirement with no associated flow
- WHEN the project is authored without a `flows` field on that requirement
- THEN the requirement is valid and the field is simply absent
