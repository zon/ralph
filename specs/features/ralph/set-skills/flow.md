# Set Skills Flow

## Purpose
Install ralph's skills from a source branch of the ralph repository into the current target repository.

## Flow

**Module:** `internal/setup`

```go
func SetSkills(branch string) error {
    root, err := git.RepoRoot()
    if err != nil {
        return err
    }

    names, err := skills.Discover(branch)
    if err != nil {
        return err
    }

    fetched, err := skills.FetchAll(branch, names)
    if err != nil {
        return err
    }

    skills.PruneStale(root, fetched)
    return skills.InstallAll(root, fetched)
}
```

### Helpers

- **`git.RepoRoot()`** — returns the root directory of the git repository containing the current working directory, or an error when the working directory is not inside a repository
- **`skills.Discover(branch)`** — lists the ralph-prefixed skill names available on the given branch of the ralph repository
- **`skills.FetchAll(branch, names)`** — fetches each skill's contents from the given branch and returns them with their links already rewritten to the resolved branch; any fetch failure aborts the batch
- **`skills.PruneStale(root, fetched)`** — removes ralph-prefixed skill directories from the target repository that are not present in the fetched set
- **`skills.InstallAll(root, fetched)`** — writes the fetched skills into `.claude/skills/` under the target repository root, overwriting existing entries with the same name

## Tests

**Module:** `internal/setup`

```go
func TestSetSkills_InstallsRalphSkills(t *testing.T) {
    target := repos.targetRepo(t)
    source := skills.sourceBranch(t).
        with(skills.aRalphSkill("ralph-write-spec")).
        with(skills.aNonRalphSkill("internal-tool"))

    err := SetSkills(source.branch())

    skills.requireOK(t, err)
    skills.requireInstalled(t, target, "ralph-write-spec")
    skills.requireNotInstalled(t, target, "internal-tool")
}

func TestSetSkills_OverwritesExistingSkill(t *testing.T) {
    target := repos.targetRepo(t).with(skills.anInstalledSkill("ralph-write-spec", "old"))
    source := skills.sourceBranch(t).with(skills.aRalphSkill("ralph-write-spec").withBody("new"))

    err := SetSkills(source.branch())

    skills.requireOK(t, err)
    skills.requireInstalledBody(t, target, "ralph-write-spec", "new")
}

func TestSetSkills_RemovesStaleRalphSkill(t *testing.T) {
    target := repos.targetRepo(t).with(skills.anInstalledSkill("ralph-old-skill", "stale"))
    source := skills.sourceBranch(t).with(skills.aRalphSkill("ralph-write-spec"))

    err := SetSkills(source.branch())

    skills.requireOK(t, err)
    skills.requireNotInstalled(t, target, "ralph-old-skill")
}

func TestSetSkills_LeavesNonRalphSkillsUntouched(t *testing.T) {
    target := repos.targetRepo(t).with(skills.anInstalledSkill("my-custom-skill", "mine"))
    source := skills.sourceBranch(t).with(skills.aRalphSkill("ralph-write-spec"))

    err := SetSkills(source.branch())

    skills.requireOK(t, err)
    skills.requireInstalledBody(t, target, "my-custom-skill", "mine")
}

func TestSetSkills_RewritesLinksToResolvedBranch(t *testing.T) {
    target := repos.targetRepo(t)
    source := skills.sourceBranch(t).onBranch("v2").
        with(skills.aRalphSkill("ralph-write-spec").linkingTo("docs/formats/specs.md"))

    err := SetSkills("v2")

    skills.requireOK(t, err)
    skills.requireSkillLink(t, target, "ralph-write-spec", skills.rawURL("v2", "docs/formats/specs.md"))
}

func TestSetSkills_DiscoveryFailureWritesNothing(t *testing.T) {
    target := repos.targetRepo(t)
    skills.sourceBranch(t).failsDiscovery()

    err := SetSkills(skills.defaultBranch())

    skills.requireError(t, err)
    skills.requireEmpty(t, target)
}

func TestSetSkills_FetchFailureWritesNothing(t *testing.T) {
    target := repos.targetRepo(t)
    skills.sourceBranch(t).
        with(skills.aRalphSkill("ralph-write-spec")).
        failsFetchFor("ralph-write-spec")

    err := SetSkills(skills.defaultBranch())

    skills.requireError(t, err)
    skills.requireEmpty(t, target)
}

func TestSetSkills_OutsideGitRepoErrors(t *testing.T) {
    repos.outsideAnyRepo(t)

    err := SetSkills(skills.defaultBranch())

    skills.requireError(t, err)
}
```

### Helpers

- **`repos.targetRepo(t)`** — initializes a temporary git repository, makes it the current working directory, and returns a handle for assertions
- **`repos.outsideAnyRepo(t)`** — switches the current working directory to a location not inside any git repository
- **`skills.sourceBranch(t)`** — returns a builder for the simulated ralph source branch served to discovery and fetch helpers
- **`skills.aRalphSkill(name)`** — returns a builder for a skill whose directory name has the `ralph-` prefix, populated with default content
- **`skills.aNonRalphSkill(name)`** — returns a builder for a skill whose directory name lacks the `ralph-` prefix
- **`skills.anInstalledSkill(name, body)`** — pre-populates a skill in the target repository with the given body
- **`skills.defaultBranch()`** — returns the branch name used when the user does not pass `--branch`
- **`skills.rawURL(branch, path)`** — constructs the expected raw content URL for a path on a given branch
- **`skills.requireOK(t, err)`** — asserts the flow completed without error
- **`skills.requireError(t, err)`** — asserts the flow returned an error
- **`skills.requireInstalled(t, target, name)`** — asserts the named skill exists in the target repository
- **`skills.requireNotInstalled(t, target, name)`** — asserts the named skill does not exist in the target repository
- **`skills.requireInstalledBody(t, target, name, body)`** — asserts the named skill exists with the given body
- **`skills.requireSkillLink(t, target, name, url)`** — asserts the installed skill contains the given link
- **`skills.requireEmpty(t, target)`** — asserts no skills were written to the target repository
