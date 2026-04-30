# Set Skills Flow

## Purpose

Discover, fetch, and install ralph-prefixed skills from the ralph GitHub repository into the current git repository.

## Flow

```go
func setSkills(branch string) error {
    repoRoot, err := findGitRoot()
    if err != nil {
        return ErrNotInGitRepo
    }

    skills, err := discoverSkills(branch)
    if err != nil {
        return err
    }

    contents, err := fetchAll(skills, branch)
    if err != nil {
        return err
    }

    installSkills(repoRoot, contents)
    removeStaleSkills(repoRoot, skills)
    return nil
}
```

### Helpers

- **`findGitRoot()`** — locates the root directory of the current git repository
- **`discoverSkills(branch)`** — queries GitHub for ralph-prefixed skills on the given branch
- **`fetchAll(skills, branch)`** — fetches and rewrites each skill's content; returns an error if any fetch fails, leaving nothing written
- **`installSkills(repoRoot, contents)`** — writes all skill files into the target repository
- **`removeStaleSkills(repoRoot, skills)`** — deletes any installed ralph-prefixed skills not present in the current set

## Tests

```go
func TestSetSkills_Success(t *testing.T) {
    inGitRepo()
    skillsAvailable("ralph-write-spec", "ralph-write-flow")

    err := setSkills("main")

    assert.NoError(t, err)
    assertSkillsInstalled(t, "ralph-write-spec", "ralph-write-flow")
}

func TestSetSkills_NotInGitRepo(t *testing.T) {
    notInGitRepo()

    err := setSkills("main")

    assert.ErrorIs(t, err, ErrNotInGitRepo)
    assertNoSkillsWritten(t)
}

func TestSetSkills_DiscoveryFailure(t *testing.T) {
    inGitRepo()
    discoveryWillFail()

    err := setSkills("main")

    assert.Error(t, err)
    assertNoSkillsWritten(t)
}

func TestSetSkills_FetchFailure(t *testing.T) {
    inGitRepo()
    skillsAvailable("ralph-write-spec")
    fetchWillFail("ralph-write-spec")

    err := setSkills("main")

    assert.Error(t, err)
    assertNoSkillsWritten(t)
}

func TestSetSkills_BranchOverride(t *testing.T) {
    inGitRepo()
    skillsAvailableOnBranch("ralph-write-spec", "v2")

    err := setSkills("v2")

    assert.NoError(t, err)
    assertSkillsInstalled(t, "ralph-write-spec")
}

func TestSetSkills_RemovesStaleSkills(t *testing.T) {
    inGitRepo()
    existingSkill("ralph-old-skill")
    skillsAvailable("ralph-write-spec")

    setSkills("main")

    assertSkillsInstalled(t, "ralph-write-spec")
    assertSkillRemoved(t, "ralph-old-skill")
}

func TestSetSkills_LeavesNonRalphSkillsUntouched(t *testing.T) {
    inGitRepo()
    existingSkill("my-custom-skill")
    skillsAvailable("ralph-write-spec")

    setSkills("main")

    assertSkillPresent(t, "my-custom-skill")
}
```

### Helpers

- **`inGitRepo()`** — sets up the current directory as inside a git repository
- **`notInGitRepo()`** — sets up the current directory as outside any git repository
- **`skillsAvailable(names...)`** — configures the discovery API to return the given skills on the default branch
- **`skillsAvailableOnBranch(name, branch)`** — configures the discovery API to return the skill on the specified branch
- **`discoveryWillFail()`** — configures the discovery API to return an error
- **`fetchWillFail(skill)`** — configures the fetch for the named skill to return an error
- **`existingSkill(name)`** — places a skill directory in the target repository before the test runs
- **`assertSkillsInstalled(t, names...)`** — asserts each named skill is present in the target repository
- **`assertNoSkillsWritten(t)`** — asserts no skill files were written to the target repository
- **`assertSkillRemoved(t, name)`** — asserts the named skill no longer exists in the target repository
- **`assertSkillPresent(t, name)`** — asserts the named skill exists unchanged in the target repository
