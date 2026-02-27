# Writing Projects

Projects are YAML files that define work for AI agents. Requirements describe **what should be accomplished**, not how to accomplish it.

## Quick Example

```yaml
name: user-authentication
description: Add user authentication

requirements:
  - category: backend
    description: Authentication API
    items:
      - Users can register with email and password
      - Users can log in with valid credentials
      - JWT tokens are issued on successful authentication
    passing: false

  - category: frontend
    description: Authentication UI
    items:
      - Users can access login and registration forms
      - Login redirects to dashboard on success
      - Invalid credentials show error messages
    passing: false
```

## File Format

```yaml
name: project-identifier          # Used for branch naming (ralph/<name>)
description: Brief description    # Used in PR title

requirements:
  - category: backend              # Group related requirements
    description: What to accomplish
    items:
      - Specific outcome 1
      - Specific outcome 2
    passing: false                 # false = needs work, true = complete
```

## Writing Good Requirements

**Focus on outcomes, not implementation:**

✅ Good:
- Users can log in with email and password
- Invalid credentials are rejected with error messages
- Session tokens expire after 24 hours

❌ Bad:
- Create login API endpoint
- Add password validation function
- Implement JWT expiration middleware

**Guidelines:**
- Write from user/system perspective ("Users can...", "System validates...")
- Be specific about expected behavior
- Break complex work into multiple requirements
- Order items logically when dependent

## Ralph Workflow

1. Creates branch `ralph/<project-name>`
2. Runs `before` commands from `.ralph/config.yaml` (compilation, codegen, setup, etc.)
3. Starts services from `.ralph/config.yaml`
4. For each requirement where `passing: false`:
   - AI implements and validates
   - Updates status in report.md
5. Commits changes and creates PR
6. Stops services

## Before Commands

The `before` key in `.ralph/config.yaml` defines commands that run once before services start and before the iteration loop begins.

```yaml
before:
  - name: compile
    command: go
    args: [build, -o, bin/app, ./cmd/app]
    workDir: /path/to/project  # optional

  - name: generate
    command: make
    args: [generate]
```

- Commands run sequentially and must exit successfully before ralph proceeds
- Each entry requires `name` and `command`; `args` and `workDir` are optional
- Useful for compilation, code generation, dependency installation, database migrations, etc.

**Single iteration mode** (`--once`): Runs one iteration without branching/PR, useful for local testing.

**Argo Workflow submission** (default): Submits the workflow to Kubernetes using Argo Workflows. Use `--local` to run on this machine instead. See [remote-execution.md](remote-execution.md) for details.

## Examples

### Feature Addition

```yaml
name: csv-export
description: Add CSV export functionality

requirements:
  - category: backend
    description: CSV export endpoint
    items:
      - Users can request data export in CSV format
      - Export endpoint requires authentication
      - All user records are included
    passing: false
```

### Bug Fix

```yaml
name: fix-pagination
description: Fix pagination edge cases

requirements:
  - category: backend
    description: Pagination handles edge cases
    items:
      - Last page shows correct number of items
      - Empty pages return appropriate response
      - Page size limits are enforced
    passing: false
```

### Multi-Component

```yaml
name: notifications
description: User notification system

requirements:
  - category: database
    description: Notification persistence
    items:
      - Notifications stored with user, message, timestamp
      - Read/unread status tracked
    passing: false

  - category: backend
    description: Notification API
    items:
      - Users can fetch notifications
      - Users can mark as read
    passing: false
    
  - category: frontend
    description: Notification UI
    items:
      - Notification bell shows unread count
      - Dropdown displays notification list
    passing: false
```

## Tips

- Start small with one requirement
- Use `--dry-run` to preview
- Trust the AI - focus on outcomes
- File naming: `feature-name.yaml` not `project1.yaml`
