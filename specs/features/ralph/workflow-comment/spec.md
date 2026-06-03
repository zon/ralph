# Workflow Comment Specification

## Purpose

`ralph workflow comment` prompts the AI agent with the content of a PR comment after the workspace is ready, running one development iteration against instructions derived from the comment body.

## Requirements

### Requirement: Workspace Setup

The system SHALL prepare the container workspace as defined in [workflow-workspace/spec.md](../workflow-workspace/spec.md) before doing any work. The workspace is prepared with the project branch as the target, creating it if it does not yet exist on the remote.

### Requirement: Comment Execution

The system SHALL invoke the AI agent with the comment body as the prompt after the workspace is ready.

#### Scenario: Missing comment body

- GIVEN no comment body argument is provided to `ralph workflow comment`
- WHEN the command starts
- THEN an error is returned before any work is done

#### Scenario: Successful iteration

- GIVEN a valid project file exists and the comment body is non-empty
- WHEN `ralph workflow comment` runs
- THEN the AI agent is invoked with a rendered prompt incorporating the comment body, PR number, branch, repo owner, and repo name
- AND the agent runs to completion

### Requirement: Instruction Rendering

The system SHALL render the AI prompt from the configured comment instructions template, substituting PR context variables.

#### Scenario: Template variables injected

- GIVEN a comment instructions template containing `{{.CommentBody}}`, `{{.PRNumber}}`, `{{.PRBranch}}`, `{{.RepoOwner}}`, or `{{.RepoName}}`
- WHEN the prompt is rendered
- THEN each placeholder is replaced with the corresponding value from the command arguments

#### Scenario: Custom instructions

- GIVEN `commentInstructionsFile` is configured in the webhook app config and its content has been passed to the container
- WHEN the prompt is rendered
- THEN the custom template is used instead of the built-in default

#### Scenario: Template parse failure

- GIVEN the instructions template contains invalid Go template syntax
- WHEN the prompt is rendered
- THEN the raw template text is used as the prompt without substitution

### Requirement: Service Management

The system SHOULD start configured services before the agent runs and stop them after, unless `--no-services` is set.

#### Scenario: Services started

- GIVEN `services` are defined in `.ralph/config.yaml` and `--no-services` is not set
- WHEN `ralph workflow comment` runs
- THEN services are started before the AI agent and stopped when the command exits

#### Scenario: No-services flag

- GIVEN `--no-services` is set
- WHEN `ralph workflow comment` runs
- THEN no services are started or stopped regardless of config

### Requirement: Comment Response

After any commits have been pushed, the system SHALL post a reply comment on the PR. The reply SHALL answer any questions in the original comment and summarize any changes committed.

#### Scenario: Changes committed

- GIVEN the agent committed and pushed changes in response to the comment
- WHEN the commits have been pushed
- THEN a reply is posted on the PR summarizing what was committed

#### Scenario: No changes committed

- GIVEN the agent made no code changes
- WHEN the agent finishes
- THEN a reply is posted on the PR responding to the comment content

### Requirement: Commit Agent Changes

The system SHALL commit and push any code changes the agent produces. No pull request SHALL be created or updated.

#### Scenario: Agent changes committed

- GIVEN the AI agent makes file changes in response to the comment
- WHEN the agent finishes
- THEN the changes are committed and pushed to the project branch
- AND no pull request is created or updated

#### Scenario: No changes produced

- GIVEN the AI agent makes no file changes
- WHEN the agent finishes
- THEN no commit is created
