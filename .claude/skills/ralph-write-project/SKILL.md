---
name: ralph-write-project
description: Creates and validates a ralph project file defining work for the ralph agent to execute
---

# Write Project

Create a well-formed project file based on the user's description of the work to be done.

## Steps

1. **Understand the work.** If the user's request is vague, ask clarifying questions:
   - What feature or change does this project cover?
   - Does it target a documented feature directory under `specs/features/`?
   - Does the work require a version bump?

2. **Locate the feature directory** if the project targets a documented feature. Feature directories live under `specs/features/<component>/<feature>/` and may contain any of `spec.md`, `orchestration.md`, and `architecture.yaml` — all optional.

3. **Read the project format docs** to refresh your understanding:
   - [docs/formats/project.md](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/project.md)

4. **Read the coding and testing standards** so requirements are consistent with how this codebase is written and tested:
   - `docs/code.md`
   - `docs/testing.md`

5. **Check the module category** for every module the requirements will touch. Read `specs/architecture.yaml`. If the project targets a feature and `<feature-dir>/architecture.yaml` is present, read that too — it describes modules introduced or changed by the feature. Look up the `category` field for each affected module path. The category's `signatures` and `orchestration` flag (defined in [docs/formats/architecture.md](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/architecture.md)) determine what form the code and tests must take. Apply those constraints when writing `items`, `code`, and `tests` for each requirement.

6. **Draft orchestration-based requirements.** If `<feature-dir>/orchestration.md` is present, read it and create a requirement for each implementation shape it defines. Source `code` and `tests` entries exclusively from it — never invent shapes.

7. **Draft scenario-based requirements.** If `<feature-dir>/spec.md` is present, read it and add its scenarios to any matching requirements from step 6. If a scenario doesn't correspond to an orchestration requirement, create a new requirement for it with `scenarios` only.

8. **Draft remaining requirements** as `items` for any work not covered by the orchestration or spec — additional constraints, edge cases, operational requirements, and the version bump if needed.

9. **Write the file** as JSON at `./projects/<slug>.json`, following the format and guidelines in [docs/formats/project.md](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/project.md). The format doc shows YAML — write the equivalent JSON instead.

10. **Validate** by running `ralph validate ./projects/<slug>.json`. This checks the structure and renames the file to `./projects/<slug>.yaml`.

11. **Report** the file path and a one-line summary of what the project covers.
