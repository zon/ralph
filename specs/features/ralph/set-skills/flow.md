# Set Skills Flow

## Purpose

Install ralph skills from the source repository into the target repository, replacing previously installed ralph skills while preserving unrelated skills.

## Flow

**Module:** `setup`

```go
func setSkills(branch string) error {
    root, err := repoRoot()
    if err != nil {
        return err
    }

    names, err := discoverSkills(branch)
    if err != nil {
        return err
    }

    skills, err := fetchSkills(names, branch)
    if err != nil {
        return err
    }

    rewriteLinks(skills, branch)

    if err := pruneRalphSkills(root, names); err != nil {
        return err
    }

    return installSkills(root, skills)
}
```

### Helpers

- **`repoRoot()`** [`git`] — returns the root directory of the git repository containing the current working directory, or an error if not inside a git repository
- **`discoverSkills(branch)`** [`skills`] — queries the GitHub Contents API for `.claude/skills` on `branch` and returns the names of entries with a `ralph-` prefix
- **`fetchSkills(names, branch)`** [`skills`] — fetches each named skill's `SKILL.md` from the raw GitHub URL on `branch`, returning a collection of skills keyed by name
- **`rewriteLinks(skills, branch)`** [`skills`] — rewrites relative links to absolute raw GitHub URLs on `branch` and normalizes existing ralph raw URLs to use `branch`
- **`pruneRalphSkills(root, keep)`** [`skills`] — removes every `.claude/skills/ralph-*` directory under `root` whose name is not in `keep`
- **`installSkills(root, skills)`** [`skills`] — writes each skill's `SKILL.md` to `.claude/skills/<name>/SKILL.md` under `root`, overwriting any existing file

## Tests

**Module:** `setup.test`

```go
test("skills installed", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), aSkill("ralph-write-spec"), aSkill("ralph-write-flow"))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-flow", "ralph-write-spec"})
})

test("not in git repo", func(t *testing.T) {
    notInARepo(t)
    assert.Error(t, setSkills(defaultBranch()))
})

test("discovery failure leaves repo untouched", func(t *testing.T) {
    repo := aRepoWithSkill(t, "ralph-write-spec")
    discoveryFails(t, defaultBranch())
    assert.Error(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-spec"})
})

test("fetch failure leaves repo untouched", func(t *testing.T) {
    repo := aRepoWithSkill(t, "ralph-write-spec")
    skillsAvailable(t, defaultBranch(), aSkill("ralph-write-flow"))
    fetchFails(t, "ralph-write-flow", defaultBranch())
    assert.Error(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-spec"})
})

test("non-ralph source skills ignored", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), aSkill("ralph-write-spec"), aSkill("internal-tool"))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-spec"})
})

test("stale ralph skills removed", func(t *testing.T) {
    repo := aRepoWithSkill(t, "ralph-old-skill")
    skillsAvailable(t, defaultBranch(), aSkill("ralph-write-spec"))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, installedSkills(t, repo), []string{"ralph-write-spec"})
})

test("non-ralph local skills preserved", func(t *testing.T) {
    repo := aRepoWithSkill(t, "my-custom-skill")
    skillsAvailable(t, defaultBranch(), aSkill("ralph-write-spec"))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Contains(t, installedSkills(t, repo), "my-custom-skill")
    assert.Contains(t, installedSkills(t, repo), "ralph-write-spec")
})

test("existing ralph skill overwritten", func(t *testing.T) {
    repo := aRepoWithSkill(t, "ralph-write-spec")
    skillsAvailable(t, defaultBranch(), aSkillWithContent("ralph-write-spec", "updated"))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, skillContent(t, repo, "ralph-write-spec"), "updated")
})

test("relative link rewritten to branch raw url", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), aSkillWithContent("ralph-write-spec", relativeLink("docs/formats/specs.md")))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, skillContent(t, repo, "ralph-write-spec"), ralphRawLink(defaultBranch(), "docs/formats/specs.md"))
})

test("existing ralph url branch normalized", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, "v2", aSkillWithContent("ralph-write-spec", ralphRawLink("main", "docs/formats/specs.md")))
    require.NoError(t, setSkills("v2"))
    assert.Equal(t, skillContent(t, repo, "ralph-write-spec"), ralphRawLink("v2", "docs/formats/specs.md"))
})

test("foreign absolute link preserved", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, defaultBranch(), aSkillWithContent("ralph-write-spec", foreignLink()))
    require.NoError(t, setSkills(defaultBranch()))
    assert.Equal(t, skillContent(t, repo, "ralph-write-spec"), foreignLink())
})

test("branch override applied", func(t *testing.T) {
    repo := aRepo(t)
    skillsAvailable(t, "v2", aSkill("ralph-write-spec"))
    require.NoError(t, setSkills("v2"))
    assert.Contains(t, installedSkills(t, repo), "ralph-write-spec")
})
```

### Helpers

- **`aRepo(t)`** [`git.fixtures`] — creates an isolated temporary git repository and makes it the working directory for the test, returning its root
- **`aRepoWithSkill(t, name)`** [`skills.fixtures`] — creates a repo containing an existing skill installed at `.claude/skills/<name>/SKILL.md`, returning the repo root
- **`notInARepo(t)`** [`git.fixtures`] — sets the working directory to a path outside any git repository
- **`defaultBranch()`** [`skills.fixtures`] — returns the branch ralph uses when none is specified
- **`aSkill(name)`** [`skills.fixtures`] — describes an available skill on the source branch with placeholder content
- **`aSkillWithContent(name, body)`** [`skills.fixtures`] — describes an available skill on the source branch with the given `SKILL.md` content
- **`skillsAvailable(t, branch, skills...)`** [`skills.fixtures`] — configures the test environment so the source repository exposes the given skills on `branch`
- **`discoveryFails(t, branch)`** [`skills.fixtures`] — configures the test environment so skill discovery on `branch` returns an error
- **`fetchFails(t, name, branch)`** [`skills.fixtures`] — configures the test environment so fetching `name` on `branch` returns an error
- **`installedSkills(t, repo)`** [`skills.fixtures`] — returns the sorted list of skill names installed under `.claude/skills/` in `repo`
- **`skillContent(t, repo, name)`** [`skills.fixtures`] — returns the contents of the installed `SKILL.md` for `name` in `repo`
- **`relativeLink(path)`** [`skills.fixtures`] — returns a `SKILL.md` body containing a relative link to `path`
- **`ralphRawLink(branch, path)`** [`skills.fixtures`] — returns a `SKILL.md` body containing the canonical ralph raw GitHub URL for `path` on `branch`
- **`foreignLink()`** [`skills.fixtures`] — returns a `SKILL.md` body containing an absolute URL pointing to a host other than the ralph raw content URL
