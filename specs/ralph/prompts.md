# Prompts Specification

## Purpose

A repository names the prompt a launcher runs for each GitHub event or scheduled run in the top-level `prompts` map of `.ralph/config.yaml`. The launcher chart holds no prompt text, so a prompt change reaches the next run without a launcher release.

## Requirements

### Requirement: Prompt map

The config SHALL accept a top-level `prompts` mapping. A key SHALL name a GitHub event, or `schedule` for a scheduled run. Keys SHOULD use lower camel case. A value SHALL be either prompt text or a mapping with a required `file` path.

```yaml
prompts:
  push: |
    Review the repository and make one small improvement.
  pullRequest:
    file: .ralph/prompts/pull-request.md
  schedule:
    file: .ralph/prompts/daily.md
```

`file` SHALL name a file relative to the repository root. The launcher SHALL read it after the clone.

Other commands SHALL ignore `prompts`.

#### Scenario: Inline prompt

- GIVEN `prompts.push` holds prompt text
- WHEN a push run resolves its prompt
- THEN the text is the prompt

#### Scenario: File prompt

- GIVEN `prompts.pullRequest` names `.ralph/prompts/pull-request.md`
- AND that file exists in the cloned repository
- WHEN a pull request run resolves its prompt
- THEN the file contents are the prompt

#### Scenario: Other commands ignore prompts

- GIVEN `.ralph/config.yaml` has a `prompts` section
- WHEN the user runs `ralph run`, `ralph loop`, or `ralph validate`
- THEN the section changes nothing

---

### Requirement: Prompt lookup

The launcher SHALL normalize the delivered event name and each key before comparing them: lowercase the value, then remove `_` and `-`. The GitHub event `pull_request` SHALL read `pullRequest`, `issue_comment` SHALL read `issueComment`, and `push` SHALL read `push`. `schedule` SHALL be the key for a scheduled run.

When the launcher loads the config, two keys whose normalized forms match SHALL be an error that names both keys.

A missing key, an empty prompt, a missing file, or an empty file SHALL fail the run with an error that names the event. The launcher SHALL NOT fall back to another prompt.

#### Scenario: Event name normalizes for lookup

- GIVEN `prompts.pullRequest` holds a prompt
- WHEN the launcher handles a `pull_request` event
- THEN it selects that prompt

#### Scenario: Snake case key matches

- GIVEN `prompts.issue_comment` holds a prompt
- WHEN the launcher handles an `issue_comment` event
- THEN it selects that prompt

#### Scenario: Scheduled run uses the schedule key

- GIVEN `prompts.schedule` holds a prompt
- WHEN a scheduled run starts
- THEN it selects that prompt

#### Scenario: Missing prompt fails

- GIVEN `prompts` has no key for the event
- WHEN the run resolves its prompt
- THEN it fails with an error that names the event
- AND no branch, service, or pass work starts

#### Scenario: Empty prompt fails

- GIVEN the key for the event holds only whitespace
- WHEN the run resolves its prompt
- THEN it fails with an error that names the event

#### Scenario: Missing file fails

- GIVEN the key for the event names a file that does not exist
- WHEN the run resolves its prompt
- THEN it fails with an error that names the event

#### Scenario: Empty file fails

- GIVEN the key for the event names an empty file
- WHEN the run resolves its prompt
- THEN it fails with an error that names the event

#### Scenario: Keys that normalize alike are rejected

- GIVEN `prompts` holds both `pullRequest` and `pull_request`
- WHEN the launcher loads the config
- THEN it fails with an error that names both keys
