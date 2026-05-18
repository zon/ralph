# Pass Requirement Specification

## Purpose

Provide a CLI command for marking a project requirement as passing or failing, so users and tooling never need to edit project YAML files directly.

## Requirements

### Requirement: Mark Requirement Passing

The system SHALL update a requirement's `passing` field to `true` when `ralph pass <project> <slug>` is run.

#### Scenario: Mark passing

- GIVEN a project file with a requirement whose slug is `my-req` and `passing: false`
- WHEN the user runs `ralph pass ./projects/my-project.yaml my-req`
- THEN the requirement's `passing` field is set to `true` in the project file
- AND a confirmation message is printed

#### Scenario: Already passing

- GIVEN a requirement that already has `passing: true`
- WHEN the user runs `ralph pass ./projects/my-project.yaml my-req`
- THEN the `passing` field remains `true` in the project file
- AND a confirmation message is printed

### Requirement: Mark Requirement Failing

The system SHALL update a requirement's `passing` field to `false` when `--false` is provided.

#### Scenario: Mark failing

- GIVEN a project file with a requirement whose slug is `my-req` and `passing: true`
- WHEN the user runs `ralph pass ./projects/my-project.yaml my-req --false`
- THEN the requirement's `passing` field is set to `false` in the project file
- AND a confirmation message is printed

### Requirement: Error Handling

The system SHALL exit with a non-zero status and a descriptive error message when the command cannot complete.

#### Scenario: Project file not found

- GIVEN a project file path that does not exist
- WHEN the user runs `ralph pass ./projects/missing.yaml my-req`
- THEN a non-zero exit code is returned
- AND an error message indicates the file was not found

#### Scenario: Slug not found

- GIVEN a project file that does not contain a requirement with the given slug
- WHEN the user runs `ralph pass ./projects/my-project.yaml unknown-slug`
- THEN a non-zero exit code is returned
- AND an error message indicates no requirement with that slug exists in the project
