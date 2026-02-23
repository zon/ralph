package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
)

// buildParameters builds workflow parameters from the params map
func buildParameters(params map[string]string) []map[string]interface{} {
	allParams := []string{"project-path", "instructions-md"}
	var parameters []map[string]interface{}
	for _, name := range allParams {
		param := map[string]interface{}{"name": name}
		if value, exists := params[name]; exists {
			param["value"] = value
		} else {
			param["value"] = ""
		}
		parameters = append(parameters, param)
	}
	return parameters
}

// buildExecutionScript builds the shell script that runs in the container
func buildExecutionScript(dryRun, verbose bool, cfg *config.RalphConfig) string {
	ralphCmd := "ralph \"$PROJECT_PATH\" --local"
	if dryRun {
		ralphCmd += " --dry-run"
	}
	if verbose {
		ralphCmd += " --verbose"
	}
	ralphCmd += " --no-notify"

	appBotName := config.DefaultAppName + "[bot]"
	appBotEmail := config.DefaultAppName + "[bot]@users.noreply.github.com"

	script := fmt.Sprintf(`#!/bin/sh
set -e

echo "Setting up GitHub App token..."
export GITHUB_TOKEN=$(ralph github-token --owner "$GITHUB_REPO_OWNER" --repo "$GITHUB_REPO_NAME")

echo "Configuring git for HTTPS authentication..."
git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

echo "Setting up OpenCode credentials..."
mkdir -p ~/.local/share/opencode
cp /secrets/opencode/auth.json ~/.local/share/opencode/auth.json

echo "Configuring git user..."
git config --global user.name "%s"
git config --global user.email "%s"

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

echo "Writing parameter files..."
mkdir -p /workspace/repo/.ralph

if [ -n "$INSTRUCTIONS_MD" ]; then
  printf '%%s' "$INSTRUCTIONS_MD" > /workspace/repo/.ralph/instructions.md
fi

echo "Running ralph..."
%s

echo "Execution complete!"
`, appBotName, appBotEmail, ralphCmd)
	return script
}

// buildMergeScript builds the shell script that checks requirements and merges the PR
func buildMergeScript() string {
	appBotName := config.DefaultAppName + "[bot]"
	appBotEmail := config.DefaultAppName + "[bot]@users.noreply.github.com"

	script := fmt.Sprintf(`#!/bin/sh
set -e

echo "Setting up GitHub App token..."
export GITHUB_TOKEN=$(ralph github-token --owner "$GITHUB_REPO_OWNER" --repo "$GITHUB_REPO_NAME")

echo "Configuring git for HTTPS authentication..."
git config --global url."https://x-access-token:${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

echo "Configuring git user..."
git config --global user.name "%s"
git config --global user.email "%s"

echo "Cloning repository: $GIT_REPO_URL"
git clone -b "$GIT_BRANCH" "$GIT_REPO_URL" /workspace/repo
cd /workspace/repo

echo "Checking out PR branch: $PR_BRANCH"
git fetch origin "$PR_BRANCH"
git checkout "$PR_BRANCH"

echo "Checking requirement status..."
PASSING=$(ralph --once "$PROJECT_PATH" --dry-run 2>&1 | grep -c "passing: true" || true)
FAILING=$(ralph --once "$PROJECT_PATH" --dry-run 2>&1 | grep -c "passing: false" || true)

# Check all requirements pass using ralph's own config parsing
cat "$PROJECT_PATH" | grep "passing: false" > /tmp/failing_reqs.txt 2>&1 || true
if [ -s /tmp/failing_reqs.txt ]; then
  echo "Not all requirements are passing. Aborting merge."
  cat /tmp/failing_reqs.txt
  exit 0
fi

echo "All requirements passing. Proceeding with merge..."

echo "Removing project file: $PROJECT_PATH"
rm "$PROJECT_PATH"

echo "Committing deletion of project file..."
git add -A
git commit -m "Remove completed project file: $PROJECT_PATH"

echo "Pushing changes..."
git push origin "$PR_BRANCH"

echo "Merging PR via gh CLI..."
gh pr merge "$PR_BRANCH" --merge --delete-branch

echo "Merge complete!"
`, appBotName, appBotEmail)
	return script
}

// buildVolumeMounts builds volume mounts for secrets and configMaps
func buildVolumeMounts(cfg *config.RalphConfig) []map[string]interface{} {
	mounts := []map[string]interface{}{
		{"name": "github-credentials", "mountPath": "/secrets/github", "readOnly": true},
		{"name": "opencode-credentials", "mountPath": "/secrets/opencode", "readOnly": true},
	}

	for i, cm := range cfg.Workflow.ConfigMaps {
		mount := map[string]interface{}{
			"name":     sanitizeName(cm.Name),
			"readOnly": true,
		}
		if cm.DestFile != "" {
			mount["mountPath"] = cm.DestFile
			mount["subPath"] = filepath.Base(cm.DestFile)
			mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(cm.Name), i)
		} else if cm.DestDir != "" {
			mount["mountPath"] = cm.DestDir
		} else {
			mount["mountPath"] = fmt.Sprintf("/configmaps/%s", cm.Name)
		}
		mounts = append(mounts, mount)
	}

	for i, secret := range cfg.Workflow.Secrets {
		mount := map[string]interface{}{
			"name":     sanitizeName(secret.Name),
			"readOnly": true,
		}
		if secret.DestFile != "" {
			mount["mountPath"] = secret.DestFile
			mount["subPath"] = filepath.Base(secret.DestFile)
			mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(secret.Name), i)
		} else if secret.DestDir != "" {
			mount["mountPath"] = secret.DestDir
		} else {
			mount["mountPath"] = fmt.Sprintf("/secrets/%s", secret.Name)
		}
		mounts = append(mounts, mount)
	}

	return mounts
}

// buildVolumes builds volumes for secrets and configMaps
func buildVolumes(cfg *config.RalphConfig) []map[string]interface{} {
	volumes := []map[string]interface{}{
		{
			"name": "github-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.GitHubSecretName,
			},
		},
		{
			"name": "opencode-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.OpenCodeSecretName,
			},
		},
	}

	for i, cm := range cfg.Workflow.ConfigMaps {
		volumeName := sanitizeName(cm.Name)
		if cm.DestFile != "" {
			volumeName = fmt.Sprintf("%s-%d", sanitizeName(cm.Name), i)
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"configMap": map[string]interface{}{
					"name": cm.Name,
					"items": []map[string]interface{}{
						{"key": filepath.Base(cm.DestFile), "path": filepath.Base(cm.DestFile)},
					},
				},
			})
		} else {
			volumes = append(volumes, map[string]interface{}{
				"name":      volumeName,
				"configMap": map[string]interface{}{"name": cm.Name},
			})
		}
	}

	for i, secret := range cfg.Workflow.Secrets {
		volumeName := sanitizeName(secret.Name)
		if secret.DestFile != "" {
			volumeName = fmt.Sprintf("%s-%d", sanitizeName(secret.Name), i)
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"secret": map[string]interface{}{
					"secretName": secret.Name,
					"items": []map[string]interface{}{
						{"key": filepath.Base(secret.DestFile), "path": filepath.Base(secret.DestFile)},
					},
				},
			})
		} else {
			volumes = append(volumes, map[string]interface{}{
				"name":   volumeName,
				"secret": map[string]interface{}{"secretName": secret.Name},
			})
		}
	}

	return volumes
}

// sanitizeName sanitizes a name for use as a Kubernetes volume/resource name.
func sanitizeName(name string) string {
	sanitized := strings.ReplaceAll(name, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	return strings.ToLower(sanitized)
}
