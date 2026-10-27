# Prompt Branch Slug

You are an AI agent naming a git branch for a direct prompt run.

Read the description below. Ralph names the branch from it.
Propose a short, descriptive slug for the branch.

## Description
{{.Description}}

The slug must be a single token: lowercase letters, digits, and hyphens only
(e.g. "update-readme"). No leading, trailing, or consecutive hyphens.
Write nothing else.

Write the slug to the file: {{.OutputFile}}
