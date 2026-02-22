package workflow

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	execcontext "github.com/zon/ralph/internal/context"
	"github.com/zon/ralph/internal/k8s"
	"github.com/zon/ralph/internal/version"
	"gopkg.in/yaml.v3"
)

// DefaultContainerVersion returns the default container image tag read from the embedded VERSION file.
// Kept as a function for use in tests.
func DefaultContainerVersion() string {
	return version.Version()
}

// GenerateWorkflow generates an Argo Workflow YAML for remote execution.
// cloneBranch is the branch the container will clone (current local branch).
// projectBranch is the branch the container will create and work on (derived from the project file name).
func GenerateWorkflow(ctx *execcontext.Context, projectName, cloneBranch, projectBranch string, dryRun, verbose bool) (string, error) {
	// Get git remote URL
	remoteURL, err := getRemoteURL()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	// Get git repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get absolute path to project file
	absProjectFile, err := filepath.Abs(ctx.ProjectFile)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project file path: %w", err)
	}

	// Calculate relative path from repo root
	relProjectPath, err := filepath.Rel(repoRoot, absProjectFile)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative project path: %w", err)
	}

	return GenerateWorkflowWithGitInfo(ctx, projectName, remoteURL, cloneBranch, projectBranch, relProjectPath, dryRun, verbose)
}

// GenerateWorkflowWithGitInfo generates an Argo Workflow YAML with provided git information
// This allows for easier testing by accepting git info as parameters
func GenerateWorkflowWithGitInfo(ctx *execcontext.Context, projectName, repoURL, cloneBranch, projectBranch, relProjectPath string, dryRun, verbose bool) (string, error) {
	// Load ralph config
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Read project file content
	projectContent, err := os.ReadFile(ctx.ProjectFile)
	if err != nil {
		return "", fmt.Errorf("failed to read project file: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Build workflow parameters
	params := map[string]string{
		"project-file": string(projectContent),
		"project-path": relProjectPath,
	}

	// Check for config.yaml
	configPath := filepath.Join(cwd, ".ralph", "config.yaml")
	if configData, err := os.ReadFile(configPath); err == nil {
		params["config-yaml"] = string(configData)
	}

	// Check for instructions.md
	instructionsPath := filepath.Join(cwd, ".ralph", "instructions.md")
	if instructionsData, err := os.ReadFile(instructionsPath); err == nil {
		params["instructions-md"] = string(instructionsData)
	}

	// Determine image repository and tag
	imageRepo := "ghcr.io/zon/ralph"
	imageTag := DefaultContainerVersion()
	if ralphConfig.Workflow.Image.Repository != "" {
		imageRepo = ralphConfig.Workflow.Image.Repository
	}
	if ralphConfig.Workflow.Image.Tag != "" {
		imageTag = ralphConfig.Workflow.Image.Tag
	}
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	// Build the workflow structure
	workflow := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]interface{}{
			"generateName": fmt.Sprintf("ralph-%s-", projectName),
		},
		"spec": buildWorkflowSpec(
			image,
			repoURL,
			cloneBranch,
			projectBranch,
			params,
			ralphConfig,
			dryRun,
			verbose,
		),
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow to YAML: %w", err)
	}

	return string(yamlData), nil
}

// buildWorkflowSpec constructs the workflow spec
func buildWorkflowSpec(image, repoURL, cloneBranch, projectBranch string, params map[string]string, cfg *config.RalphConfig, dryRun, verbose bool) map[string]interface{} {
	spec := map[string]interface{}{
		"entrypoint": "ralph-executor",
		// TTL to auto-delete after 1 day
		"ttlStrategy": map[string]interface{}{
			"secondsAfterCompletion": 86400, // 1 day
		},
		// Keep pods for 10 minutes after completion for log inspection
		"podGC": map[string]interface{}{
			"strategy":            "OnWorkflowCompletion",
			"deleteDelayDuration": "10m",
		},
		"arguments": map[string]interface{}{
			"parameters": buildParameters(params),
		},
		"templates": []interface{}{
			buildMainTemplate(image, repoURL, cloneBranch, projectBranch, cfg, dryRun, verbose),
		},
	}

	return spec
}

// buildParameters builds workflow parameters from the params map
func buildParameters(params map[string]string) []map[string]interface{} {
	// Define required and optional parameters
	allParams := []string{"project-file", "project-path", "config-yaml", "instructions-md"}
	var parameters []map[string]interface{}

	for _, name := range allParams {
		param := map[string]interface{}{
			"name": name,
		}

		if value, exists := params[name]; exists {
			param["value"] = value
		} else {
			// Set default empty string for optional parameters
			param["value"] = ""
		}

		parameters = append(parameters, param)
	}

	return parameters
}

