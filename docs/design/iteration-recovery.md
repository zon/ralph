# Iteration Recovery Design

`ralph run` and `ralph loop` share one recovery rule for a failed AI iteration: capture the failure, carry it into the next prompt, and keep going unless the failure is fatal.

## Non-Fatal Failures

A non-fatal failure is recorded and the loop continues. The next iteration's prompts include the error as the failure of the previous attempt, so the agent can address its cause. The picker and the development prompt both carry it.

A successful iteration clears the recorded error. Only the most recent failure is carried, so consecutive failures do not accumulate a history.

A failed iteration is not committed. It still consumes one iteration from the cap, so retrying a failure never runs past the limit.

## Fatal Failures

A fatal failure stops the loop immediately and is returned as an error. Billing and quota errors are fatal. No further prompt is built or run.

## Related

* [Run Design](run.md)
* [Loop Design](loop.md)
