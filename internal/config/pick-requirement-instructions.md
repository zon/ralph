# Requirement Picker Agent Context

## Project Information

You are an AI coding agent working on this project.

## Project Requirements

{{.ProjectContent}}
{{- if .Notes}}
## System Notes

{{range .Notes}}{{.}}

{{end}}
{{- end}}
{{- if .CommitLog}}
## Recent Git History

{{.CommitLog}}
{{end -}}
## Instructions

1. Review the requirements in the project file above
2. Review the recent git history to understand the context
3. Look for requirements with 'passing: false' - these need implementation
4. Select the highest-priority failing requirement based on:
   - Dependencies on other requirements
   - Logical ordering of features
   - Impact on the overall project
5. Write the selected requirement's YAML to a file named `picked-requirement.yaml`
   - Include the full requirement content (category, description, items, etc.)
6. Do NOT make any code changes - only write the requirement YAML file
