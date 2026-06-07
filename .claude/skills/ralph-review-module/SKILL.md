---
name: ralph-review-module
description: Reviews a module against the architecture standards for its category (entry, orchestration, implementation), reports gaps, and creates a project to code the recommendations. Use when the user wants to audit a module, bring it up to standard, or understand what it's missing.
---

# Review Module

Audit a module against the architecture and testing standards for its declared category, report what is compliant and what is missing, then create a project to carry out the recommendations.

## Steps

1. **Identify the module path.** Use the argument passed to the skill if provided (e.g. `internal/argo`). If none, ask the user which module to review.

2. **Read the architecture registry** at [specs/architecture.yaml](specs/architecture.yaml) to find the module's entry. Note its `category` (`entry`, `orchestration`, or `implementation`) and its declared `description`.

3. **Read the standards docs:**
   - [docs/code.md](docs/code.md) — module placement rules and orchestration vs implementation constraints.
   - [docs/testing.md](docs/testing.md) — testing patterns, mock rules, and what must never be called in tests.

4. **Read all files in the module directory.** Understand what the module currently contains: structs, interfaces, free functions, test files, mock files.

5. **Check against category standards.** Use the category's `description` and `signatures` fields from [specs/architecture.yaml](specs/architecture.yaml) as the primary definition of what belongs in that category. Then apply whatever rules [docs/code.md](docs/code.md) and [docs/testing.md](docs/testing.md) specify for that category — read those files and derive the criteria from them rather than assuming any fixed set of rules.

6. **Compile the findings** into three sections:

   **Compliant** — what the module already does correctly.

   **Gaps** — specific things missing or wrong, each tied to a rule from the standards docs. For each gap, name the file and the exact issue.

   **Recommendations** — ordered list of changes to bring the module up to standard. Be concrete: name the file to create or edit and what to add or change. Do not hesitate to redefine interfaces or change signatures anywhere in the repo — recommend the clean end state. Do not recommend compatibility layers, re-exports, or deprecation shims unless the user has explicitly asked for backwards compatibility.

7. **Report the findings** to the user. If the module is fully compliant, say so clearly and stop — no project is needed.

8. **If gaps exist, create a project to code the recommendations.** Invoke the `ralph-write-project` skill, handing it the module path and the **Recommendations** list from step 6 as the work to be done. Let that skill draft, write, and validate the project file according to its own steps.

9. **Report the project file path** alongside the findings, so the user can hand it to the ralph agent to execute.
