# Writing Requirements

An [item](glossary.md#item) is one iteration's worth of work. Items describe **what should happen** and may define high-level interfaces, but should not include low-level implementation detail.

## Good vs Bad Examples

✅ Good:
- Users can log in with email and password
- Invalid credentials are rejected with error messages
- Session tokens expire after 24 hours
- `POST /auth/login` accepts `{ email, password }` and returns a JWT

❌ Bad:
- Add password validation function
- Implement JWT expiration middleware
- Use bcrypt with cost factor 12

## Guidelines

- Write from the user, client, or developer perspective — user interfaces, network interfaces, and high-level APIs
- Be specific about expected behavior
- Break complex work into multiple items — one item is one iteration, so an item that needs three separate rounds of work should be three items
- Give each item a `slug`, `id`, or `name` so commit messages name the work rather than just its index

**Do not include** work the runner handles automatically. Most runners run the test suite and fix failures on their own, so entries like "all existing tests pass" or "no regressions" are redundant. Check what your runner does before adding an item for it.

## Where Implementation Detail Belongs

The prohibition above is about the item text, not about the project. Implementation shapes do belong in a project — in the `code` and `tests` fields, sourced from the feature's [orchestration document](orchestration.md) rather than invented. See [Project Format](project.md#code-and-tests).

The distinction is authorship. An item's prose states the outcome, and its `code` entries relay a shape someone already designed. What must not happen is an agent making architecture decisions inside an item because the item told it to "implement middleware".
