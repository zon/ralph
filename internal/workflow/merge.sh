#!/bin/sh
set -e

echo "Setting up GitHub App token..."
export GITHUB_TOKEN=$(ralph github-token --owner "$GITHUB_REPO_OWNER" --repo "$GITHUB_REPO_NAME")

echo "Configuring git for HTTPS authentication..."
git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

echo "Configuring git user..."
git config --global user.name "{{.BotName}}"
git config --global user.email "{{.BotEmail}}"

echo "Cloning repository: $GIT_REPO_URL"
git clone -b "$GIT_BRANCH" "$GIT_REPO_URL" /workspace/repo
cd /workspace/repo
ralph setup-workspace

echo "Checking out PR branch: $PR_BRANCH"
git fetch origin "$PR_BRANCH"
git checkout "$PR_BRANCH"

echo "Running ralph merge..."
ralph merge "$PR_BRANCH" --local --pr "$PR_NUMBER"

echo "Merge complete!"
