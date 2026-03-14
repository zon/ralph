#!/bin/sh
set -e

{{- if .DebugBranch}}
echo "Debug mode: cloning ralph branch '{{.DebugBranch}}' to /workspace/ralph..."
git clone -b "{{.DebugBranch}}" https://github.com/zon/ralph.git /workspace/ralph
ralph() { _ralph_cwd="$(pwd)" && (cd /workspace/ralph && go run ./cmd/ralph/main.go "$@") ; }
ralph_run() { _ralph_cwd="$(pwd)" && (cd /workspace/ralph && go run ./cmd/ralph/main.go -C "$_ralph_cwd" "$@") ; }
{{- else}}
ralph() { command ralph "$@"; }
ralph_run() { ralph "$@"; }
{{- end}}

echo "Setting up GitHub App token and configuring git authentication..."
ralph set-github-token --owner "$GITHUB_REPO_OWNER" --repo "$GITHUB_REPO_NAME"

echo "Setting up OpenCode credentials..."
mkdir -p ~/.local/share/opencode
cp /secrets/opencode/auth.json ~/.local/share/opencode/auth.json

echo "Configuring git user..."
git config --global user.name "{{.BotName}}"
git config --global user.email "{{.BotEmail}}"

echo "Cloning repository: $GIT_REPO_URL"
git clone -b "$GIT_BRANCH" "$GIT_REPO_URL" /workspace/repo
cd /workspace/repo
ralph setup-workspace

echo "Determining base branch dynamically..."
if [ "$BASE_BRANCH_OVERRIDE" = "true" ]; then
  echo "Using explicit --base flag: $BASE_BRANCH"
elif [ "$GIT_BRANCH" != "$PROJECT_BRANCH" ]; then
  BASE_BRANCH="$GIT_BRANCH"
  echo "Current branch ($GIT_BRANCH) != project branch ($PROJECT_BRANCH), using current branch as base: $BASE_BRANCH"
else
  echo "Current branch ($GIT_BRANCH) == project branch ($PROJECT_BRANCH), using default branch: $BASE_BRANCH"
fi

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

echo "Checking if project branch is behind base branch..."
MERGE_NEEDED=false
if git rev-parse --verify "$BASE_BRANCH" > /dev/null 2>&1; then
  MERGE_BASE=$(git merge-base HEAD "$BASE_BRANCH")
  BASE_COMMIT=$(git rev-parse "$BASE_BRANCH")
  HEAD_COMMIT=$(git rev-parse HEAD)
  
  if [ "$MERGE_BASE" != "$BASE_COMMIT" ]; then
    echo "Project branch is behind base branch by $(git rev-list --count "$BASE_BRANCH"..HEAD) commit(s)"
    MERGE_NEEDED=true
    
    echo "Attempting to merge base branch..."
    if git merge "$BASE_BRANCH" --no-edit; then
      echo "Merge successful (fast-forward or no conflicts)"
    else
      echo "Merge had conflicts - resolving with AI..."
      git merge --abort || true
      
      echo "Running AI to resolve merge conflicts..."
      cat > /tmp/merge-instructions.md << 'EOF'
You need to resolve merge conflicts between the base branch ($BASE_BRANCH) and the current branch ($PROJECT_BRANCH).

Steps:
1. Run 'git merge $BASE_BRANCH' to see the conflicts
2. Examine the conflicting files and resolve each conflict
3. After resolving, run 'git add <resolved-files>' and 'git commit'
4. Write a brief summary of the merge to 'report.md'

Focus on accepting the correct changes from both branches. If there are test failures after resolving, fix them.
EOF
      
      ralph_run /tmp/merge-instructions.md --local{{.VerboseFlag}} --no-notify || true
      
      if git ls-files --others --exclude-standard | grep -q "report.md"; then
        echo "AI generated merge summary"
      fi
      
      if git diff --cached --quiet; then
        echo "AI did not commit the merge - committing now..."
        git add -A
        git commit -m "Merge $BASE_BRANCH into $PROJECT_BRANCH" || true
      fi
    fi
  else
    echo "Project branch is up-to-date with base branch"
  fi
else
  echo "Base branch $BASE_BRANCH not found locally, skipping merge check"
fi

echo "Running ralph..."
ralph_run "$PROJECT_PATH" --local{{.VerboseFlag}} --no-notify

opencode stats

echo "Execution complete!"
