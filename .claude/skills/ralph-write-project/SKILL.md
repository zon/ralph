---
name: ralph-write-project
description: Creates and validates a ralph project YAML file that defines work for the ralph agent to execute. Use when the user wants to define a new project, write requirements for ralph, or translate a feature or fix request into a structured ralph project.
---

# Write Ralph Project

Read `docs/projects.md` and `docs/writing-requirements.md` before starting.

## Steps

1. **Understand the work.** If the request is vague, ask clarifying questions before proceeding.

2. **Locate the spec and flow files** if the work targets a documented feature. Read them to understand the requirements and target implementation shape.

3. **Draft the requirements.** The agent sees only the selected requirement at runtime — requirements must be self-contained. Copy key scenarios, constraints, and flow function shapes directly into the items rather than leaving the agent to find them.

4. **Write the file** to `./projects/<name>.yaml` with all `passing: false`.

5. **Validate:**
   ```sh
   ralph validate ./projects/<name>.yaml
   ```

6. **Report** the file path and a one-line summary.
