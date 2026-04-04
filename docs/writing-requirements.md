# Writing Good Requirements

Requirements describe **what should happen** and may define high-level interfaces, but should not include low-level implementation detail.

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
- Break complex work into multiple requirements

**Do not include** items ralph handles automatically — it runs tests and fixes failures on its own. Items like "all existing tests pass" or "no regressions" are redundant.
