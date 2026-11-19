# Iteration Recovery Specification

## Purpose

`ralph run` and `ralph loop` follow one recovery rule when an iteration fails: keep the code the attempt produced, drop the feedback it left behind, carry the failure into the next prompt so the agent can correct course, and keep the loop going unless the failure is fatal.

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

### Requirement: A failed iteration keeps its code changes and discards its feedback files

When an iteration fails with a non-fatal error, the command SHALL leave the attempt's working-tree code changes in place, so the next iteration builds on that work. It SHALL remove the attempt's `report.md` and `blocked.md` before the next iteration starts. A later commit SHALL NOT use the stale report as its message or treat the stale block as current. If a feedback file cannot be removed, the command SHALL stop the loop and return the error. A loop iteration whose report is missing or unreadable SHALL fail the same way.

#### Scenario: The failed attempt's code changes are kept

- GIVEN an iteration's AI pass fails with a non-fatal error after changing repository code
- WHEN the failure is processed
- THEN the code changes remain in the working tree

#### Scenario: Feedback files are removed

- GIVEN an iteration's AI pass fails with a non-fatal error
- AND the attempt left `report.md` or `blocked.md`
- WHEN the failure is processed
- THEN both feedback files are removed

#### Scenario: A later commit cannot use the stale report

- GIVEN a failed iteration left a `report.md`
- WHEN a later iteration succeeds and commits
- THEN the commit message comes from that iteration's report, not the stale one

#### Scenario: A feedback file that cannot be removed stops the loop

- GIVEN an iteration's AI pass fails with a non-fatal error
- AND removing a feedback file fails
- WHEN the failure is processed
- THEN the loop stops
- AND the removal error is returned

#### Scenario: A loop iteration with an unreadable report is a failed iteration

- GIVEN a loop iteration's AI pass succeeds
- AND its report is missing or unreadable
- WHEN the iteration ends
- THEN no commit is created
- AND its feedback files are discarded
- AND the error is carried into the next prompt

---

### Requirement: The next prompt says the kept changes are optional

A prompt that carries a previous failure and writes code SHALL say any changes the failed attempt left are still in the working tree and are not required. It SHALL say the agent can restore the repository to its last commit if they are not useful.

#### Scenario: The development prompt says the changes are optional

- GIVEN the previous iteration failed with a non-fatal error
- WHEN the next development prompt is built
- THEN it says any changes the failed attempt left are still in the working tree
- AND it says they are not required
- AND it says the agent can restore the repository to its last commit

#### Scenario: The loop prompt says the changes are optional

- GIVEN the previous iteration failed with a non-fatal error
- WHEN the next loop prompt is built
- THEN it says any changes the failed attempt left are still in the working tree
- AND it says they are not required
- AND it says the agent can restore the repository to its last commit

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
