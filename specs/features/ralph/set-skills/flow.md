# Set Skills Flow

## Purpose

Discover ralph skills from the source repository and install them into the target repository, removing stale ralph skills and preserving non-ralph skills.

## Flow

**Module:** `setup`

```go
func setSkills(branch string) error {
    repoRoot, err := git.FindRepoRoot()
    if err != nil {
        return err
    }

    skills, err := discoverSkills(branch)
    if err != nil {
        return err
    }

    contents, err := fetchSkillContents(skills, branch)
    if err != nil {
        return err
    }

    rewritten := rewriteLinks(contents, branch)

    if err := removeStaleSkills(repoRoot, skills); err != nil {
        return err
    }

    return writeSkills(repoRoot, rewritten)
}
```

### Helpers

- **`git.FindRepoRoot()`** [`git`] — returns the root of the git repository containing the current working directory, or an error if not inside a git repository
- **`discoverSkills(branch)`** [`skills`] — queries the GitHub Contents API for `.claude/skills` on `branch`; returns the names of entries with a `ralph-` prefix
- **`fetchSkillContents(skills, branch)`** [`skills`] — fetches `SKILL.md` for each skill from the raw GitHub URL on `branch`; returns a map of skill name to raw content
- **`rewriteLinks(contents, branch)`** [`skills`] — rewrites relative links in each `SKILL.md` to absolute raw GitHub URLs and normalizes existing ralph raw URLs to use `branch`
- **`removeStaleSkills(repoRoot, skills)`** [`skills`] — deletes any `.agents/skills/ralph-*` directory in `repoRoot` whose name is not present in `skills`, and removes the corresponding `.claude/skills/<skill>/` directory for each deleted skill
- **`writeSkills(repoRoot, contents)`** [`skills`] — writes each skill's `SKILL.md` to `.agents/skills/<skill>/SKILL.md` under `repoRoot`, creating directories as needed, then creates a symbolic link at `.claude/skills/<skill>/SKILL.md` pointing to `.agents/skills/<skill>/SKILL.md`

## Tests

**Module:** `setup`

```go
test("skills installed successfully", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), "ralph-write-spec", "ralph-write-flow")
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-flow", "ralph-write-spec"})
    assert.Equal(t, claudeLinks(t, repo), []string{"ralph-write-flow", "ralph-write-spec"})
})

test("not in git repo returns error", func(t *testing.T) {
    noRepo(t)
    assert.Error(t, setSkills(defaultBranch()))
})

test("discovery failure returns error without writing files", func(t *testing.T) {
    repo := aRepo(t)
    discoveryWillFail(t, defaultBranch())
    assert.Error(t, setSkills(defaultBranch()))
    assert.Empty(t, installedSkills(t, repo))
})

test("fetch failure returns error without writing files", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), "ralph-write-spec")
    fetchWillFail(t, "ralph-write-spec", defaultBranch())
    assert.Error(t, setSkills(defaultBranch()))
    assert.Empty(t, installedSkills(t, repo))
})

test("non-ralph skills excluded", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), "ralph-write-spec", "internal-tool")
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-spec"})
})

test("stale ralph skills removed", func(t *testing.T) {
    repo := aRepoWithSkill(t, "ralph-old-skill")
    skillsAvailable(t, defaultBranch(), "ralph-write-spec")
    require.NoError(t, setSkills(defaultBranch()))
    assert.NotContains(t, installedSkills(t, repo), "ralph-old-skill")
    assert.NotContains(t, claudeLinks(t, repo), "ralph-old-skill")
})

test("non-ralph skills preserved", func(t *testing.T) {
    repo := aRepoWithSkill(t, "my-custom-skill")
    skillsAvailable(t, defaultBranch(), "ralph-write-spec")
    require.NoError(t, setSkills(defaultBranch()))
    assert.Contains(t, installedSkills(t, repo), "my-custom-skill")
})

test("branch override applies to discovery and fetch", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, "v2", "ralph-write-spec")
    require.NoError(t, setSkills("v2"))
    assert.Contains(t, installedSkills(t, repo), "ralph-write-spec")
})
```

### Helpers

- **`aRepo(t)`** [`git`] — creates an isolated temporary git repository and sets it as the working directory for the test
- **`aRepoWithSkill(t, name)`** [`skills`] — creates a repo with an existing skill directory at `.agents/skills/<name>/` and a corresponding symlink at `.claude/skills/<name>/SKILL.md`
- **`noRepo(t)`** [`git`] — sets the working directory to a path outside any git repository
- **`defaultBranch()`** [`skills`] — returns `"main"`
- **`skillsAvailable(t, branch, names...)`** [`skills`] — configures the test environment so the GitHub Contents API returns the given skill names for `branch`
- **`discoveryWillFail(t, branch)`** [`skills`] — configures the test environment so the GitHub Contents API returns an error for `branch`
- **`fetchWillFail(t, skill, branch)`** [`skills`] — configures the test environment so fetching `SKILL.md` for `skill` on `branch` returns an error
- **`installedSkills(t, repo)`** [`skills`] — returns the sorted list of directory names under `.agents/skills/` in `repo`
- **`claudeLinks(t, repo)`** [`skills`] — returns the sorted list of directory names under `.claude/skills/` in `repo` that are symbolic links to `.agents/skills/`
