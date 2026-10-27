# Prompt Branch Slug

You are an AI agent preparing a git branch for a direct prompt run.

Read the prompt below. Ralph will run it on the branch.
Propose a short, descriptive slug for the branch.

## Prompt
{{.Prompt}}

The slug must be a single token: lowercase letters, digits, and hyphens only
(e.g. "update-readme"). No leading, trailing, or consecutive hyphens.
Write nothing else.

Write the slug to the file: {{.OutputFile}}
