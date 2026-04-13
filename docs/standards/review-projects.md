# Review Projects

Review projects are [Ralph projects](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/projects.md) written to address issues found during code review. Review projects concern themselves with code architecture and organization rather than adding new domain logic.

Ignore the typical guidelines for [writing good requirements](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/writing-requirements.md) when writing a review project. Instead, write requirements in a code-review style: each requirement item must describe the specific problem found in the code and explain concretely how to fix it. Include file paths and function names where relevant.

Example requirement items:
- `internal/foo/bar.go` - `processItems` allocates a new slice on every call; pre-allocate with the known capacity before the loop
- `internal/auth/token.go` - `ValidateToken` does not check token expiry; add an expiry check and return an error if the token is expired

## Grouping Review Projects

Group review projects by **module and concern type**. Each project should address one concern type within one module.

**Modules** correspond to Go packages (e.g. `internal/auth`, `internal/ai`).

**Concern types:**
- **Correctness** — missing validation, incorrect logic, error handling gaps, security issues
- **Performance** — unnecessary allocations, redundant work, inefficient data structures
- **Structure** — misplaced responsibilities, poor abstractions, naming, code organization
- **Observability** — missing logging, incomplete metrics, insufficient error context

For example: "auth correctness", "ai performance", "webhook structure". If issues in the same module span multiple concern types, create a separate project for each.
