# Plan: Domain Function

You are an architect. Your task is to plan how to best represent a major feature from `outlines/major-features.md` as one or more domain functions, following the standard in `docs/standards/domain-functions.md`. Produce `./plan/{major-feature}.md` where `{major-feature}` is a lowercase, hyphen-separated name matching the feature.

## What to Read First

1. `docs/standards/domain-functions.md` — understand what a domain function is and what it must not contain.
2. `outlines/major-features.md` — find the major feature you are planning for. Read its description, user interface, and business logic bullets carefully.

## How to Plan

1. Identify the top-level entry point for the feature. This is the domain function that represents the feature's primary action — the one a reader would look at first to understand what the feature does.
2. Decompose the business logic bullets into named steps. Each step that encodes a real-world rule or process is a candidate for a helper domain function called by the top-level function.
3. Name functions using plain verbs and nouns that reflect domain language, not technical infrastructure (e.g., `openPullRequest`, not `callGithubAPI`).
4. Keep each function's body to simple, linear flow: assignments, calls, conditionals, and returns only. No loops of technical mechanics, no error handling boilerplate, no implementation details.
5. Do not plan implementation modules (HTTP clients, database queries, Kubernetes calls). Those are infrastructure and belong outside domain functions.

## Output Format

Produce `./plan/{major-feature}.md` with the following structure. Identify the primary language of the codebase and write all function signatures and bodies in that language.

```markdown
# Plan: {Feature Name}

## Overview

<One or two sentences on what domain functions are needed and why this decomposition fits the feature.>

## Domain Functions

### `{functionName}(ctx)`

<One sentence on what real-world action this function performs.>

```{lang}
func {functionName}(ctx) {
  x = {step}(ctx)
  y = {step}(x)
  {step}(x, y)
  return y
}
```

### `{helperFunctionName}(x)`

<One sentence on the rule or process this helper encodes.>

```{lang}
func {helperFunctionName}(x) {
  if {condition} {
    {action}
  }
  return {result}
}
```
```

List the top-level function first, then helpers in call order. Every function in the plan must map to at least one business logic bullet from the feature outline.
