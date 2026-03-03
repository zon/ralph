#!/bin/sh
set -e

{{- if .DebugBranch}}
echo "Debug mode: cloning ralph branch '{{.DebugBranch}}' to /workspace/ralph..."
git clone -b "{{.DebugBranch}}" https://github.com/zon/ralph.git /workspace/ralph
RALPH_CMD="go run /workspace/ralph/cmd/ralph/main.go"
{{- else}}
RALPH_CMD="ralph"
{{- end}}

echo "Setting up GitHub App token and configuring git authentication..."
$RALPH_CMD set-github-token --owner "$GITHUB_REPO_OWNER" --repo "$GITHUB_REPO_NAME"

echo "Setting up OpenCode credentials..."
mkdir -p ~/.local/share/opencode
cp /secrets/opencode/auth.json ~/.local/share/opencode/auth.json

echo "Configuring git user..."
git config --global user.name "{{.BotName}}"
git config --global user.email "{{.BotEmail}}"

echo "Cloning repository: $GIT_REPO_URL"
git clone -b "$GIT_BRANCH" "$GIT_REPO_URL" /workspace/repo
cd /workspace/repo
$RALPH_CMD setup-workspace

echo "Fetching base branch: $BASE_BRANCH"
git fetch origin "$BASE_BRANCH":"$BASE_BRANCH" 2>/dev/null || git fetch origin "$BASE_BRANCH" 2>/dev/null || true

if [ "$PROJECT_BRANCH" != "$GIT_BRANCH" ]; then
  echo "Fetching remote branches..."
  git fetch origin
  if git ls-remote --exit-code --heads origin "$PROJECT_BRANCH" > /dev/null 2>&1; then
    echo "Checking out existing remote branch: $PROJECT_BRANCH"
    git checkout "$PROJECT_BRANCH"
  else
    echo "Creating and checking out new branch: $PROJECT_BRANCH"
    git checkout -b "$PROJECT_BRANCH"
  fi
fi

echo "Running ralph..."
$RALPH_CMD "$PROJECT_PATH" --local{{.DryRunFlag}}{{.VerboseFlag}} --no-notify

opencode stats

echo "Execution complete!"
