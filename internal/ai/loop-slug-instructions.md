# Loop Branch Slug

You are an AI agent preparing a git branch for a loop run.

Read the steps below. The loop will run them on the branch.
Propose a short, descriptive slug for the branch.

## Steps
{{range .Steps}}- {{.}}
{{end}}

The slug must be a single token: lowercase letters, digits, and hyphens only
(e.g. "fmt-vet"). No leading, trailing, or consecutive hyphens.
Write nothing else.

Write the slug to the file: {{.OutputFile}}
