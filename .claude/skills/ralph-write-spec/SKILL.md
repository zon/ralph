---
name: ralph-write-spec
description: Creates a spec document in /specs. Use when the user wants to write a spec, plan a feature spec, document behavior requirements, or add scenarios for a feature area.
---

# Write Spec

Create a well-formed spec file in `./specs/features/` based on the user's description of the feature or behavior area.

## Steps

1. **Understand the scope.** If the user's request is vague, ask clarifying questions:
   - What feature or behavior area does this spec cover?
   - Which component does it belong to? (e.g. `ralph`, `webhook`)
   - Is this a new feature or documenting existing behavior?
   - Are there known edge cases or failure modes to capture?

2. **Read the spec format docs** to refresh your understanding:
   - `docs/formats/specs.md`

3. **Determine the file path.** Check the existing `specs/features/` structure to match its convention as described in `docs/formats/specs.md`.

4. **Choose the rigor level** as described in `docs/formats/specs.md` (default to Lite).

5. **Draft the spec** following the format and guidelines in `docs/formats/specs.md`.

6. **Write the file** to `./specs/features/<component>/<feature>/spec.md`.

7. **Report** the file path and a one-line summary of what the spec covers.
