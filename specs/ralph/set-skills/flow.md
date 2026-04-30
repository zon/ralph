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

    contents := map[string]string{}
    for _, skill := range skills {
        content, err := fetchSkill(skill, branch)
        if err != nil {
            return err
        }
        contents[skill] = rewriteLinks(content, branch)
    }

    for skill, content := range contents {
        writeSkill(repoRoot, skill, content)
    }

    removeStaleSkills(repoRoot, skills)
    return nil
}
```

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

    setSkills("v2")

    assertSkillsFetchedFromBranch(t, "v2")
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

func TestSetSkills_RewritesRelativeLinks(t *testing.T) {
    inGitRepo()
    skillsAvailable("ralph-write-spec")
    skillContentContains("ralph-write-spec", relativeLink("docs/planning/specs.md"))

    setSkills("main")

    assertInstalledSkillContains(t, "ralph-write-spec", absoluteRalphLink("main", "docs/planning/specs.md"))
}
```
