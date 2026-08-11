# Writing Good Requirements

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
- Give each item a `slug`, `id`, or `name` so commits read `Ralph item 2 (login) completed` rather than `Ralph item 2 completed`

**Do not include** work ralph handles automatically — it runs tests and fixes failures on its own. Entries like "all existing tests pass" or "no regressions" are redundant.
