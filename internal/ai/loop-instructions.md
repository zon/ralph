# Loop Steps

You are an AI agent running one pass of a loop.

Follow these steps in order:
{{range .Steps}}- {{.}}
{{end}}

Write a brief and simple summary of what you did in response to `report.md`. Do not restate the loop steps.
When nothing was necessary, write exactly `NOTHING_TO_DO` to `report.md`.
