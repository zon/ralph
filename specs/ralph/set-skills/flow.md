# Set Skills Flow

## Purpose

Discover ralph-prefixed skills from the ralph GitHub repository, fetch each skill's `SKILL.md`, rewrite links, and sync them into the target repository's `.claude/skills/` directory.

## Flow

```go
func setSkills(repoRoot, branch string) error {
    available, err := discoverSkills(branch)
    if err != nil {
        return err
    }

    for _, skill := range available {
        content, err := fetchSkill(skill, branch)
        if err != nil {
            return err
        }
        if err := writeSkill(repoRoot, skill, rewriteLinks(content, branch)); err != nil {
            return err
        }
    }

    return removeStaleSkills(repoRoot, available)
}

func discoverSkills(branch string) ([]string, error) {
    entries, err := githubContents("zon/ralph", ".claude/skills", branch)
    if err != nil {
        return nil, err
    }
    var skills []string
    for _, entry := range entries {
        if entry.Type == "dir" && strings.HasPrefix(entry.Name, "ralph-") {
            skills = append(skills, entry.Name)
        }
    }
    return skills, nil
}

func fetchSkill(skill, branch string) (string, error) {
    url := fmt.Sprintf(
        "https://raw.githubusercontent.com/zon/ralph/refs/heads/%s/.claude/skills/%s/SKILL.md",
        branch, skill,
    )
    return httpGet(url)
}

func rewriteLinks(content, branch string) string {
    content = expandRelativeLinks(content, branch)
    content = updateRalphURLBranch(content, branch)
    return content
}

func writeSkill(repoRoot, skill, content string) error {
    dir := filepath.Join(repoRoot, ".claude", "skills", skill)
    os.MkdirAll(dir, 0755)
    return os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0644)
}

func removeStaleSkills(repoRoot string, available []string) error {
    skillsDir := filepath.Join(repoRoot, ".claude", "skills")
    entries, _ := os.ReadDir(skillsDir)
    for _, entry := range entries {
        if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "ralph-") {
            continue
        }
        if !slices.Contains(available, entry.Name()) {
            os.RemoveAll(filepath.Join(skillsDir, entry.Name()))
        }
    }
    return nil
}
```

## Tests

```go
test("skills installed into empty repo", func() {
    root := targetRepo().empty()
    sourceSkills("ralph-write-spec", "ralph-write-flow", "internal-tool")
    setSkills(root, "main")
    expectSkillInstalled(root, "ralph-write-spec")
    expectSkillInstalled(root, "ralph-write-flow")
    expectSkillAbsent(root, "internal-tool")
})

test("existing skills overwritten", func() {
    root := targetRepo().withSkills("ralph-write-spec")
    sourceSkills("ralph-write-spec")
    setSkills(root, "main")
    expectSkillInstalled(root, "ralph-write-spec")
})

test("removed skills deleted", func() {
    root := targetRepo().withSkills("ralph-old-skill")
    sourceSkills("ralph-write-spec")
    setSkills(root, "main")
    expectSkillAbsent(root, "ralph-old-skill")
    expectSkillInstalled(root, "ralph-write-spec")
})

test("non-ralph skills untouched", func() {
    root := targetRepo().withSkills("my-custom-skill")
    sourceSkills("ralph-write-spec")
    setSkills(root, "main")
    expectSkillInstalled(root, "my-custom-skill")
})

test("branch override", func() {
    root := targetRepo().empty()
    sourceSkillsOnBranch("ralph-write-spec", "v2")
    setSkills(root, "v2")
    expectSkillInstalled(root, "ralph-write-spec")
})

test("discovery failure returns error", func() {
    root := targetRepo().empty()
    githubContentsWillFail()
    err := setSkills(root, "main")
    expect(err).notNil()
    expectNoFilesWritten(root)
})

test("fetch failure returns error", func() {
    root := targetRepo().empty()
    sourceSkills("ralph-write-spec")
    skillFetchWillFail("ralph-write-spec")
    err := setSkills(root, "main")
    expect(err).notNil()
    expectNoFilesWritten(root)
})

test("relative link expanded to absolute", func() {
    content := rewriteLinks("see [docs](docs/planning/specs.md)", "main")
    expect(content).toContain("https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/planning/specs.md")
})

test("existing ralph URL branch updated", func() {
    content := rewriteLinks(
        "https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/planning/specs.md",
        "v2",
    )
    expect(content).toContain("refs/heads/v2/docs/planning/specs.md")
    expect(content).not().toContain("refs/heads/main/")
})

test("non-ralph absolute URL unchanged", func() {
    url := "https://example.com/some/doc.md"
    content := rewriteLinks(url, "main")
    expect(content).toBe(url)
})
```
