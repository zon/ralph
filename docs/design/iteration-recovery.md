# Iteration Recovery Design

`ralph run` and `ralph loop` recover from a failed iteration the same way: keep its code, discard its feedback, carry the error into the next prompt, and continue unless the failure is fatal.

## Non-Fatal Failures

Ralph records a non-fatal failure and continues the loop. The next iteration's prompts say the previous iteration failed and include the error Ralph caught, so the agent can address its cause. The development and loop prompts also say any changes the failed attempt left are still in the working tree. They say the changes are not required and the agent can restore the repository to its last commit if they are not useful.

A successful iteration clears the recorded error. Ralph carries only the most recent failure, so consecutive failures do not accumulate a history.

Ralph does not commit a failed iteration, but it keeps the attempt's working-tree code changes, so the next iteration builds on that work. It discards the attempt's feedback files, `report.md` and `blocked.md`, before the next iteration starts. A later commit then cannot use the stale report as its message or treat the stale block as current. If Ralph cannot remove a feedback file, the loop stops. A failure still consumes one iteration from the cap, so retrying never runs past the limit.

A loop iteration whose report is missing or unreadable fails the same way.

## Fatal Failures

A fatal failure stops the loop immediately, and Ralph returns an error. Billing and quota errors are fatal. Ralph builds and runs no further prompt.
