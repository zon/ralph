# Validate Orchestration

## Purpose
Load a project YAML file, ask a local agent to repair it when loading fails, and rewrite the file in canonical form once it loads successfully.

## Orchestration

**Module:** `internal/validate`

```go
const MaxAttempts = 10

type Validator struct {
    project ProjectClient
    agent   AgentClient
    model   string
}

func (v *Validator) Validate(path string) (*project.Project, error) {
    for attempt := 1; attempt <= MaxAttempts; attempt++ {
        proj, loadErr := v.project.Load(path)
        if loadErr == nil {
            return proj, v.project.Save(path, proj)
        }
        if attempt == MaxAttempts {
            return nil, loadErr
        }
        before := v.project.ReadFile(path)
        v.agent.FixProject(path, loadErr, v.model)
        after := v.project.ReadFile(path)
        if bytes.Equal(before, after) {
            return nil, ErrNoChange
        }
    }
    return nil, ErrUnreachable
}
```

The `model` field is resolved at wiring time using two-level precedence: `validate.model` from the ralph config takes priority; if unset, the top-level `model` field is used as the fallback.

### Helpers

- **`v.project.Load(path)`** — reads the file at `path` and unmarshals it into a project, returning a schema error if the contents are not a well-formed project
- **`v.project.Save(path, proj)`** — marshals the project back to disk at `path` using the canonical layout shared with `ralph pass`
- **`v.project.ReadFile(path)`** — reads the raw bytes of the file at `path` for change detection before and after an agent fix attempt
- **`v.agent.FixProject(path, loadErr, model)`** — invokes the AI agent locally to rewrite the file at `path` so it parses, using the given model and the most recent load error as context

## Tests

**Module:** `internal/validate`

```go
func TestValidateSucceedsOnFirstLoad(t *testing.T) {
    proj := project.Any()
    svc := withMocks(
        withProject(thatLoads(proj)),
    )
    result, err := svc.Validate(project.AnyPath())
    require.NoError(t, err)
    require.Equal(t, proj, result)
    require.Equal(t, proj, project.LastSaved())
    require.Empty(t, FixCalls())
}

func TestValidateRepairsThenSucceeds(t *testing.T) {
    proj := project.Any()
    svc := withMocks(
        withProject(thatLoadsAfterFailures(1, proj)),
    )
    result, err := svc.Validate(project.AnyPath())
    require.NoError(t, err)
    require.Equal(t, proj, result)
    require.Len(t, FixCalls(), 1)
}

func TestValidateGivesUpAfterMaxAttempts(t *testing.T) {
    svc := withMocks(
        withProject(thatAlwaysFailsToLoad()),
    )
    _, err := svc.Validate(project.AnyPath())
    require.Error(t, err)
    require.Len(t, FixCalls(), MaxAttempts-1)
}

func TestValidateFailsFastWhenAgentMakesNoChange(t *testing.T) {
    svc := withMocks(
        withProject(thatAlwaysFailsToLoadWithUnchangedFile()),
    )
    _, err := svc.Validate(project.AnyPath())
    require.ErrorIs(t, err, ErrNoChange)
    require.Len(t, FixCalls(), 1)
}

func TestValidatePropagatesAgentFailure(t *testing.T) {
    svc := withMocks(
        withProject(thatAlwaysFailsToLoad()),
        withAgent(thatFailsToFix()),
    )
    _, err := svc.Validate(project.AnyPath())
    require.Error(t, err)
}

func TestValidatePropagatesSaveFailure(t *testing.T) {
    svc := withMocks(
        withProject(thatLoadsButFailsToSave(project.Any())),
    )
    _, err := svc.Validate(project.AnyPath())
    require.Error(t, err)
}

func TestValidateUsesValidateSpecificModel(t *testing.T) {
    svc := withMocks(
        withModel("validate-model"),
        withProject(thatLoadsAfterFailures(1, project.Any())),
    )
    _, err := svc.Validate(project.AnyPath())
    require.NoError(t, err)
    require.Equal(t, "validate-model", FixCalls()[0].model)
}

func TestValidateFallsBackToMainModel(t *testing.T) {
    svc := withMocks(
        withModel("main-model"),
        withProject(thatLoadsAfterFailures(1, project.Any())),
    )
    _, err := svc.Validate(project.AnyPath())
    require.NoError(t, err)
    require.Equal(t, "main-model", FixCalls()[0].model)
}
```

### Helpers

- **`withMocks(opts...)`** — constructs a `Validator` with default mock implementations; pass option helpers to configure specific dependencies
- **`withProject(client)`** — option that sets the project client on the mock validator
- **`withAgent(client)`** — option that sets the agent client on the mock validator
- **`withModel(model)`** — option that sets the resolved model string on the mock validator
- **`thatLoads(proj)`** — returns a project client whose `Load` returns `proj` and whose `Save` records the saved value
- **`thatLoadsAfterFailures(n, proj)`** — returns a project client whose `Load` fails `n` times and then returns `proj`
- **`thatAlwaysFailsToLoad()`** — returns a project client whose `Load` always returns an error
- **`thatAlwaysFailsToLoadWithUnchangedFile()`** — returns a project client whose `Load` always fails and whose `ReadFile` always returns the same bytes, simulating an agent that makes no change
- **`thatLoadsButFailsToSave(proj)`** — returns a project client whose `Load` returns `proj` and whose `Save` returns an error
- **`thatFailsToFix()`** — returns an agent client whose `FixProject` returns an error
- **`FixCalls()`** — returns the list of `(path, error, model)` structs recorded from calls to `FixProject` during the test
- **`project.Any()`** — returns a valid project value; defined in `internal/project`
- **`project.AnyPath()`** — returns a project file path suitable for use in tests; defined in `internal/project`
- **`project.LastSaved()`** — returns the most recent project passed to `Save` during the test; defined in `internal/project`
