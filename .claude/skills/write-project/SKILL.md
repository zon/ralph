---
name: write-project
description: Creates and validates a ralph project YAML file that defines work for the ralph agent to execute. Use when the user wants to define a new project, write requirements for ralph, or translate a feature or fix request into a structured ralph project.
---

# Write Ralph Project

Create a well-formed ralph project YAML file in `./projects/` based on the user's description of the work.

## Steps

1. **Understand the work.** If the user's request is vague, ask clarifying questions:
   - What is the goal? (feature, bug fix, refactor, etc.)
   - Which parts of the codebase are affected?
   - Are there interface changes (new CLI flags, API routes, config fields)?

2. **Read the project format docs** to refresh your understanding of the format and version bump rules:
   - `docs/projects.md`
   - `docs/writing-requirements.md`

3. **Determine the project name.** Use a lowercase, hyphen-separated identifier that reflects the work (e.g. `fix-pagination`, `csv-export`). This becomes the branch name `ralph/<name>`.

4. **Draft the requirements.** Group related work into categories. For each category write a clear `description` and specific `items`. Follow the guidelines in `docs/writing-requirements.md`:
   - Write from the user, client, or developer perspective
   - Describe **what** should happen, not **how** to implement it
   - Be specific about expected behavior
   - Do NOT include "all tests pass" or "no regressions" — ralph handles that automatically

5. **Determine version bump requirements.** Check `docs/projects.md` for version bump rules:
   - Does the repo use versioning? Check `internal/version/VERSION` and `charts/ralph-webhook/Chart.yaml`.
   - If yes, add a `versioning` category with the appropriate bump level for each resource.
   - Use the interface-change heuristic: patch for internal changes, minor for new user-facing behavior, major for breaking changes.

6. **Write the file** to `./projects/<name>.yaml` with all `passing: false`.

7. **Validate** the project:
   ```sh
   ralph validate ./projects/<name>.yaml
   ```
   Fix any errors reported before finishing.

8. **Report** the file path and a one-line summary of what the project will do.

## Output Format

```yaml
name: project-name
description: Brief description used in PR title

requirements:
  - category: <category>
    description: What this group of work accomplishes
    items:
      - Specific outcome written from the user/developer perspective
      - Another outcome
    passing: false

  - category: versioning
    description: Version bump
    items:
      - Apply a semver <patch|minor|major> bump to internal/version/VERSION and charts/ralph-webhook/Chart.yaml
    passing: false
```

## Notes

- Each `items` entry should be a complete, standalone outcome — not a task checklist item.
- Prefer fewer, well-scoped requirements over many granular ones.
- The `description` at the top becomes the PR title, so make it clear and specific.
- If the user provides low-level implementation detail ("add a function called X"), translate it into an outcome ("X is handled by Y package") rather than copying it verbatim.
