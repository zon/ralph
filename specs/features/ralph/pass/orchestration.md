# Pass Orchestration

## Purpose

Mark a project requirement as passing or failing by slug, updating the project YAML file in place.

## Orchestration

**Module:** `internal/orchestration/pass`

```go
type PassCmd struct {
    ProjectFile string `arg:""`
    Slug        string `arg:""`
    False       bool   `name:"false"`
}

func (c *PassCmd) Run() error {
    proj, err := project.LoadProject(c.ProjectFile)
    if err != nil {
        return err
    }

    if err := project.UpdateRequirementStatus(proj, c.Slug, !c.False); err != nil {
        return err
    }

    return project.SaveProject(c.ProjectFile, proj)
}
```

### Helpers

- **`project.LoadProject(path)`** — reads and validates the project YAML file at the given path
- **`project.UpdateRequirementStatus(proj, slug, passing)`** — finds the requirement by slug and sets its `passing` field; returns an error if the slug does not exist
- **`project.SaveProject(path, proj)`** — marshals the project and writes it back to the given path

## Tests

**Module:** `internal/orchestration/pass`

```go
func TestPassCmd_MarkPassing(t *testing.T) {
    path := project.FileWithRequirement(t, "my-req", false)
    cmd := &PassCmd{ProjectFile: path, Slug: "my-req"}
    require.NoError(t, cmd.Run())
    assert.True(t, project.RequirementStatus(t, path, "my-req"))
}

func TestPassCmd_MarkFailing(t *testing.T) {
    path := project.FileWithRequirement(t, "my-req", true)
    cmd := &PassCmd{ProjectFile: path, Slug: "my-req", False: true}
    require.NoError(t, cmd.Run())
    assert.False(t, project.RequirementStatus(t, path, "my-req"))
}

func TestPassCmd_FileNotFound(t *testing.T) {
    cmd := &PassCmd{ProjectFile: project.NonExistentFile(t), Slug: "my-req"}
    assert.Error(t, cmd.Run())
}

func TestPassCmd_SlugNotFound(t *testing.T) {
    path := project.FileWithRequirement(t, "my-req", false)
    cmd := &PassCmd{ProjectFile: path, Slug: "unknown-slug"}
    assert.Error(t, cmd.Run())
}
```

### Helpers

- **`project.FileWithRequirement(t, slug, passing)`** — writes a valid single-requirement project YAML to a temp file and returns the path; owned by `internal/project` since it constructs a `Project`
- **`project.RequirementStatus(t, path, slug)`** — loads the project at path and returns the `passing` value for the named requirement
- **`project.NonExistentFile(t)`** — returns a path inside `t.TempDir()` that is guaranteed not to exist
