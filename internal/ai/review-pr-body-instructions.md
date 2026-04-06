Write a concise PR description (2-4 paragraphs max) for this code review.

Review Name: {{.ProjectName}}
{{if .ProjectDescription}}
Description: {{.ProjectDescription}}
{{end}}

## Findings Summary
{{range .Requirements}}
{{.}}
{{end}}

Review the requirements above and write a summary that:
1. Lists the key findings from the review
2. Highlights any critical issues that need attention
3. Notes what's working well

Write your summary to the file: {{.AbsPath}}