// buildMainTemplate builds the main execution template
func buildMainTemplate(image, repoURL, cloneBranch, projectBranch string, cfg *config.RalphConfig, dryRun, verbose bool) map[string]interface{} {
	template := map[string]interface{}{
		"name": "ralph-executor",
		"container": map[string]interface{}{
			"image": image,
			"command": []string{
				"/bin/sh",
				"-c",
			},
			"args": []string{
				buildExecutionScript(dryRun, verbose, cfg),
			},
			"env":          buildEnvVars(repoURL, cloneBranch, projectBranch, cfg),
			"volumeMounts": buildVolumeMounts(cfg),
			"workingDir":   "/workspace",
		},
		"volumes": buildVolumes(cfg),
	}

	return template
}

// buildExecutionScript builds the shell script that runs in the container
func buildExecutionScript(dryRun, verbose bool, cfg *config.RalphConfig) string {
	// Build ralph command with flags
	// Always pass --local so the container executes directly instead of submitting another workflow
	ralphCmd := "ralph \"$PROJECT_PATH\" --local"
	if dryRun {
		ralphCmd += " --dry-run"
	}
	if verbose {
		ralphCmd += " --verbose"
	}
	// Always disable notifications when running in workflow container
	ralphCmd += " --no-notify"

	appBotName := config.DefaultAppName + "[bot]"
	appBotEmail := config.DefaultAppName + "[bot]@users.noreply.github.com"

	script := fmt.Sprintf(`#!/bin/sh
set -e

echo "Setting up GitHub App token..."
export GITHUB_TOKEN=$(ralph github-token)

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

echo "Writing project file to: $PROJECT_PATH"
mkdir -p "$(dirname "$PROJECT_PATH")"
echo "$PROJECT_FILE" > "$PROJECT_PATH"

echo "Writing parameter files..."
mkdir -p /workspace/repo/.ralph

if [ -n "$CONFIG_YAML" ]; then
  printf '%%s' "$CONFIG_YAML" > /workspace/repo/.ralph/config.yaml
fi

if [ -n "$INSTRUCTIONS_MD" ]; then
  printf '%%s' "$INSTRUCTIONS_MD" > /workspace/repo/.ralph/instructions.md
fi

echo "Running ralph..."
%s

echo "Execution complete!"
`, appBotName, appBotEmail, ralphCmd)
	return script
}

// buildEnvVars builds environment variables for the container
func buildEnvVars(repoURL, cloneBranch, projectBranch string, cfg *config.RalphConfig) []map[string]interface{} {
	envVars := []map[string]interface{}{
		{
			"name":  "GIT_REPO_URL",
			"value": repoURL,
		},
		{
			"name":  "GIT_BRANCH",
			"value": cloneBranch,
		},
		{
			"name":  "PROJECT_BRANCH",
			"value": projectBranch,
		},
		{
			"name":  "PROJECT_FILE",
			"value": "{{workflow.parameters.project-file}}",
		},
		{
			"name":  "PROJECT_PATH",
			"value": "{{workflow.parameters.project-path}}",
		},
		{
			"name":  "CONFIG_YAML",
			"value": "{{workflow.parameters.config-yaml}}",
		},
		{
			"name":  "INSTRUCTIONS_MD",
			"value": "{{workflow.parameters.instructions-md}}",
		},
		{
			"name":  "RALPH_WORKFLOW_EXECUTION",
			"value": "true",
		},
		{
			"name":  "BASE_BRANCH",
			"value": cfg.BaseBranch,
		},
	}

	// Add user-specified environment variables from config
	for key, value := range cfg.Workflow.Env {
		envVars = append(envVars, map[string]interface{}{
			"name":  key,
			"value": value,
		})
	}

	return envVars
}

