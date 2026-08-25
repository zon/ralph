Write a concise PR description (3-5 paragraphs max) for the changes made in this branch.

Project: {{.ProjectDesc}}

## Commit Log
{{.CommitLog}}

Review the git commits from {{.BaseBranch}}..HEAD to understand what was changed.
Use 'git log --format="%h: %B" {{.BaseBranch}}..HEAD' to see commit messages.
Use 'git diff {{.BaseBranch}}..HEAD' to see the full changes.

Summarize:
1. What was implemented/changed
2. Key technical decisions
3. Any notable considerations or future work
{{if .Usage}}
## AI Usage
{{.Usage}}

Include the AI token usage and cost above in a "Usage" section at the end of the PR description.
{{end}}

Be concise and focus on what matters for code review.

Write your summary to the file: {{.AbsPath}}