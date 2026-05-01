# Project Specification

## Purpose

Define the YAML format for ralph project files, which describe work for AI agents to implement.

## Requirements

### Requirement: Project File Format

A project file MUST be a valid YAML file with `name`, `description`, and `requirements` fields.

#### Scenario: Valid project

- GIVEN a YAML file with `name`, `description`, and at least one requirement with `category`, `description`, `items`, and `passing`
- WHEN the project is loaded
- THEN it is accepted and used to drive the iteration loop

#### Scenario: Missing required fields

- GIVEN a YAML file missing `name` or `requirements`
- WHEN `ralph validate <file>` is run
- THEN an error is reported describing the missing field

### Requirement: Requirement Status

Each requirement MUST have a `passing` boolean field indicating whether the AI needs to work on it.

#### Scenario: Failing requirement selected

- GIVEN multiple requirements, some with `passing: false`
- WHEN the iteration loop selects the next requirement
- THEN the highest-priority failing requirement is chosen

#### Scenario: All passing

- GIVEN all requirements have `passing: true`
- WHEN ralph runs
- THEN no AI agent iteration is performed and no PR is created

### Requirement: Branch Naming

The system SHALL derive the git branch name from the project `name` field.

#### Scenario: Branch name sanitization

- GIVEN a project with `name: user-authentication`
- WHEN ralph runs
- THEN the working branch is `ralph/user-authentication`

### Requirement: Project Validation

The system SHALL provide a `ralph validate <file>` command that checks a project file without executing it.

#### Scenario: Valid file

- GIVEN a correctly structured project YAML file
- WHEN `ralph validate <file>` is run
- THEN exit code 0 is returned with a success message

#### Scenario: Invalid file

- GIVEN a project YAML file with schema errors
- WHEN `ralph validate <file>` is run
- THEN a non-zero exit code is returned and each error is described

### Requirement: Spec and Flow References

A project MAY include `spec` and `flow` fields linking to the feature's documentation. These are reference metadata — their content is not injected into the agent's prompt.

#### Scenario: Spec and flow provided

- GIVEN a project with `spec` and `flow` fields
- WHEN the project is loaded
- THEN the fields are stored as metadata and the project is valid

### Requirement: Version Requirements

Projects in versioned repositories SHOULD include a `version` requirement specifying the bump level, not the target version.

#### Scenario: Patch bump

- GIVEN a requirement that specifies `patch` bump level
- WHEN ralph implements the requirement
- THEN the current version is incremented by one patch level

#### Scenario: Minor bump

- GIVEN a requirement that specifies `minor` bump level
- WHEN ralph implements the requirement
- THEN the current version is incremented by one minor level and the patch is reset to 0