// buildVolumeMounts builds volume mounts for secrets and configMaps
func buildVolumeMounts(cfg *config.RalphConfig) []map[string]interface{} {
	mounts := []map[string]interface{}{
		{
			"name":      "github-credentials",
			"mountPath": "/secrets/github",
			"readOnly":  true,
		},
		{
			"name":      "opencode-credentials",
			"mountPath": "/secrets/opencode",
			"readOnly":  true,
		},
	}

	// Add user-specified configMaps
	for i, cm := range cfg.Workflow.ConfigMaps {
		mount := map[string]interface{}{
			"name":     sanitizeName(cm.Name),
			"readOnly": true,
		}

		if cm.DestFile != "" {
			// Mount specific key (filename) to the destination file path
			mount["mountPath"] = cm.DestFile
			mount["subPath"] = filepath.Base(cm.DestFile)
		} else if cm.DestDir != "" {
			// Mount entire ConfigMap to the destination directory
			mount["mountPath"] = cm.DestDir
		} else {
			// Fallback: mount to a default location
			mount["mountPath"] = fmt.Sprintf("/configmaps/%s", cm.Name)
		}

		// If mounting multiple items from the same ConfigMap to different paths,
		// we need unique volume names
		if cm.DestFile != "" {
			mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(cm.Name), i)
		}

		mounts = append(mounts, mount)
	}

	// Add user-specified secrets
	for i, secret := range cfg.Workflow.Secrets {
		mount := map[string]interface{}{
			"name":     sanitizeName(secret.Name),
			"readOnly": true,
		}

		if secret.DestFile != "" {
			// Mount specific key (filename) to the destination file path
			mount["mountPath"] = secret.DestFile
			mount["subPath"] = filepath.Base(secret.DestFile)
		} else if secret.DestDir != "" {
			// Mount entire Secret to the destination directory
			mount["mountPath"] = secret.DestDir
		} else {
			// Fallback: mount to a default location
			mount["mountPath"] = fmt.Sprintf("/secrets/%s", secret.Name)
		}

		// If mounting multiple items from the same Secret to different paths,
		// we need unique volume names
		if secret.DestFile != "" {
			mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(secret.Name), i)
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

	// Add user-specified configMaps
	for i, cm := range cfg.Workflow.ConfigMaps {
		volumeName := sanitizeName(cm.Name)

		// If mounting a specific file, we need a unique volume name
		// and use items to select the specific key
		if cm.DestFile != "" {
			volumeName = fmt.Sprintf("%s-%d", sanitizeName(cm.Name), i)
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"configMap": map[string]interface{}{
					"name": cm.Name,
					"items": []map[string]interface{}{
						{
							"key":  filepath.Base(cm.DestFile),
							"path": filepath.Base(cm.DestFile),
						},
					},
				},
			})
		} else {
			// Mount entire ConfigMap
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"configMap": map[string]interface{}{
					"name": cm.Name,
				},
			})
		}
	}

	// Add user-specified secrets
	for i, secret := range cfg.Workflow.Secrets {
		volumeName := sanitizeName(secret.Name)

		// If mounting a specific file, we need a unique volume name
		// and use items to select the specific key
		if secret.DestFile != "" {
			volumeName = fmt.Sprintf("%s-%d", sanitizeName(secret.Name), i)
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"secret": map[string]interface{}{
					"secretName": secret.Name,
					"items": []map[string]interface{}{
						{
							"key":  filepath.Base(secret.DestFile),
							"path": filepath.Base(secret.DestFile),
						},
					},
				},
			})
		} else {
			// Mount entire Secret
			volumes = append(volumes, map[string]interface{}{
				"name": volumeName,
				"secret": map[string]interface{}{
					"secretName": secret.Name,
				},
			})
		}
	}

	return volumes
}

// sanitizeName sanitizes a name for use as a volume name
func sanitizeName(name string) string {
	// Replace invalid characters with hyphens
	sanitized := strings.ReplaceAll(name, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	return strings.ToLower(sanitized)
}

// getCurrentBranch gets the current git branch
func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getRemoteURL gets the git remote URL
func getRemoteURL() (string, error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getRepoRoot gets the git repository root directory
func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// SubmitWorkflow submits a workflow to Argo and returns the workflow name
func SubmitWorkflow(ctx *execcontext.Context, workflowYAML string, ralphConfig *config.RalphConfig) (string, error) {
	// Check if argo CLI is installed
	if _, err := exec.LookPath("argo"); err != nil {
		return "", fmt.Errorf("argo CLI not found - please install Argo CLI to use remote execution: https://github.com/argoproj/argo-workflows/releases")
	}

	// Determine namespace
	namespace := ralphConfig.Workflow.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Determine context
	kubeContext := ralphConfig.Workflow.Context

	// Build argo submit command
	args := []string{"submit", "-"}

	// Add namespace
	args = append(args, "-n", namespace)

	// Add context if specified
	if kubeContext != "" {
		args = append(args, "--context", kubeContext)
	}

	// Add watch flag if requested
	if ctx.ShouldWatch() {
		args = append(args, "--watch")
	}

	// Execute argo submit
	cmd := exec.CommandContext(context.Background(), "argo", args...)
	cmd.Stdin = strings.NewReader(workflowYAML)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to submit workflow: %w\nOutput: %s", err, string(output))
	}

	// Extract workflow name from output
	// Output format: "Name:                ralph-project-name-xxxxx"
	workflowName := extractWorkflowName(string(output))
	if workflowName == "" {
		// Fallback: return first line if we can't extract name
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) > 0 {
			workflowName = strings.TrimSpace(lines[0])
		}
	}

	return workflowName, nil
}

