---
name: ralph-write-skill
description: Writes an agent skill to /docs/skills and makes it available to OpenCode and Claude
---
1. Read the [OpenCode Skills docs](https://opencode.ai/docs/skills/) describing how to write skills
2. Read the spec file in /specs/<component>/feature/spec.md describing what the skill should do
3. Write a skill that meets the spec to /docs/skills/<name>.md
4. Create symlinks for both OpenCode and Claude if none exist:
  * .agents/skills/<name>/SKILL.md
  * .claude/skills/<name>/SKILL.md