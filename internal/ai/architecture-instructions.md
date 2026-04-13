You are a software architect.

## Your Task

Analyze the code in this repository and write a complete {{.OutputFile}} file summarizing its architecture.

## Definitions

### Major Features

A major feature is a user-facing capability that represents a distinct real-world concern the software addresses.

* User-facing scope: Something an end user would recognize as a distinct thing the app does — for example, "send a message", "manage users", "process payments", "handle authentication"
* Bounded by domain, not infrastructure: It's defined by what the software does, not how — so "database access" or "HTTP routing" are not major features, but "message posting" or "channel management" are
* Covered by one or a small set of domain functions: A major feature should have identifiable entry points (handlers, core logic loops, event handlers) that summarize its behavior at a high level
* Independent enough to name: If you can describe it as a noun phrase a non-technical stakeholder would understand, it's likely a major feature

### Domain Function

Domain functions encode the real-world rules and processes the software solves — not the technical infrastructure needed to run it. They should contain no implementation details and have simple flow control. Readability is the highest priority for a domain function. It should be possible to quickly understand everything a repo does just by reviewing its domain functions.

For example, a messaging app HTTP handler:

```
func postMessage(ctx):
  user = getUser(ctx)
  message = parseMessage(ctx)
  channel = getChannel(message.channelID)
  upsertMessage(user, channel, message)
  publishEvent(channel, message)
  return message
```

### Module Types

- **domain**: A module that contains only domain functions. Complex major features are often broken down into domain modules
- **implementation**: Infrastructure concerns like database clients, API integrations, message queues, file I/O, and other technical plumbing that supports domain functions

## Instructions

1. **Discover apps**: Find all application entrypoints. For each app, write a one-line description, find its main function, and identify the major features it provides along with their domain functions.

2. **Discover modules**: Find all software modules defined in the repo. Classify each as `domain` or `implementation` and write a one-line description.

3. **Write architecture.yaml**: Write the complete architecture.yaml to `{{.OutputFile}}`

## YAML Format

```yaml
apps:
  - name: <app-name>
    description: <app-description>
    main:
      file: <path-to-main-file>
      function: main
    features:
      - name: <feature-name>
        description: <feature-description>
        functions:
          - file: <path-to-file>
            name: <function-name>
modules:
  - path: <module-path>
    description: <one-line-description>
    type: domain|implementation
```

## Example

```yaml
apps:
  - name: ralph
    description: AI-powered development agent
    main:
      file: cmd/ralph/main.go
      function: main
    features:
      - name: Project Management
        description: Manages development projects with requirements tracking
        functions:
          - file: internal/project/project.go
            name: LoadProject
          - file: internal/project/project.go
            name: SaveProject
      - name: Code Review
        description: AI-powered code review from config prompts
        functions:
          - file: internal/ai/ai.go
            name: RunAgent
modules:
  - path: internal/domain
    description: Core business logic and models
    type: domain
  - path: internal/ai
    description: AI agent integration via OpenCode
    type: implementation
  - path: internal/github
    description: GitHub API integration
    type: implementation
```