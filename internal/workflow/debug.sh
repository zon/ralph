#!/bin/sh
set -e

{{- if .DebugBranch}}
echo "Debug mode: cloning ralph branch '{{.DebugBranch}}' to /workspace/ralph..."
git clone -b "{{.DebugBranch}}" https://github.com/zon/ralph.git /workspace/ralph

# Create a wrapper script for ralph to use the cloned source
cat > /usr/local/bin/ralph <<EOF
#!/bin/sh
cd /workspace/ralph && go run ./cmd/ralph/main.go "\$@"
EOF
chmod +x /usr/local/bin/ralph
{{- end}}

echo "Running ralph workflow..."
ralph workflow{{.VerboseFlag}}{{.NoServicesFlag}}{{.ModelFlag}}

echo "Execution complete!"
