---
name: ralph-write-architecture
description: Creates or edits the architecture document at specs/architecture.yaml. Use when the user wants to outline the deep modules of an application, document current architecture, or plan future modules.
---

# Write Architecture

Create or update the architecture document at `./specs/architecture.yaml`.

## Steps

1. **Read the architecture format docs** at `docs/formats/architecture.md`.

2. **Read the existing architecture document** at `./specs/architecture.yaml` if one exists, so edits preserve unrelated modules.

3. **Understand the scope.** If the user's request is vague, ask clarifying questions.

4. **Survey the codebase** when documenting current architecture to confirm module paths and responsibilities.

5. **Draft the architecture** following the format in `docs/formats/architecture.md`.

6. **Write the file** to `./specs/architecture.yaml`.

7. **Report** the file path and a one-line summary of the modules covered.
