# Review Projects

Review projects are [Ralph projects](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/projects.md) written to address issues found during code review. Review projects concern themselves with code architecture and organization rather than adding new domain logic.

Ignore the typical guidelines for [writing good requirements](https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/writing-requirements.md) when writing a review project. Instead, write requirements in a code-review style: each requirement item must describe the specific problem found in the code and explain concretely how to fix it. Include file paths and function names where relevant.

Example requirement items:
- `internal/foo/bar.go` - `processItems` allocates a new slice on every call; pre-allocate with the known capacity before the loop
- `internal/auth/token.go` - `ValidateToken` does not check token expiry; add an expiry check and return an error if the token is expired
