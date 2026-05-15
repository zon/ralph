---
name: ralph-write-flow
description: Creates a flow document in /specs. Use when the user wants to write a flow, design the domain logic shape for a feature, or produce an idealized implementation template alongside a spec.
---

# Write Flow

Create a well-formed flow file in `./specs/features/` based on the user's description of the feature or process.

## Steps

1. **Understand the scope.** If the user's request is vague, ask clarifying questions:
   - What process or operation does this flow model?
   - Which component and feature does it belong to? (e.g. `ralph/run`, `webhook/events`)
   - What are the main success and failure paths?

2. **Read the flow format docs** to refresh your understanding:
   - `docs/formats/flows.md`

3. **Determine the file path.** Check the existing `specs/features/` structure and place the flow at `./specs/features/<component>/<feature>/flow.md`.

4. **Determine the language** by reading the relevant source files for the feature area.

5. **Check the architecture.** Read `specs/architecture.yaml` and, if it exists, the feature's `specs/features/<component>/<feature>/architecture.yaml` to identify the existing modules the flow should use. Prefer reusing modules defined there over inventing new ones.

6. **Draft the flow and tests** following the format and guidelines in `docs/formats/flows.md`.

7. **Write the file** to `./specs/features/<component>/<feature>/flow.md`.

8. **Report** the file path and a one-line summary of what the flow models.
