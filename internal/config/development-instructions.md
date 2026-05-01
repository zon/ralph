# Development Agent Context

## Project Information

You are an AI coding agent working on this project.

## Selected Requirement

{{.SelectedRequirement}}

The full project file is available at: `{{.ProjectFilePath}}`.
{{if .Notes}}
## System Notes

{{range .Notes}}{{.}}

{{end}}
{{- end}}
{{- if .CommitLog}}
## Recent Git History

{{.CommitLog}}
{{end -}}
{{if .Services}}
## Services

Read these logs to diagnose service issues:
{{range .Services}}- `{{.Name}}.log`
{{end}}
{{end -}}
## Instructions

1. Implement the selected requirement above
2. Organize the implementation as a collection of concern-separated deep modules — each module handles one concern end-to-end through a simple interface, hiding internal complexity
3. Write tests covering the new functionality BEFORE or ALONGSIDE implementation
   - Tests should verify the requirement's acceptance criteria
   - Run tests to ensure they pass
4. When complete, write a concise report in 'report.md' formatted as a git commit message
   - Include ONLY: brief summary of what was implemented and what tests were added
   - Keep it short and high-level (suitable for a commit message)
   - Do NOT include detailed explanations, code snippets, or implementation details
5. Update the requirement in the project YAML file to 'passing: true' ONLY if:
   - The requirement is fully implemented and complete
   - All tests are passing
6. If you are COMPLETELY BLOCKED and cannot make any progress:
   - Write a summary to 'blocked.md' explaining what blocked you
   - Include what you tried and why it didn't work
   - Do NOT update the requirement to passing
