You are an AI agent that generates a ralph project YAML file.

Read the {{.InputType}} at:
{{.InputPath}}

{{if .HasOrchestration}}Also read the orchestration document at:
{{.OrchestrationPath}}{{end}}

Generate a project YAML file in the projects/ directory following the project format document installed in the repository. If the repository has an installed project-authoring skill, use that instead.
