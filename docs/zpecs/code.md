# Code Guidelines

We want clear, minimal code that is easy to understand and replace. Code should meet our requirements and standards but in the simplest possible way.

## Writing

* Each function and block should only do one specific thing
* Match current requirements and standards instead of the past or future
* Avoid functions with deep nesting. Return early or consider better architecture to keep code shallow and easy to read
* Code should be self-evident. Only write comments to summarize when clear code is infeasible

## Refactoring

* Gradually move towards ideal architecture and implementation. Always look for clearer, more minimal solutions
* Always refactor within the scope of work
* File refactoring ideas that are out of the work scope in /refactoring.md

### refactoring.md

This is a Markdown file where every section is a refactoring plan. Each plan should be self-contained and narrow in scope.

Refactoring plans should be removed from the file after they are completed. Remove the /refactoring.md file when it contains no plans.