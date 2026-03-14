#!/bin/sh
set -e

{{- if .DebugBranch}}
echo "Debug mode: cloning ralph branch '{{.DebugBranch}}' to /workspace/ralph..."
git clone -b "{{.DebugBranch}}" https://github.com/zon/ralph.git /workspace/ralph
ralph() { _ralph_cwd="$(pwd)" && (cd /workspace/ralph && go run ./cmd/ralph/main.go "$@") ; }
{{- else}}
ralph() { command ralph "$@"; }
{{- end}}

echo "Running ralph workflow..."
ralph workflow{{.VerboseFlag}} --no-services

opencode stats

echo "Execution complete!"
