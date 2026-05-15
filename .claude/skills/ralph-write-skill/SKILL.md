---
name: ralph-write-skill
description: Writes an agent skill to .claude/skills and makes it available to OpenCode and Claude
---
1. Read the [OpenCode Skills docs](https://opencode.ai/docs/skills/) describing how to write skills
2. Read the spec file in /specs/<component>/feature/spec.md describing what the skill should do
3. Write a skill that meets the spec to `.claude/skills/<name>/SKILL.md`
   * Skills should reference existing documentation rather than repeat its contents — link to the docs and let the agent read them when needed
