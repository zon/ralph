# Writing Projects

Projects are YAML files that define work for AI agents.

## Format

```yaml
name: project-identifier          # Used for branch naming (ralph/<name>)
description: Brief description    # Used in PR title

requirements:
  - category: backend             # Group related requirements
    description: What to accomplish
    items:
      - Specific outcome 1
      - Specific outcome 2
    passing: false                # false = needs work, true = complete
```

A project can have multiple requirements across different categories. Ralph reads all requirements where `passing: false` and picks the highest priority to work on.

## Naming Projects

The `name` field becomes the branch name: `ralph/<name>`. Use lowercase, hyphen-separated identifiers:

- `user-authentication`
- `fix-pagination`
- `csv-export`

Name your project file to match: `user-authentication.yaml`.

## Writing Good Requirements

Requirements describe **what should happen** and may define high-level interfaces, but should not include low-level implementation detail.

✅ Good:
- Users can log in with email and password
- Invalid credentials are rejected with error messages
- Session tokens expire after 24 hours
- `POST /auth/login` accepts `{ email, password }` and returns a JWT

❌ Bad:
- Add password validation function
- Implement JWT expiration middleware
- Use bcrypt with cost factor 12

**Guidelines:**
- Write from the user, client, or developer perspective — user interfaces, network interfaces, and high-level APIs
- Be specific about expected behavior
- Break complex work into multiple requirements

**Do not include** items ralph handles automatically — it runs tests and fixes failures on its own. Items like "all existing tests pass" or "no regressions" are redundant.

## Version Bumps

When a project warrants a version bump, add a `version` requirement specifying the bump level — not the target version number. Ralph determines the current version and applies the bump itself.

✅ Good:
- Apply a semver patch bump to `internal/version/VERSION` and `charts/ralph-webhook/Chart.yaml`

❌ Bad:
- Bump version to 3.2.11
- Set `appVersion` to "3.2.11" and `version` to "0.2.62"

Use **patch** for bug fixes and small additions, **minor** for new features, and **major** for breaking changes.
