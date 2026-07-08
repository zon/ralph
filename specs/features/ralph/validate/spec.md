# Validate Command Specification

## Purpose

Define the behavior of the `ralph validate` command, which checks that a project file (JSON or YAML) unmarshals into a well-formed project, asks a locally-run agent to repair the file when it does not, and rewrites the file in canonical YAML format on success. When the input file has a `.json` extension, the validated output is written to a new `.yaml` file and the original `.json` file is removed.

## Requirements

### Requirement: Command Invocation

The system SHALL provide a `ralph validate <file>` subcommand that accepts a path to a project file (JSON or YAML) as its sole positional argument.

#### Scenario: File path provided

- GIVEN a path to a project file
- WHEN the user runs `ralph validate <file>`
- THEN the command loads the file from that path and begins validation

#### Scenario: File path missing

- GIVEN no positional argument
- WHEN the user runs `ralph validate`
- THEN the command exits with a non-zero status and reports that a project file path is required

### Requirement: Project Unmarshalling

The command MUST attempt to unmarshal the file into the project model using the same loader used by other ralph commands. The loader accepts both JSON and YAML input. The project is considered valid when it parses and satisfies the project schema (required fields populated, requirements well-formed).

#### Scenario: Well-formed project

- GIVEN a file that parses and satisfies the project schema
- WHEN `ralph validate <file>` is run
- THEN unmarshalling succeeds on the first attempt
- AND no agent is invoked

#### Scenario: Unmarshalling failure

- GIVEN a file that fails to parse or fails schema checks
- WHEN `ralph validate <file>` is run
- THEN the command enters the fix loop described below
- AND the underlying error is reported to the user before each fix attempt

### Requirement: Local Agent Fix Loop

When unmarshalling fails, the command MUST invoke an AI agent locally to repair the file in place, then retry unmarshalling. The loop MUST continue until the project unmarshals successfully or the attempt limit is reached.

The agent MUST be run locally on the current machine (the same execution mode as `ralph run --local`), never delegated to a remote workflow runner. The agent MUST use the model resolved from the ralph config file, with no command-line override required.

Model resolution follows a two-level precedence: if `validate.model` is set in `.ralph/config.yaml` that model is used; otherwise the top-level `model` field is used as the fallback.

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

#### Scenario: Validate-specific model used when configured

- GIVEN `validate.model` is set in `.ralph/config.yaml`
- WHEN the fix loop invokes the agent
- THEN the validate-specific model is used
- AND the user is not required to pass a model flag

#### Scenario: Fallback to main model when validate model is unset

- GIVEN `validate.model` is not set in `.ralph/config.yaml`
- AND the top-level `model` field is set
- WHEN the fix loop invokes the agent
- THEN the top-level model is used as the fallback

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

After unmarshalling succeeds, the command MUST marshal the project model back to disk as YAML using the same serialization path as `ralph pass`. This ensures the output is always in canonical YAML layout regardless of the input format.

#### Scenario: File rewritten in canonical format

- GIVEN a project file that unmarshals successfully (immediately or after agent fixes)
- WHEN `ralph validate <file>` finishes validation
- THEN the file is rewritten using the same marshalling routine as `ralph pass`
- AND the on-disk content matches the canonical YAML representation of the parsed project

#### Scenario: Already-canonical YAML file is unchanged

- GIVEN a YAML file that is already in canonical format
- WHEN `ralph validate <file>` rewrites it
- THEN the resulting file content is byte-identical to the input

#### Scenario: JSON file renamed to YAML

- GIVEN a project file with a `.json` extension that unmarshals successfully
- WHEN `ralph validate <file>` finishes validation
- THEN the validated project is written to a new file with the same name but a `.yaml` extension
- AND the original `.json` file is removed

#### Scenario: Empty values omitted from canonical output

- GIVEN a validated project with unset/empty fields (e.g., `extraIterations` is nil)
- WHEN the project is marshalled to canonical YAML
- THEN fields with empty or nil values are omitted from the output
- AND no fields are emitted with empty values

### Requirement: Successful Validation Output

When validation completes successfully, the command MUST exit with status code 0 and emit a confirmation message identifying the project slug and the number of requirements it contains.

#### Scenario: Valid project file

- GIVEN a project file that ends up valid (with or without agent fixes)
- WHEN `ralph validate <file>` finishes
- THEN the command exits with status code 0
- AND a message confirms the project is valid and reports its slug and requirement count
