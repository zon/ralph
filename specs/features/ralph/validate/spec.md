# Validate Command Specification

## Purpose

Define the behavior of the `ralph validate` command, which checks that a project YAML file unmarshals into a well-formed project, asks a locally-run agent to repair the file when it does not, and rewrites the file in canonical format on success.

## Requirements

### Requirement: Command Invocation

The system SHALL provide a `ralph validate <file>` subcommand that accepts a path to a project YAML file as its sole positional argument.

#### Scenario: File path provided

- GIVEN a path to a project YAML file
- WHEN the user runs `ralph validate <file>`
- THEN the command loads the file from that path and begins validation

#### Scenario: File path missing

- GIVEN no positional argument
- WHEN the user runs `ralph validate`
- THEN the command exits with a non-zero status and reports that a project file path is required

### Requirement: Project Unmarshalling

The command MUST attempt to unmarshal the file into the project model using the same loader used by other ralph commands. The project is considered valid when it parses as YAML and satisfies the project schema (required fields populated, requirements well-formed).

#### Scenario: Well-formed project

- GIVEN a YAML file that parses and satisfies the project schema
- WHEN `ralph validate <file>` is run
- THEN unmarshalling succeeds on the first attempt
- AND no agent is invoked

#### Scenario: Unmarshalling failure

- GIVEN a YAML file that fails to parse or fails schema checks
- WHEN `ralph validate <file>` is run
- THEN the command enters the fix loop described below
- AND the underlying error is reported to the user before each fix attempt

### Requirement: Local Agent Fix Loop

When unmarshalling fails, the command MUST invoke an AI agent locally to repair the file in place, then retry unmarshalling. The loop MUST continue until the project unmarshals successfully or the attempt limit is reached.

The agent MUST be run locally on the current machine (the same execution mode as `ralph run --local`), never delegated to a remote workflow runner. The agent MUST use the model configured in the ralph config file, with no command-line override required.

#### Scenario: Agent fixes a malformed file

- GIVEN a file whose contents do not unmarshal into a valid project
- WHEN `ralph validate <file>` is run
- THEN the command invokes the agent with the file path and the unmarshalling error
- AND the agent rewrites the file
- AND the command retries unmarshalling against the updated file

#### Scenario: Local execution

- GIVEN any failed unmarshalling
- WHEN the fix loop invokes the agent
- THEN the agent runs on the local machine using the same path used by `ralph run --local`
- AND no Argo workflow or remote runner is involved

#### Scenario: Model selection from config

- GIVEN a ralph config file with a `model` field
- WHEN the fix loop invokes the agent
- THEN that model is used
- AND the user is not required to pass a model flag

### Requirement: Fix Loop Limit

The fix loop MUST be capped at 10 total unmarshalling attempts (the initial attempt plus up to 9 agent-assisted retries). If the project still fails to unmarshal after the final attempt, the command MUST exit with a non-zero status code and report that the limit was reached along with the most recent unmarshalling error.

#### Scenario: Project becomes valid within the limit

- GIVEN a file that the agent successfully repairs within 10 attempts
- WHEN `ralph validate <file>` is run
- THEN the loop exits as soon as unmarshalling succeeds
- AND the command proceeds to canonical formatting

#### Scenario: Limit exceeded

- GIVEN a file that the agent cannot repair in 10 attempts
- WHEN `ralph validate <file>` is run
- THEN the command exits with a non-zero status
- AND the error message reports that the 10-attempt limit was reached
- AND the error message includes the final unmarshalling error

### Requirement: Canonical Formatting

After unmarshalling succeeds, the command MUST marshal the project model back to the file using the same serialization path as `ralph pass`. This ensures the file is written in the canonical layout regardless of how it was originally formatted.

#### Scenario: File rewritten in canonical format

- GIVEN a project file that unmarshals successfully (immediately or after agent fixes)
- WHEN `ralph validate <file>` finishes validation
- THEN the file is rewritten using the same marshalling routine as `ralph pass`
- AND the on-disk content matches the canonical representation of the parsed project

#### Scenario: Already-canonical file is unchanged

- GIVEN a file that is already in canonical format
- WHEN `ralph validate <file>` rewrites it
- THEN the resulting file content is byte-identical to the input

### Requirement: Successful Validation Output

When validation completes successfully, the command MUST exit with status code 0 and emit a confirmation message identifying the project slug and the number of requirements it contains.

#### Scenario: Valid project file

- GIVEN a project file that ends up valid (with or without agent fixes)
- WHEN `ralph validate <file>` finishes
- THEN the command exits with status code 0
- AND a message confirms the project is valid and reports its slug and requirement count
