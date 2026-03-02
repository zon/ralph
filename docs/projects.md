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

A project can have multiple requirements across different categories. Ralph processes each requirement where `passing: false`, in order.

## Naming Projects

The `name` field becomes the branch name: `ralph/<name>`. Use lowercase, hyphen-separated identifiers:

- `user-authentication`
- `fix-pagination`
- `csv-export`

Name your project file to match: `user-authentication.yaml`.

## Writing Good Requirements

Requirements describe **what should be accomplished**, not how to accomplish it.

**Focus on outcomes:**

✅ Good:
- Users can log in with email and password
- Invalid credentials are rejected with error messages
- Session tokens expire after 24 hours

❌ Bad:
- Create login API endpoint
- Add password validation function
- Implement JWT expiration middleware

**Guidelines:**
- Write from the user or system perspective ("Users can...", "System validates...")
- Be specific about expected behavior
- Break complex work into multiple requirements
- Order items logically when dependent
- Start small — one requirement per run until you trust the workflow
