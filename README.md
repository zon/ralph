# Ralph

I made a Ralph.

Ralph orchestrates AI coding agents to automate development workflows, from branch creation through pull request submission.

## Features

- 🤖 AI-driven development with OpenCode
- 🔄 One iteration per item, until every item is done
- 📋 Any YAML or JSON file with a list in it can be a project
- 🌿 Automated git operations (branch, commit, push, PR)
- 📝 Completion tracked in the commit log, not in your files
- 🚀 Service management (start/stop dev services)
- 🐙 Remote execution via Argo Workflows on Kubernetes

## Example

A project is any YAML or JSON file containing an array of work items. The simplest one is a list:

```yaml
# user-authentication.yaml
- Users can register with email and password
- Users can log in with valid credentials
- JWT tokens are issued on successful authentication
- Login redirects to the dashboard on success
```

Run it locally:

```bash
ralph user-authentication.yaml --local
```

Ralph creates a branch and picks one item per iteration. The AI agent implements it and ends its commit message with a bare `<branch>-<index>` line naming which item it finished:

```
feat: issue JWT tokens on successful authentication

user-authentication-2
```

That line is the whole tracking mechanism. Ralph reads the branch's commit log each iteration to see what is left, and opens a pull request when nothing is left.

Items can be structured instead of plain strings, and the array can be nested anywhere in the file. Point ralph at it with a [jq](https://jqlang.org/manual/) query:

```yaml
# .ralph/config.yaml
items: .requirements
```

The shape below is one convention among many. Ralph reads only the item array out of a project file. A top-level list, or any other document with a list of work in it, works the same way.

```yaml
# projects/user-authentication.yaml
slug: user-authentication
title: Add user authentication

requirements:
  - slug: register
    description: Users can register with email and password
    scenarios:
      - title: Successful registration
        items:
          - GIVEN an unused email address
          - WHEN POST /auth/register is called
          - THEN a user is created and a JWT is returned

  - slug: login
    description: Users can log in with valid credentials
    items:
      - Invalid credentials return 401 with an error message
      - Session tokens expire after 24 hours
```

Ralph never writes to the project file during a run. An item's `slug`, `id`, or `name` just labels it in logs and picker output. See [Project Files](docs/projects.md) and [Iterations](docs/iterations.md).

The spec, orchestration, architecture, and project conventions are published separately and installed into a repository at `docs/zpecs/` — see the [specs repository](https://github.com/zon/specs). Ralph itself only runs what it is given.

## Installation

### 1. Install Dependencies

- **Go**: [go.dev/dl](https://go.dev/dl)
- **Git**: [git-scm.com](https://git-scm.com/downloads)
- **GitHub CLI**: [cli.github.com](https://cli.github.com/)
- **OpenCode**: [opencode.ai](https://opencode.ai/docs/cli/)

### 2. Install Ralph

```bash
go install github.com/zon/ralph/cmd/ralph@latest
```

Ensure `$GOPATH/bin` is in your PATH:

```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### 3. Authenticate GitHub

```bash
gh auth login
```

### 4. Configure OpenCode

See [OpenCode authentication docs](https://opencode.ai/docs/cli/#auth) for setup instructions.

## More

- [CLI reference](docs/cli.md)
- [Project files](docs/projects.md)
- [Iterations and completion](docs/iterations.md)
- [Remote execution workflows](docs/workflows.md)
- [Configuration reference](docs/config.md)
