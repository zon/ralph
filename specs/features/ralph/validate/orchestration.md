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
}

func (v *Validator) Validate(path string) (*project.Project, error) {
    for attempt := 1; attempt <= MaxAttempts; attempt++ {
        proj, loadErr := v.project.Load(path)
        if loadErr == nil {
            if saveErr := v.project.Save(path, proj); saveErr != nil {
                return nil, saveErr
            }
            return proj, nil
        }
        if attempt == MaxAttempts {
            return nil, loadErr
        }
        if fixErr := v.agent.FixProject(path, loadErr); fixErr != nil {
            return nil, fixErr
        }
    }
    return nil, ErrUnreachable
}
```

### Helpers

- **`v.project.Load(path)`** — reads the file at `path` and unmarshals it into a project, returning a schema error if the contents are not a well-formed project
- **`v.project.Save(path, proj)`** — marshals the project back to disk at `path` using the canonical layout shared with `ralph pass`
- **`v.agent.FixProject(path, loadErr)`** — invokes the AI agent locally to rewrite the file at `path` so it parses, using the configured ralph model and the most recent load error as context

## Tests

**Module:** `internal/validate`

```go
func TestValidateSucceedsOnFirstLoad(t *testing.T) {
    proj := project.any()
    svc := validate.withMocks(
        validate.withProject(project.thatLoads(proj)),
    )
    result, err := svc.Validate(project.anyPath())
    require.NoError(t, err)
    require.Equal(t, proj, result)
    require.Equal(t, proj, project.lastSaved())
    require.Empty(t, agent.fixCalls())
}

func TestValidateRepairsThenSucceeds(t *testing.T) {
    proj := project.any()
    svc := validate.withMocks(
        validate.withProject(project.thatLoadsAfterFailures(1, proj)),
    )
    result, err := svc.Validate(project.anyPath())
    require.NoError(t, err)
    require.Equal(t, proj, result)
    require.Len(t, agent.fixCalls(), 1)
}

func TestValidateGivesUpAfterMaxAttempts(t *testing.T) {
    svc := validate.withMocks(
        validate.withProject(project.thatAlwaysFailsToLoad()),
    )
    _, err := svc.Validate(project.anyPath())
    require.Error(t, err)
    require.Len(t, agent.fixCalls(), validate.MaxAttempts-1)
}

func TestValidatePropagatesAgentFailure(t *testing.T) {
    svc := validate.withMocks(
        validate.withProject(project.thatAlwaysFailsToLoad()),
        validate.withAgent(agent.thatFailsToFix()),
    )
    _, err := svc.Validate(project.anyPath())
    require.Error(t, err)
}

func TestValidatePropagatesSaveFailure(t *testing.T) {
    svc := validate.withMocks(
        validate.withProject(project.thatLoadsButFailsToSave(project.any())),
    )
    _, err := svc.Validate(project.anyPath())
    require.Error(t, err)
}
```

### Helpers

- **`validate.withMocks(opts...)`** — constructs a `Validator` with default mock implementations; pass option helpers to configure specific dependencies
- **`validate.withProject(client)`** — option that sets the project client on the mock validator
- **`validate.withAgent(client)`** — option that sets the agent client on the mock validator
- **`project.any()`** — returns a valid project value owned by `internal/project`
- **`project.anyPath()`** — returns a project file path suitable for use in tests
- **`project.thatLoads(proj)`** — returns a project client whose `Load` returns `proj` and whose `Save` records the saved value
- **`project.thatLoadsAfterFailures(n, proj)`** — returns a project client whose `Load` fails `n` times and then returns `proj`
- **`project.thatAlwaysFailsToLoad()`** — returns a project client whose `Load` always returns an error
- **`project.thatLoadsButFailsToSave(proj)`** — returns a project client whose `Load` returns `proj` and whose `Save` returns an error
- **`project.lastSaved()`** — returns the most recent project passed to `Save` during the test
- **`agent.thatFailsToFix()`** — returns an agent client whose `FixProject` returns an error
- **`agent.fixCalls()`** — returns the list of `(path, error)` pairs passed to `FixProject` during the test
