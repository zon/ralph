# Blocked

## Issue
Cyclic import dependencies preventing complete separation of project management logic from configuration loading.

## What we tried
1. Moved Project and Requirement structs, LoadProject, ValidateProject, SaveProject, CheckCompletion, UpdateRequirementStatus functions from `internal/config` to `internal/project`.
2. Updated many callers in `internal/project` (iteration.go, complete.go, execute.go) to use the moved functions directly (same package).
3. Updated `internal/requirement/execute.go` to import `project` and use `project.LoadProject`/`project.CheckCompletion`, but this created a direct import cycle because `project` already imports `requirement` (for `Execute`).
4. Attempted to break the cycle by moving `requirement.Execute` to `project` package (as per another requirement). Created `internal/project/requirement_execution.go` and updated `iteration.go` to call `ExecuteDevelopmentIteration`. This resolved the cycle between `project` and `requirement`.
5. However, new import cycles emerged:
   - `ai` package uses `config.LoadProject` and `config.CheckCompletion`. Changing `ai` to import `project` creates a cycle because `project` imports `ai` (for `RunAgent`).
   - `config` tests (`config_test.go`) depend on the removed types and functions, requiring either moving tests to `project` package or keeping delegation functions in `config`.

## Remaining problems
- **Import cycle `ai` ↔ `project`**: `ai.GeneratePRSummary` and `ai.GenerateChangelog` are used by `project` but also need to call project management functions. Moving these AI functions to `project` would require also moving helper functions (`runOpenCodeAndReadResult`, `resolveModel`, etc.) and may increase coupling.
- **Test migration**: `config_test.go` contains 29 test functions, many of which test the moved project management functions. Moving them to `project` package is nontrivial and would break existing test organization.
- **Other callers**: `cmd/comment.go`, `cmd/validate.go`, `cmd/review.go`, `ai/ai.go` still reference `config.LoadProject` and need updating, but updating `ai` introduces the import cycle.

## Possible solutions
1. Keep thin delegation wrappers in `config` that forward to `project` (violates requirement but unblocks).
2. Move `ai.GeneratePRSummary` and `ai.GenerateChangelog` to `project` package, breaking the cycle (requires moving helper functions and may affect other AI functionality).
3. Refactor `ai` package to accept project management functions as dependencies (dependency injection), requiring changes to callers.

Given time constraints, we need guidance on which direction to proceed.