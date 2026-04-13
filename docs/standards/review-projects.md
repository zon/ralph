# Review Projects

Review projects are [Ralph projects](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/projects.md) written to address issues found during code review. Review projects concern themselves with code architecture and organization rather than adding new domain logic.

Ignore the typical guidelines for [writing good requirements](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/writing-requirements.md) when writing a review project. Instead, write requirements in a code-review style: each requirement item must describe the specific problem found in the code and explain concretely how to fix it. Include file paths and function names where relevant.

## Grouping Review Projects

Group review projects by **module and concern type**. Each project should address one concern type within one module.

**Concern types:**
- **Correctness** — missing validation, incorrect logic, error handling gaps, security issues
- **Performance** — unnecessary allocations, redundant work, inefficient data structures
- **Structure** — misplaced responsibilities, poor abstractions, naming, code organization
- **Observability** — missing logging, incomplete metrics, insufficient error context

Name the project after the module and concern. A project to address an auth module with mixed responsibility might be named `auth-mixed-responsibilities.yaml` for example.

Complex or unrelated code changes should be in seperate requirements. Coding agents work on one requirement at a time. Focused requirements lets coding agents focus.