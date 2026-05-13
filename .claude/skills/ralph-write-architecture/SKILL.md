---
name: ralph-write-architecture
description: Creates or edits the architecture document at specs/architecture.yaml. Use when the user wants to outline the deep modules of an application, document current architecture, or plan future modules.
---

# Write Architecture

Create or update an architecture document. Architecture files live in two places:

- **`./specs/architecture.yaml`** — current modules of the application.
- **`./specs/features/<component>/<feature>/architecture.yaml`** (optional) — future modules introduced by a specific feature.

## Steps

1. **Read the architecture format docs** at `docs/formats/architecture.md`.

2. **Determine the target file:**
   - If documenting **current** architecture, use `./specs/architecture.yaml`.
   - If planning **future** modules for a specific feature, use `./specs/features/<component>/<feature>/architecture.yaml`. Ask the user for the component and feature names if unclear.

3. **Read the existing architecture document** at the target path if one exists, so edits preserve unrelated modules.

4. **Understand the scope.** If the user's request is vague, ask clarifying questions.

5. **Survey the codebase** when documenting current architecture to confirm module paths and responsibilities.

6. **Draft the architecture** following the format in `docs/formats/architecture.md`.

7. **Write the file** to the target path.

8. **Report** the file path and a one-line summary of the modules covered.
