---
name: ralph-write-project
description: Creates and validates a ralph project YAML file defining work for the ralph agent to execute
---

# Write Project

Create a well-formed project file based on the user's description of the work to be done.

## Steps

1. **Understand the work.** If the user's request is vague, ask clarifying questions:
   - What feature or change does this project cover?
   - Is there an existing spec and orchestration for it?
   - Does the work require a version bump?

2. **Read the project format docs** to refresh your understanding:
   - [docs/formats/project.md](docs/formats/project.md)

3. **Locate the feature directory** if the work targets a documented feature. Read `spec.md` and `orchestration.md` to source scenarios and code shapes for the requirements.

4. **Draft and write the file** following the format and guidelines in [docs/formats/project.md](docs/formats/project.md).

5. **Validate** the file using the command in [docs/formats/project.md](docs/formats/project.md).

6. **Report** the file path and a one-line summary of what the project covers.
