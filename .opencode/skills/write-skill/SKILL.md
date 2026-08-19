---
name: write-skill
description: Writes an agent skill into the central skills/ directory following the agentskills.io spec. Use when the user wants to add a new skill or edit an existing one.
---

# Write Skill

Write a skill that a coding agent can invoke by name in any repository that installs it.

## Steps

1. **Read the [Agent Skills spec](https://agentskills.io/specification)** defining the skill format. The frontmatter `name` must match the directory name, and the `description` is what an agent matches against to decide whether to invoke the skill. Write it as the trigger, naming the situations that should call it.

2. **Ask for the task and output** if the user's request doesn't state them.

3. **Read [docs/zpecs/prompts.md](docs/zpecs/prompts.md).** A skill is a prompt that has to locate its own context, so the same structure applies: one task, numbered steps, explicit output.

4. **Write the skill** to `skills/<name>/SKILL.md`.

   - **Reference documentation rather than repeating it.** Link to the doc and let the agent read it when needed. A skill that restates a format goes stale the moment the format changes.
   - **Reference documents at their installed path.** Documents in this repository live at `docs/zpecs/<file>.md` and install to the same path in the target repository, so a relative markdown link resolves in both places. Use markdown links, never bare paths or code spans.
   - **Target-repository paths need no link.** Paths like `./specs/<path>.md` or `specs/architecture.yaml` resolve wherever the skill runs.
   - **Keep the frontmatter to the [spec's six fields](https://agentskills.io/specification):** `name`, `description`, `license`, `compatibility`, `metadata`, `allowed-tools`. Other runners ignore runner-specific fields, so use them only when a skill targets one runner.

5. **Report** the skill path, the situations that trigger it, and a one-line summary.