// GenerateMergeWorkflow generates an Argo Workflow YAML for merging a PR.
// It clones the repo, checks out prBranch, verifies all requirements pass,
// removes the project file, commits and pushes, then merges the PR via gh CLI.
func GenerateMergeWorkflow(projectFile, prBranch string) (string, error) {
	// Get git remote URL
	remoteURL, err := getRemoteURL()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	// Get current branch (base branch to clone from)
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get git repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	// Get absolute path to project file
	absProjectFile, err := filepath.Abs(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project file path: %w", err)
	}

	// Calculate relative path from repo root
	relProjectPath, err := filepath.Rel(repoRoot, absProjectFile)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative project path: %w", err)
	}

	return GenerateMergeWorkflowWithGitInfo(projectFile, remoteURL, currentBranch, prBranch, relProjectPath)
}

// GenerateMergeWorkflowWithGitInfo generates a merge Argo Workflow YAML with provided git information.
// This allows for easier testing by accepting git info as parameters.
func GenerateMergeWorkflowWithGitInfo(projectFile, repoURL, cloneBranch, prBranch, relProjectPath string) (string, error) {
	// Load ralph config
	ralphConfig, err := config.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	// Read project file content
	projectContent, err := os.ReadFile(projectFile)
	if err != nil {
		return "", fmt.Errorf("failed to read project file: %w", err)
	}

	// Determine image repository and tag
	imageRepo := "ghcr.io/zon/ralph"
	imageTag := DefaultContainerVersion()
	if ralphConfig.Workflow.Image.Repository != "" {
		imageRepo = ralphConfig.Workflow.Image.Repository
	}
	if ralphConfig.Workflow.Image.Tag != "" {
		imageTag = ralphConfig.Workflow.Image.Tag
	}
	image := fmt.Sprintf("%s:%s", imageRepo, imageTag)

	// Build workflow parameters
	params := []map[string]interface{}{
		{"name": "project-file", "value": string(projectContent)},
		{"name": "project-path", "value": relProjectPath},
	}

	// Build the workflow structure
	workflow := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]interface{}{
			"generateName": "ralph-merge-",
		},
		"spec": map[string]interface{}{
			"entrypoint": "ralph-merger",
			"ttlStrategy": map[string]interface{}{
				"secondsAfterCompletion": 86400,
			},
			"podGC": map[string]interface{}{
				"strategy":            "OnWorkflowCompletion",
				"deleteDelayDuration": "10m",
			},
			"arguments": map[string]interface{}{
				"parameters": params,
			},
			"templates": []interface{}{
				buildMergeTemplate(image, repoURL, cloneBranch, prBranch, ralphConfig),
			},
		},
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(workflow)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow to YAML: %w", err)
	}

	return string(yamlData), nil
}

// buildMergeTemplate builds the merge execution template
func buildMergeTemplate(image, repoURL, cloneBranch, prBranch string, cfg *config.RalphConfig) map[string]interface{} {
	template := map[string]interface{}{
		"name": "ralph-merger",
		"container": map[string]interface{}{
			"image": image,
			"command": []string{
				"/bin/sh",
				"-c",
			},
			"args": []string{
				buildMergeScript(),
			},
			"env": []map[string]interface{}{
				{"name": "GIT_REPO_URL", "value": repoURL},
				{"name": "GIT_BRANCH", "value": cloneBranch},
				{"name": "PR_BRANCH", "value": prBranch},
				{"name": "PROJECT_FILE", "value": "{{workflow.parameters.project-file}}"},
				{"name": "PROJECT_PATH", "value": "{{workflow.parameters.project-path}}"},
			},
			"volumeMounts": []map[string]interface{}{
				{"name": "github-credentials", "mountPath": "/secrets/github", "readOnly": true},
			},
			"workingDir": "/workspace",
		},
		"volumes": []map[string]interface{}{
			{
				"name": "github-credentials",
				"secret": map[string]interface{}{
					"secretName": k8s.GitHubSecretName,
				},
			},
		},
	}

	return template
}

// buildMergeScript builds the shell script that checks requirements and merges the PR
func buildMergeScript() string {
	appBotName := config.DefaultAppName + "[bot]"
	appBotEmail := config.DefaultAppName + "[bot]@users.noreply.github.com"

	script := fmt.Sprintf(`#!/bin/sh
set -e

echo "Setting up GitHub App token..."
export GITHUB_TOKEN=$(ralph github-token)

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

echo "Writing project file to: $PROJECT_PATH"
mkdir -p "$(dirname "$PROJECT_PATH")"
echo "$PROJECT_FILE" > "$PROJECT_PATH"

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

// extractWorkflowName extracts the workflow name from argo submit output
func extractWorkflowName(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "Name:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}
