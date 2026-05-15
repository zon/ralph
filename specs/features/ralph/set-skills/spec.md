# Set Skills Specification

## Purpose

Fetch Claude Code skills from the ralph GitHub repository's main branch and install them into the repository from which ralph was invoked, making ralph's skills available in that project.

## Requirements

### Requirement: Skill Discovery

The system SHALL query `https://api.github.com/repos/zon/ralph/contents/.claude/skills?ref=<branch>` to discover available skills, and SHALL only install skills whose directory name has a `ralph-` prefix.

#### Scenario: Only ralph-prefixed skills installed

- GIVEN the source branch contains `.claude/skills/ralph-write-spec/` and `.claude/skills/internal-tool/`
- WHEN the user runs `ralph set skills`
- THEN only `ralph-write-spec` is installed into the target repository
- AND `internal-tool` is ignored

#### Scenario: Discovery failure

- GIVEN the GitHub Contents API URL is unreachable or returns an error
- WHEN the user runs `ralph set skills`
- THEN an error is returned and no files are written

### Requirement: Skill Fetching

The system SHALL fetch each discovered skill's `SKILL.md` from `https://raw.githubusercontent.com/zon/ralph/refs/heads/<branch>/.claude/skills/<skill>/SKILL.md`, where `<branch>` defaults to `main` and may be overridden with `--branch`.

#### Scenario: Skills installed successfully

- GIVEN a target repository with no existing `.claude/skills/` directory
- WHEN the user runs `ralph set skills`
- THEN `.claude/skills/` is created in the target repository
- AND each discovered `ralph-` prefixed skill's `SKILL.md` is written to `.claude/skills/<skill>/SKILL.md`

#### Scenario: Existing skills overwritten

- GIVEN a target repository that already has one or more `ralph-` prefixed skills in `.claude/skills/`
- WHEN the user runs `ralph set skills`
- THEN all discovered skills are fetched and written, overwriting any with the same name

#### Scenario: Removed skills deleted

- GIVEN a target repository contains `.claude/skills/ralph-old-skill/` that is no longer present on the source branch
- WHEN the user runs `ralph set skills`
- THEN `.claude/skills/ralph-old-skill/` is removed from the target repository

#### Scenario: Non-ralph skills untouched

- GIVEN a target repository contains `.claude/skills/my-custom-skill/` without a `ralph-` prefix
- WHEN the user runs `ralph set skills`
- THEN `.claude/skills/my-custom-skill/` is left unchanged

#### Scenario: Branch override

- GIVEN the user passes `--branch v2`
- WHEN the user runs `ralph set skills --branch v2`
- THEN skills are discovered and fetched from `refs/heads/v2` instead of `refs/heads/main`

#### Scenario: Fetch failure

- GIVEN a skill's raw content URL is unreachable or returns an error
- WHEN the user runs `ralph set skills`
- THEN an error is returned and no files are written

### Requirement: Link Rewriting

The system SHALL rewrite file links in a skill's `SKILL.md` before writing it to the target repository, so all references resolve correctly regardless of where the skill is installed.

Relative links SHALL be expanded to absolute `https://raw.githubusercontent.com/zon/ralph/refs/heads/<branch>/` URLs using the resolved branch.

Absolute links already pointing to the ralph raw content URL SHALL have their branch segment replaced with the resolved branch.

#### Scenario: Relative link rewritten

- GIVEN a `SKILL.md` contains a relative link such as `docs/formats/specs.md`
- AND the resolved branch is `main`
- WHEN the skill is written to the target repository
- THEN the link is rewritten to `https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/specs.md`

#### Scenario: Existing ralph URL branch updated

- GIVEN a `SKILL.md` contains `https://raw.githubusercontent.com/zon/ralph/refs/heads/main/docs/formats/specs.md`
- AND the user passes `--branch v2`
- WHEN the skill is written to the target repository
- THEN the link is rewritten to `https://raw.githubusercontent.com/zon/ralph/refs/heads/v2/docs/formats/specs.md`

#### Scenario: Non-ralph absolute URLs unchanged

- GIVEN a `SKILL.md` contains an absolute URL pointing to a host other than the ralph repository
- WHEN the skill is written to the target repository
- THEN the link is written as-is

### Requirement: Target Repository Detection

The system SHALL install skills into the repository containing the current working directory.

#### Scenario: Current directory is inside a git repo

- GIVEN the current working directory is inside a git repository
- WHEN the user runs `ralph set skills`
- THEN skills are installed at the root of that git repository

#### Scenario: Current directory is not inside a git repo

- GIVEN the current working directory is not inside any git repository
- WHEN the user runs `ralph set skills`
- THEN an error is returned and no files are written
