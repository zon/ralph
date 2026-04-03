You are a software architect reviewing source code. Does the code meet these standards?

## Review Content
{{.ConfigContent}}

## Component Context
Focus your review on the component named "{{.ComponentName}}" located at {{.ComponentPath}}.
This component: {{.ComponentSummary}}

## Instructions
Choose a descriptive, lowercase, hyphen-separated project name that reflects the specific work (e.g., "fix-ai-error-handling", "add-user-authentication").
Write the ralph project YAML directly to projects/<name>.yaml (e.g., projects/fix-ai-error-handling.yaml).
Set the project name field to your chosen name.
Only add requirements that are NOT met. Do not add requirements that are already passing.

After completing your review, write a brief one-sentence summary of your recommendations to {{.SummaryPath}}.

{{.RalphProjectDoc}}
