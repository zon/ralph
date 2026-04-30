---
name: ralph-write-module
description: Creates module docs in /specs/modules for a feature. Use when the user wants to plan the module structure for a feature, identify what packages are needed, or document module designs before or alongside implementation.
---

# Write Module Docs

Identify the modules a feature needs and create a well-formed module doc in `./specs/modules/` for each one.

## Steps

1. **Understand the scope.** If the user's request is vague, ask clarifying questions:
   - Which feature are we designing modules for?
   - Are there existing module docs in `specs/modules/` that this feature should reuse or extend?

2. **Read the module format docs** to refresh your understanding:
   - `docs/planning/modules.md`

3. **Read the feature's planning docs.** Locate and read:
   - `specs/features/<component>/<feature>/spec.md` — to understand the behavioral requirements
   - `specs/features/<component>/<feature>/flow.md` — to identify the helpers the flow calls and the data shapes it uses

4. **Identify the modules.** From the spec and flow, determine which self-contained packages or subsystems need to be built. Use [What a Module Is](docs/planning/modules.md#what-a-module-is) to decide what warrants a doc.

5. **Draft each module doc** following [Module File Format](docs/planning/modules.md#module-file-format). Design the `## Interface` to match the call sites in the flow.

6. **Write each file** to `./specs/modules/<module>.md`.

7. **Report** the files written and a one-line summary of what each module does.
