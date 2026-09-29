# Iteration Recovery Specification

## Purpose

The shared behavior that `ralph run` and `ralph loop` use when an AI iteration fails: capture the failure, carry it into the next prompt so the agent can correct course, and keep the loop going unless the failure is fatal.

## Requirements

### Requirement: A failed iteration's error is carried into the next prompt

When an iteration's AI pass fails with a non-fatal error, the command SHALL record the error and SHALL include it in the next AI prompt it builds. The loop SHALL continue to the next iteration instead of stopping. The prompt SHALL present the error as the failure of the previous attempt so the agent can address its cause.

#### Scenario: Non-fatal failure continues the loop

- GIVEN an iteration's AI pass fails with a non-fatal error
- WHEN the failure is processed
- THEN the error is recorded
- AND the loop moves on to the next iteration

#### Scenario: The next prompt carries the error

- GIVEN an iteration failed with a non-fatal error
- WHEN the next iteration's AI prompt is built
- THEN the prompt includes the previous iteration's error
- AND the prompt says that the previous attempt failed with that error

#### Scenario: A successful iteration clears the error

- GIVEN an iteration follows a failed iteration
- WHEN that iteration's AI pass succeeds
- THEN the recorded error is cleared
- AND a later prompt carries no previous error

#### Scenario: Only the most recent failure is carried

- GIVEN consecutive iterations fail with different errors
- WHEN the next prompt is built
- THEN it carries the error from the immediately preceding failure only

---

### Requirement: Every prompt of the next iteration carries the error

When an iteration builds more than one AI prompt, the command SHALL include the recorded error in each prompt it builds for the next iteration, so the agent sees the failure whichever prompt it runs.

#### Scenario: Picker and development prompts both carry the error

- GIVEN an item iteration builds a picker prompt and then a development prompt
- AND the previous iteration failed with a non-fatal error
- WHEN the next iteration's prompts are built
- THEN both prompts include the previous iteration's error

---

### Requirement: A fatal failure stops the loop

A failure the command classifies as fatal, such as a billing or quota error, SHALL stop the loop immediately and SHALL be returned as an error. No further prompt SHALL be built or run.

#### Scenario: Fatal failure aborts the loop

- GIVEN an iteration's AI pass fails with a fatal error
- WHEN the failure is processed
- THEN the loop stops immediately
- AND the fatal error is returned
- AND no further prompt is run

---

### Requirement: Carrying a failure forward does not extend the loop

A failed iteration SHALL consume one iteration from the loop's cap, so retrying a failure never runs the loop past that cap.

#### Scenario: Repeated failures exhaust the cap

- GIVEN every iteration's AI pass fails with a non-fatal error
- AND the loop's iteration cap is N
- WHEN the loop runs
- THEN the AI pass runs N times
- AND the loop then ends
