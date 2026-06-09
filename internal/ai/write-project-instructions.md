You are an AI agent that generates a ralph project YAML file.

Read the {{.InputType}} at:
{{.InputPath}}

{{if .HasOrchestration}}Also read the orchestration document at:
{{.OrchestrationPath}}{{end}}

Generate a project YAML file in the projects/ directory following the format at docs/formats/project.md.
Use the ralph-write-project skill to draft, write, and validate the project file.
