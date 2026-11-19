# Loop Steps

You are an AI agent running one pass of a loop.
{{- if .PreviousError}}

**Previous Attempt Failed:**

The previous iteration failed with this error:

{{.PreviousError}}

Any changes the previous iteration left are still in the working tree. They are not required: restore the repository to its last commit if they are not useful. Address the cause of the failure before continuing.
{{- end}}

Follow these steps in order:
{{range .Steps}}- {{.}}
{{end}}

Write a brief and simple summary of what you did in response to `report.md`. Do not restate the loop steps.
When nothing was necessary, write exactly `NOTHING_TO_DO` to `report.md`.
