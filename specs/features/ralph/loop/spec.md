# Loop Command Specification

## Purpose

The `loop` command runs a bounded AI iteration loop over a list of steps. Given a slug, ralph looks up the matching loop config in the `loops:` section of `.ralph/config.yaml`, embeds that config's `steps` in a prompt, and runs the prompt until the agent reports nothing left to do or the `--max` cap is reached. Every iteration that does real work is committed and pushed to a `loop-<slug>` branch. When the loop ends with commits on that branch, ralph opens a pull request. Steps can also be supplied directly with `--step` flags, which replace the config's `steps`. When steps are supplied without a slug, ralph asks the AI to read the steps and propose a slug.

## Requirements

### Requirement: Slug or steps required

The command SHALL accept an optional positional slug argument and any number of `--step` flags. Exactly one of a slug argument or at least one `--step` flag SHALL be provided. When neither is given, the command SHALL return an error with usage before any execution begins.

#### Scenario: Slug argument provided

- GIVEN the user runs `ralph loop fmt`
- WHEN the command validates its input
- THEN the slug is `fmt` and execution proceeds

#### Scenario: Steps provided without a slug

- GIVEN the user runs `ralph loop --step "run gofmt" --step "run go vet"`
- WHEN the command validates its input
- THEN execution proceeds with steps `run gofmt` and `run go vet`

#### Scenario: Neither slug nor steps provided

- GIVEN the user runs `ralph loop` with no slug argument and no `--step` flags
- WHEN the command validates its input
- THEN an error is returned indicating that a slug or at least one `--step` is required
- AND no execution begins

---

### Requirement: Loop config resolution

When a slug argument is provided, the command SHALL load `.ralph/config.yaml` and find the entry in its `loops:` section whose `slug` matches the argument. The matching entry's `steps` SHALL be the resolved steps, unless `--step` flags override them (see [Steps override](#requirement-steps-override)). When no entry matches the slug, the command SHALL return an error and no execution begins.

#### Scenario: Matching loop config found

- GIVEN `.ralph/config.yaml` has a `loops:` entry with `slug: fmt` and two steps
- WHEN the user runs `ralph loop fmt`
- THEN the entry's `steps` are resolved for the prompt
- AND the branch is derived from the slug `fmt`

#### Scenario: Loop config not found

- GIVEN `.ralph/config.yaml` has no `loops:` entry whose `slug` is `fmt`
- WHEN the user runs `ralph loop fmt`
- THEN an error is returned: `loop config not found: fmt`
- AND no prompt runs

#### Scenario: Slug lookup still required when steps override

- GIVEN the user runs `ralph loop fmt --step "run tests"`
- AND no `loops:` entry has `slug: fmt`
- WHEN the command resolves the loop config
- THEN an error is returned: `loop config not found: fmt`
- AND no prompt runs

---

### Requirement: Steps override

The command SHALL accept one or more `--step` flags. When at least one `--step` flag is passed, the resolved steps SHALL be the flag values, in the order given, replacing the loop config's `steps` property entirely.

#### Scenario: Single `--step` replaces config steps

- GIVEN a `loops:` entry with `slug: fmt` whose `steps` list has two entries
- AND the user runs `ralph loop fmt --step "run tests"`
- WHEN the prompt is built
- THEN the prompt embeds only the step `run tests`

#### Scenario: Multiple `--step` flags in order

- GIVEN the user runs `ralph loop --step "step one" --step "step two" --step "step three"`
- WHEN the prompt is built
- THEN the prompt embeds the three steps in the order given

#### Scenario: No `--step` flags uses config steps

- GIVEN a `loops:` entry with `slug: fmt` whose `steps` list has two entries
- AND the user runs `ralph loop fmt` with no `--step` flags
- WHEN the prompt is built
- THEN the prompt embeds the config's two steps

---

### Requirement: Slug derived from steps

When `--step` flags are provided without a slug argument, the command SHALL invoke the AI with the steps and ask it to read them and propose a slug. The proposed slug SHALL be used to derive the branch name, and no loop config lookup SHALL occur.

#### Scenario: Slug proposed from steps

- GIVEN the user runs `ralph loop --step "run gofmt" --step "run go vet"`
- WHEN the AI is asked to propose a slug
- THEN the AI reads the steps and proposes a slug, e.g. `format-code`
- AND the branch is derived from that proposed slug

#### Scenario: AI returns no usable slug

- GIVEN the AI is asked to propose a slug from the steps
- AND the AI returns an empty or blank proposal
- WHEN the slug proposal is processed
- THEN an error is returned
- AND no prompt runs and no branch is created

---

### Requirement: Prompt construction

