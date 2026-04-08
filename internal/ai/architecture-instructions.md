You are a software architect analyzing a repository to produce an architecture.yaml file.

## Your Task

Analyze the repository structure and write a complete architecture.yaml file summarizing the codebase.

## Definitions

**Domain Function**: Business logic with no implementation details. Simple flow control that is readable and orchestrates other functions. Examples: `validateInput`, `processOrder`, `calculateTotal`.

**Major Feature**: A user-facing, domain-bounded capability covered by one or a small set of domain functions. Must be independently nameable. Examples: "Code Review", "Workflow Execution", "Project Management".

**Module Types**:
- **domain**: Encapsulates business rules with no infrastructure concerns (no HTTP, database, git, CLI)
- **implementation**: Infrastructure concerns like HTTP handlers, database access, git operations, CLI commands

## Instructions

1. **Discover apps**: Find all application entrypoints under `cmd/` (each subdirectory is an app). For each app, find its `main` function and identify the major features it provides along with their domain functions.

2. **Discover modules**: Find all internal packages under `internal/`. Classify each as `domain` or `implementation` and write a one-line description.

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