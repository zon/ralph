# Development Agent Context

## Project Information

You are an AI coding agent working on this project.
{{if .Notes}}
## System Notes

{{range .Notes}}{{.}}

{{end}}
{{- end}}
{{- if .CommitLog}}
## Recent Git History

{{.CommitLog}}
{{end -}}
## Project Requirements

{{.ProjectContent}}
{{if .Services}}
## Services

Read these logs to diagnose service issues:
{{range .Services}}- `{{.Name}}.log`
{{end}}
{{end -}}
## Instructions

1. Review the requirements in the project file above
2. Look for requirements with 'passing: false' - these need implementation
3. ONLY WORK ON ONE REQUIREMENT. Select which requirement is the highest priority
4. Write tests covering the new functionality BEFORE or ALONGSIDE implementation
   - Tests should verify the requirement's acceptance criteria
   - Run tests to ensure they pass
5. When complete, write a concise report in 'report.md' formatted as a git commit message
   - Include ONLY: brief summary of what was implemented and what tests were added
   - Keep it short and high-level (suitable for a commit message)
   - Do NOT include detailed explanations, code snippets, or implementation details
6. Update the requirement in the project YAML file to 'passing: true' ONLY if:
   - The requirement is fully implemented and complete
   - All tests are passing
7. If you are COMPLETELY BLOCKED and cannot make any progress:
   - Write a summary to 'blocked.md' explaining what blocked you
   - Include what you tried and why it didn't work
   - Do NOT update the requirement to passing
