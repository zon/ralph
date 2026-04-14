# Plan: Project Execution

## Overview

Five domain functions are needed: one top-level function that orchestrates the full execution lifecycle (validate, iterate, commit, open PR), and four helpers that each encode a distinct business rule — validating the project, running the iteration loop, committing one iteration's work, and deciding whether to open a pull request.

## Domain Functions

### `executeProject(ctx)`

Validates the project, runs development iterations until all requirements pass, then opens a pull request if commits were produced.

```go
func executeProject(ctx) {
  project = loadAndValidateProject(ctx)
  switchToBranch(project.name)
  startServices(ctx)
  iterationCount = runIterationLoop(ctx, project)
  commitLog = getCommitLog(ctx.baseBranch)
  prSummary = generatePRSummary(ctx, project, commitLog)
  openPullRequestIfAhead(ctx, project, prSummary)
}
```

### `loadAndValidateProject(ctx)`

Rejects a project that has no name or no requirements before any AI resources are consumed.

```go
func loadAndValidateProject(ctx) {
  project = loadProject(ctx.projectFile)
  if project.name == "" {
    return error("project name is required")
  }
  if len(project.requirements) == 0 {
    return error("project must have at least one requirement")
  }
  return project
}
```

### `runIterationLoop(ctx, project)`

Repeatedly selects the highest-priority failing requirement and directs the AI agent to implement it, stopping when all requirements pass, a blocked signal is detected, a billing error occurs, or the iteration ceiling is reached.

```go
func runIterationLoop(ctx, project) {
  for iteration = 1; iteration <= ctx.maxIterations; iteration++ {
    if isBlocked() {
      return error("blocked.md detected")
    }
    err = runSingleIteration(ctx, project, iteration)
    if isBillingError(err) {
      return error("billing or quota error")
    }
    project = reloadProject(ctx.projectFile)
    if allRequirementsPassing(project) {
      return iteration
    }
  }
  return error("max iterations reached with requirements still failing")
}
```

### `runSingleIteration(ctx, project, iteration)`

Runs the AI agent against the highest-priority failing requirement, then commits the resulting changes with a message derived from `report.md` or auto-generated from the diff.

```go
func runSingleIteration(ctx, project, iteration) {
  requirement = selectHighestPriorityFailingRequirement(project)
  runAIAgent(ctx, requirement)
  commitMessage = getOrGenerateCommitMessage(ctx)
  commitChanges(ctx, commitMessage)
}
```

### `openPullRequestIfAhead(ctx, project, prSummary)`

Opens a pull request only when the branch has commits ahead of the base; skips PR creation if all requirements were already passing before any iterations ran.

```go
func openPullRequestIfAhead(ctx, project, prSummary) {
  if !hasCommitsAheadOfBase(ctx.baseBranch) {
    return
  }
  prURL = createPullRequest(ctx, project, prSummary)
  return prURL
}
```
