#!/bin/sh
set -e

echo "Setting up GitHub App token..."
export GITHUB_TOKEN=$(ralph github-token --owner "$GITHUB_REPO_OWNER" --repo "$GITHUB_REPO_NAME")

echo "Configuring git for HTTPS authentication..."
git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

echo "Setting up OpenCode credentials..."
mkdir -p ~/.local/share/opencode
cp /secrets/opencode/auth.json ~/.local/share/opencode/auth.json

echo "Configuring git user..."
git config --global user.name "{{.BotName}}"
git config --global user.email "{{.BotEmail}}"

echo "Cloning repository: $GIT_REPO_URL"
git clone -b "$GIT_BRANCH" "$GIT_REPO_URL" /workspace/repo
cd /workspace/repo

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

echo "Running ralph comment..."
ralph comment "$COMMENT_BODY" --repo "$GITHUB_REPO_OWNER/$GITHUB_REPO_NAME" --branch "$PROJECT_BRANCH" --pr "$PR_NUMBER"{{.DryRunFlag}}{{.VerboseFlag}} --no-notify

echo "Execution complete!"
