# Set Skills Flow

## Purpose

Install ralph skills from the source repository into the target repository, replacing previously installed ralph skills while preserving unrelated skills.

## Flow

**Module:** `setup`

```go
func setSkills(branch string) error {
    root, err := git.Root()
    if err != nil {
        return err
    }

    names, err := skills.Discover(branch)
    if err != nil {
        return err
    }

    fetched, err := skills.Fetch(names, branch)
    if err != nil {
        return err
    }

    skills.RewriteLinks(fetched, branch)

    if err := skills.PruneRalph(root, names); err != nil {
        return err
    }

    return skills.Install(root, fetched)
}
```

### Helpers

- **`git.Root()`** — returns the root directory of the git repository containing the current working directory, or an error if not inside a git repository
- **`skills.Discover(branch)`** — queries the GitHub Contents API for `.claude/skills` on `branch` and returns the names of entries with a `ralph-` prefix
- **`skills.Fetch(names, branch)`** — fetches each named skill's `SKILL.md` from the raw GitHub URL on `branch`, returning a collection keyed by name
- **`skills.RewriteLinks(fetched, branch)`** — rewrites relative links to absolute raw GitHub URLs on `branch` and normalizes existing ralph raw URLs to use `branch`
- **`skills.PruneRalph(root, keep)`** — removes every `.claude/skills/ralph-*` directory under `root` whose name is not in `keep`
- **`skills.Install(root, fetched)`** — writes each skill's `SKILL.md` to `.claude/skills/<name>/SKILL.md` under `root`, overwriting any existing file

## Tests

**Module:** `setup`

```go
test("skills installed", func(t *testing.T) {
    repo := git.AnyRepo(t)
    skills.Available(t, skills.DefaultBranch(), skills.Any("ralph-write-spec"), skills.Any("ralph-write-flow"))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Installed(t, repo), []string{"ralph-write-flow", "ralph-write-spec"})
})

test("not in git repo", func(t *testing.T) {
    git.NoRepo(t)
    assert.Error(t, setSkills(skills.DefaultBranch()))
})

test("discovery failure leaves repo untouched", func(t *testing.T) {
    repo := skills.AnyRepoWith(t, "ralph-write-spec")
    skills.DiscoveryFails(t, skills.DefaultBranch())
    assert.Error(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Installed(t, repo), []string{"ralph-write-spec"})
})

test("fetch failure leaves repo untouched", func(t *testing.T) {
    repo := skills.AnyRepoWith(t, "ralph-write-spec")
    skills.Available(t, skills.DefaultBranch(), skills.Any("ralph-write-flow"))
    skills.FetchFails(t, "ralph-write-flow", skills.DefaultBranch())
    assert.Error(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Installed(t, repo), []string{"ralph-write-spec"})
})

test("non-ralph source skills ignored", func(t *testing.T) {
    repo := git.AnyRepo(t)
    skills.Available(t, skills.DefaultBranch(), skills.Any("ralph-write-spec"), skills.Any("internal-tool"))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Installed(t, repo), []string{"ralph-write-spec"})
})

test("stale ralph skills removed", func(t *testing.T) {
    repo := skills.AnyRepoWith(t, "ralph-old-skill")
    skills.Available(t, skills.DefaultBranch(), skills.Any("ralph-write-spec"))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Installed(t, repo), []string{"ralph-write-spec"})
})

test("non-ralph local skills preserved", func(t *testing.T) {
    repo := skills.AnyRepoWith(t, "my-custom-skill")
    skills.Available(t, skills.DefaultBranch(), skills.Any("ralph-write-spec"))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Contains(t, skills.Installed(t, repo), "my-custom-skill")
    assert.Contains(t, skills.Installed(t, repo), "ralph-write-spec")
})

test("existing ralph skill overwritten", func(t *testing.T) {
    repo := skills.AnyRepoWith(t, "ralph-write-spec")
    skills.Available(t, skills.DefaultBranch(), skills.WithBody("ralph-write-spec", "updated"))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Body(t, repo, "ralph-write-spec"), "updated")
})

test("relative link rewritten to branch raw url", func(t *testing.T) {
    repo := git.AnyRepo(t)
    skills.Available(t, skills.DefaultBranch(), skills.WithBody("ralph-write-spec", skills.RelativeLink("docs/formats/specs.md")))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Body(t, repo, "ralph-write-spec"), skills.RalphRawLink(skills.DefaultBranch(), "docs/formats/specs.md"))
})

test("existing ralph url branch normalized", func(t *testing.T) {
    repo := git.AnyRepo(t)
    skills.Available(t, "v2", skills.WithBody("ralph-write-spec", skills.RalphRawLink("main", "docs/formats/specs.md")))
    require.NoError(t, setSkills("v2"))
    assert.Equal(t, skills.Body(t, repo, "ralph-write-spec"), skills.RalphRawLink("v2", "docs/formats/specs.md"))
})

test("foreign absolute link preserved", func(t *testing.T) {
    repo := git.AnyRepo(t)
    skills.Available(t, skills.DefaultBranch(), skills.WithBody("ralph-write-spec", skills.ForeignLink()))
    require.NoError(t, setSkills(skills.DefaultBranch()))
    assert.Equal(t, skills.Body(t, repo, "ralph-write-spec"), skills.ForeignLink())
})

test("branch override applied", func(t *testing.T) {
    repo := git.AnyRepo(t)
    skills.Available(t, "v2", skills.Any("ralph-write-spec"))
    require.NoError(t, setSkills("v2"))
    assert.Contains(t, skills.Installed(t, repo), "ralph-write-spec")
})
```

### Helpers

- **`git.AnyRepo(t)`** — creates an isolated temporary git repository and makes it the working directory for the test, returning its root
- **`git.NoRepo(t)`** — sets the working directory to a path outside any git repository
- **`skills.AnyRepoWith(t, name)`** — creates a repo containing an existing skill installed at `.claude/skills/<name>/SKILL.md`, returning the repo root
- **`skills.DefaultBranch()`** — returns the branch ralph uses when none is specified
- **`skills.Any(name)`** — describes an available skill on the source branch with placeholder content
- **`skills.WithBody(name, body)`** — describes an available skill on the source branch with the given `SKILL.md` content
- **`skills.Available(t, branch, ...)`** — configures the test environment so the source repository exposes the given skills on `branch`
- **`skills.DiscoveryFails(t, branch)`** — configures the test environment so skill discovery on `branch` returns an error
- **`skills.FetchFails(t, name, branch)`** — configures the test environment so fetching `name` on `branch` returns an error
- **`skills.Installed(t, repo)`** — returns the sorted list of skill names installed under `.claude/skills/` in `repo`
- **`skills.Body(t, repo, name)`** — returns the contents of the installed `SKILL.md` for `name` in `repo`
- **`skills.RelativeLink(path)`** — returns a `SKILL.md` body containing a relative link to `path`
- **`skills.RalphRawLink(branch, path)`** — returns a `SKILL.md` body containing the canonical ralph raw GitHub URL for `path` on `branch`
- **`skills.ForeignLink()`** — returns a `SKILL.md` body containing an absolute URL pointing to a host other than the ralph raw content URL