The command SHALL build a prompt that embeds the resolved steps, in order. The prompt SHALL instruct the AI to follow the steps and to write a brief summary of what was done to `report.md`. The prompt SHALL also instruct the AI that when nothing was necessary, it MUST write exactly the constant string `NOTHING_TO_DO` to `report.md` instead of a summary.

#### Scenario: Config steps embedded

- GIVEN a `loops:` entry with `slug: fmt` and steps `step one` and `step two`
- WHEN the prompt is built
- THEN the prompt embeds both steps in order

#### Scenario: Steps from flags embedded

- GIVEN the user passes `--step "run tests"`
- WHEN the prompt is built
- THEN the prompt embeds the step `run tests`

#### Scenario: Prompt requires a `report.md` summary

- GIVEN the prompt is built
- WHEN the AI reads it
- THEN the prompt instructs the AI to write a brief summary of what it did to `report.md`

#### Scenario: Prompt names the nothing-to-do constant

- GIVEN the prompt is built
- WHEN the AI reads it
- THEN the prompt instructs the AI to write exactly `NOTHING_TO_DO` to `report.md` when nothing was necessary

---

### Requirement: Iteration loop

The command SHALL run the prompt repeatedly as an iteration loop. Each iteration SHALL invoke the AI with the prompt and then read `report.md`. The loop SHALL stop when the report content equals the constant string `NOTHING_TO_DO` (trimmed of surrounding whitespace) or when the number of iterations reaches the `--max` cap, whichever comes first. The `--max` flag SHALL default to `10` and SHALL be a positive integer.

#### Scenario: Stops on the nothing-to-do report

- GIVEN the AI writes exactly `NOTHING_TO_DO` to `report.md` during iteration 3
- WHEN the report is read
- THEN the loop stops immediately
- AND no further iteration runs

#### Scenario: Stops at the `--max` cap

- GIVEN `--max 3` and the AI never writes `NOTHING_TO_DO`
- WHEN the loop reaches the third iteration
- THEN the loop stops after iteration 3
- AND no fourth iteration runs

#### Scenario: Default `--max` is 10

- GIVEN no `--max` flag is passed
- AND the AI never writes `NOTHING_TO_DO`
- WHEN the loop runs
- THEN the loop runs at most 10 iterations

#### Scenario: Custom `--max` overrides the default

- GIVEN the user passes `--max 5`
- WHEN the loop runs
- THEN the loop runs at most 5 iterations

#### Scenario: Non-positive `--max` rejected

- GIVEN the user passes `--max 0`
- WHEN the command validates the flag
- THEN an error is returned
- AND no loop runs

---

### Requirement: Commit and push each iteration

After each iteration whose report is not the nothing-to-do constant, the command SHALL commit the AI's changes and push them to the branch `loop-<slug>`, creating the branch from the current branch when it does not already exist. The commit message SHALL be the report content. `report.md` SHALL be deleted after the commit. An iteration whose report equals `NOTHING_TO_DO` SHALL NOT be committed.

#### Scenario: Changes committed and pushed

- GIVEN the AI made changes and wrote a summary to `report.md`
- WHEN the iteration completes
- THEN the changes are committed with the report content as the message
- AND the commit is pushed to `loop-<slug>`

#### Scenario: Branch created from the current branch

- GIVEN no `loop-<slug>` branch exists and the current branch is `main`
- WHEN the first commit is pushed
- THEN the branch `loop-<slug>` is created from `main`

#### Scenario: `report.md` deleted after the commit

- GIVEN the AI wrote a summary to `report.md`
- WHEN the changes are committed
- THEN `report.md` is removed from the working tree after the commit

#### Scenario: Nothing-to-do iteration is not committed

- GIVEN the AI wrote exactly `NOTHING_TO_DO` to `report.md`
- WHEN the iteration completes
- THEN no commit is created and nothing is pushed for that iteration

---

### Requirement: Pull request on completion

When the loop ends, the command SHALL open a pull request from the branch `loop-<slug>` to the branch it was created from, if and only if at least one commit was made on `loop-<slug>`. When no commits were made, the command SHALL NOT open a pull request and SHALL exit successfully.

#### Scenario: Pull request opened when commits exist

- GIVEN the loop ended after committing changes on `loop-<slug>`
- WHEN the loop finishes
- THEN a pull request is opened from `loop-<slug>` to the branch it was created from

#### Scenario: No pull request when nothing was committed

- GIVEN the loop stopped on the nothing-to-do report in its first iteration
- AND no commit was made on `loop-<slug>`
- WHEN the loop finishes
- THEN no pull request is opened
- AND the command exits successfully
