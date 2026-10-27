# AI Design

Ralph's agents run through the OpenCode CLI. Most prompts are embedded instruction templates filled with the run's context and run as one non-interactive OpenCode pass. A direct prompt is the user's text, run verbatim. Ralph writes no code of its own. It selects work, writes prompts, reads reports, and commits what the agent produced.

## Prompts

* Item picker — choose one incomplete item by dependency, ordering, and impact. Writes `picked-item.txt`.
* Item development — implement the selected item. Writes `report.md`, optionally `blocked.md`.
* Changelog — describe changes when there is no report. Writes `report.md`.
* PR summary — write the pull request description from the commit log. Writes a temporary file.
* Service fix — diagnose a service that failed to start. Writes repository code.
* Merge conflicts — resolve conflicts, run tests, and stage the result. Writes repository code.
* Loop — run a configured step list. Writes `report.md`.
* Loop slug — propose a slug from steps. Writes a temporary file.
* Prompt slug — propose a branch slug from the prompt. Writes a temporary file.
* Direct prompt — run the user's text. Writes repository code when the prompt asks.

The development prompt gives the item verbatim with its index and key. It states the exact completion trailer to write when the item is done, and it tells the agent not to edit the project file. See [Completion Design](completion.md).

The picker reports a pick reference, not the item text. Ralph resolves the reference, its text, or the item's canonical YAML, so a picker that copies text still names a selection.

## Model and Agent

`--model` wins over the config `model`. `--variant` wins over config `variant` and is appended to the model as `provider/model#variant`. A variant with no model fails.

`--agent` wins over the config `agent`. It applies only to prompts that write repository code: development, service fixes, merge conflicts, and direct prompts. Supporting prompts, such as the picker, changelog, PR summary, and prompt slug, run with OpenCode's primary agent.

An unset model or agent leaves the choice to OpenCode. See [Config Design](app/ralph-config.md).

## Reports

An agent communicates results through files, not through its chat output:

* `report.md` holds the commit message. Its last line is the completion trailer when an item is done.
* `blocked.md` asks for help. It explains the block and carries no trailer.
* `picked-item.txt` holds the picker's `ref: <hash>` selection.
* `NOTHING_TO_DO` in `report.md` ends a loop with no commit.

Ralph commits the report verbatim. It never rewrites a trailer. See [Run Design](run.md) and [Loop Design](loop.md).

A direct prompt has no report contract. Its transcript is the output. The prompt commits for itself when its instructions ask. See [Prompt Design](prompt.md).

## Failures

Ralph classifies an agent failure as fatal or non-fatal. Billing and quota errors are fatal. A non-fatal failure is carried into the next iteration's prompts. See [Iteration Recovery Design](iteration-recovery.md).

A direct prompt is one pass, so an agent failure fails the command. It has no next iteration. The prompt slug runs first, and a failed slug fails the command before the prompt starts. A service failure or base conflict can run the matching fix prompt before the direct prompt. When the pass leaves commits, the PR summary prompt runs before the pull request.

Inside a workflow container, Ralph prints accumulated token usage and cost when the run ends.
