# Service Startup Failed

{{.Error}}

## Service Details

- **Name:** {{.ServiceName}}
- **Start command:** `{{.ServiceCmd}}`
{{- if .ServicePort}}
- **Health check:** port {{.ServicePort}} must be accepting connections
{{- end}}

## Instructions

1. Diagnose why the service failed to start using the error above
2. Fix the issue so the service starts successfully and passes its health check
3. When complete, write a concise report in 'report.md' formatted as a git commit message
   - Include ONLY: brief summary of what was fixed
   - Keep it short and high-level (suitable for a commit message)
   - Do NOT include detailed explanations, code snippets, or implementation details
