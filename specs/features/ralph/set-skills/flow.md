# Set Skills Flow

## Purpose

Fetch and install ralph skills from the ralph GitHub repository into the target repository, removing stale ralph skills and leaving non-ralph skills untouched.

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

- **`git.RepoRoot()`** — returns the root of the git repository containing the current working directory; fails if not inside any repo
- **`skills.Discover(branch)`** — queries the GitHub Contents API and returns the names of all ralph-prefixed skill directories on the given branch; fails if the API is unreachable or returns an error
- **`skills.FetchAll(branch, names)`** — fetches each skill's `SKILL.md` from the ralph raw content URL, rewrites relative and stale-branched links to the resolved branch, and returns the results; fails if any fetch fails
- **`skills.PruneStale(root, fetched)`** — removes ralph-prefixed skill directories from the target repo that are absent from the fetched set; leaves non-ralph skills untouched
- **`skills.InstallAll(root, fetched)`** — writes each fetched skill's `SKILL.md` into `.claude/skills/<name>/`, creating directories as needed

## Tests

**Module:** `internal/setup`

```go
func TestSetSkills_InstallsRalphSkillsOnly(t *testing.T) {
    repo := target.repo(t)
    branch := source.branch(t).
        withRalphSkill("ralph-write-spec", "content").
        withNonRalphSkill("internal-tool", "content").
        start()

    err := SetSkills(branch.name())

    require.NoError(t, err)
    repo.hasSkill("ralph-write-spec")
    repo.doesNotHaveSkill("internal-tool")
}

func TestSetSkills_OverwritesExistingSkill(t *testing.T) {
    repo := target.repo(t).withInstalledSkill("ralph-write-spec", "old")
    branch := source.branch(t).withRalphSkill("ralph-write-spec", "new").start()

    err := SetSkills(branch.name())

    require.NoError(t, err)
    repo.skillContains("ralph-write-spec", "new")
}

func TestSetSkills_RemovesStaleRalphSkill(t *testing.T) {
    repo := target.repo(t).withInstalledSkill("ralph-old-skill", "stale")
    branch := source.branch(t).withRalphSkill("ralph-write-spec", "content").start()

    err := SetSkills(branch.name())

    require.NoError(t, err)
    repo.doesNotHaveSkill("ralph-old-skill")
}

func TestSetSkills_LeavesNonRalphSkillsUntouched(t *testing.T) {
    repo := target.repo(t).withInstalledSkill("my-custom-skill", "mine")
    branch := source.branch(t).withRalphSkill("ralph-write-spec", "content").start()

    err := SetSkills(branch.name())

    require.NoError(t, err)
    repo.skillContains("my-custom-skill", "mine")
}

func TestSetSkills_RewritesLinksToResolvedBranch(t *testing.T) {
    repo := target.repo(t)
    branch := source.branch(t).onBranch("v2").
        withRalphSkill("ralph-write-spec", "[spec](docs/formats/specs.md)").
        start()

    err := SetSkills("v2")

    require.NoError(t, err)
    repo.skillContains("ralph-write-spec", source.rawURL("v2", "docs/formats/specs.md"))
}

func TestSetSkills_DiscoveryFailureWritesNothing(t *testing.T) {
    repo := target.repo(t)
    source.branch(t).failsDiscovery().start()

    err := SetSkills(source.defaultBranch())

    require.Error(t, err)
    repo.hasNoSkills()
}

func TestSetSkills_FetchFailureWritesNothing(t *testing.T) {
    repo := target.repo(t)
    source.branch(t).
        withRalphSkill("ralph-write-spec", "content").
        failsFetchFor("ralph-write-spec").
        start()

    err := SetSkills(source.defaultBranch())

    require.Error(t, err)
    repo.hasNoSkills()
}

func TestSetSkills_OutsideGitRepoErrors(t *testing.T) {
    target.outsideRepo(t)

    err := SetSkills(source.defaultBranch())

    require.Error(t, err)
}
```

### Helpers

- **`target.repo(t)`** — creates a temporary git repository, sets it as the current working directory, and returns a handle for setup and assertions
- **`target.outsideRepo(t)`** — sets the current working directory to a path that is not inside any git repository
- **`repo.withInstalledSkill(name, body)`** — pre-installs a skill with the given name and body in the target repository's `.claude/skills/` directory; returns the repo handle for chaining
- **`repo.hasSkill(name)`** — asserts that the named skill directory and `SKILL.md` exist in the target repository
- **`repo.doesNotHaveSkill(name)`** — asserts that the named skill is absent from the target repository
- **`repo.skillContains(name, substr)`** — asserts that the named skill's `SKILL.md` contains the given substring
- **`repo.hasNoSkills()`** — asserts that no skills are installed in the target repository
- **`source.branch(t)`** — returns a builder for a mock skill source that simulates the GitHub Contents API and raw content server; registers server shutdown via `t.Cleanup`
- **`source.defaultBranch()`** — returns the default branch name (`main`)
- **`source.rawURL(branch, path)`** — returns the expected raw GitHub URL for a file path on the given branch
- **`builder.onBranch(name)`** — configures the source to serve skills from the named branch
- **`builder.withRalphSkill(name, body)`** — registers a ralph-prefixed skill on the source
- **`builder.withNonRalphSkill(name, body)`** — registers a non-ralph-prefixed skill on the source
- **`builder.failsDiscovery()`** — configures the source to return an error for the Contents API discovery request
- **`builder.failsFetchFor(name)`** — configures the source to return an error when fetching the named skill's `SKILL.md`
- **`builder.start()`** — starts the mock HTTP server, wires it as the HTTP backend for the skills package, and returns the started source handle
