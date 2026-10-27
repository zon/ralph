# Iteration Recovery Design

`ralph run` and `ralph loop` share one recovery rule for a failed AI iteration: capture the failure, carry it into the next prompt, and keep going unless the failure is fatal.

## Non-Fatal Failures

Ralph records a non-fatal failure and continues the loop. The next iteration's prompts include the error as the failure of the previous attempt, so the agent can address its cause. The picker and the development prompt both carry it.

A successful iteration clears the recorded error. Ralph carries only the most recent failure, so consecutive failures do not accumulate a history.

Ralph does not commit a failed iteration. It still consumes one iteration from the cap, so retrying a failure never runs past the limit.

## Fatal Failures

A fatal failure stops the loop immediately, and Ralph returns an error. Billing and quota errors are fatal. Ralph builds and runs no further prompt.
