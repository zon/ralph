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
	"gopkg.in/yaml.v3"
)

// DefaultContainerVersion is the default container image tag (set via ldflags during build)
var DefaultContainerVersion = "latest"

// GenerateWorkflow generates an Argo Workflow YAML for remote execution
func GenerateWorkflow(ctx *execcontext.Context, projectName string) (string, error) {
	// Get current branch
	currentBranch, err := getCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get git remote URL
	remoteURL, err := getRemoteURL()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}

	return GenerateWorkflowWithGitInfo(ctx, projectName, remoteURL, currentBranch)
}

// GenerateWorkflowWithGitInfo generates an Argo Workflow YAML with provided git information
// This allows for easier testing by accepting git info as parameters
func GenerateWorkflowWithGitInfo(ctx *execcontext.Context, projectName, repoURL, branch string) (string, error) {
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
	imageTag := DefaultContainerVersion
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
			branch,
			params,
			ralphConfig,
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
func buildWorkflowSpec(image, repoURL, branch string, params map[string]string, cfg *config.RalphConfig) map[string]interface{} {
	spec := map[string]interface{}{
		"entrypoint": "ralph-executor",
		// TTL to auto-delete after 1 day
		"ttlStrategy": map[string]interface{}{
			"secondsAfterCompletion": 86400, // 1 day
		},
		// Delete pods after completion
		"podGC": map[string]interface{}{
			"strategy": "OnWorkflowCompletion",
		},
		"arguments": map[string]interface{}{
			"parameters": buildParameters(params),
		},
		"templates": []interface{}{
			buildMainTemplate(image, repoURL, branch, cfg),
		},
	}

	return spec
}

// buildParameters builds workflow parameters from the params map
func buildParameters(params map[string]string) []map[string]interface{} {
	var parameters []map[string]interface{}
	for name, value := range params {
		parameters = append(parameters, map[string]interface{}{
			"name":  name,
			"value": value,
		})
	}
	return parameters
}

// buildMainTemplate builds the main execution template
func buildMainTemplate(image, repoURL, branch string, cfg *config.RalphConfig) map[string]interface{} {
	template := map[string]interface{}{
		"name": "ralph-executor",
		"container": map[string]interface{}{
			"image": image,
			"command": []string{
				"/bin/sh",
				"-c",
			},
			"args": []string{
				buildExecutionScript(),
			},
			"env":          buildEnvVars(repoURL, branch, cfg),
			"volumeMounts": buildVolumeMounts(cfg),
			"workingDir":   "/workspace",
		},
		"volumes": buildVolumes(cfg),
	}

	return template
}

// buildExecutionScript builds the shell script that runs in the container
func buildExecutionScript() string {
	script := `#!/bin/sh
set -e

echo "Setting up git credentials..."
mkdir -p ~/.ssh
cp /secrets/git/ssh-privatekey ~/.ssh/id_ed25519
chmod 600 ~/.ssh/id_ed25519
ssh-keyscan github.com >> ~/.ssh/known_hosts

echo "Setting up GitHub token..."
export GITHUB_TOKEN=$(cat /secrets/github/token)

echo "Setting up OpenCode credentials..."
mkdir -p ~/.local/share/opencode
cp /secrets/opencode/auth.json ~/.local/share/opencode/auth.json

echo "Cloning repository: $GIT_REPO_URL"
git clone -b "$GIT_BRANCH" "$GIT_REPO_URL" /workspace/repo
cd /workspace/repo

echo "Writing parameter files..."
mkdir -p /workspace/repo/.ralph
echo "$PROJECT_FILE" > /tmp/project.yaml

if [ -n "$CONFIG_YAML" ]; then
  echo "$CONFIG_YAML" > /workspace/repo/.ralph/config.yaml
fi

if [ -n "$INSTRUCTIONS_MD" ]; then
  echo "$INSTRUCTIONS_MD" > /workspace/repo/.ralph/instructions.md
fi

echo "Running ralph..."
ralph /tmp/project.yaml

echo "Execution complete!"
`
	return script
}

// buildEnvVars builds environment variables for the container
func buildEnvVars(repoURL, branch string, cfg *config.RalphConfig) []map[string]interface{} {
	envVars := []map[string]interface{}{
		{
			"name":  "GIT_REPO_URL",
			"value": repoURL,
		},
		{
			"name":  "GIT_BRANCH",
			"value": branch,
		},
		{
			"name": "PROJECT_FILE",
			"valueFrom": map[string]interface{}{
				"parameter": map[string]string{
					"name": "project-file",
				},
			},
		},
	}

	// Add optional parameters
	envVars = append(envVars, map[string]interface{}{
		"name": "CONFIG_YAML",
		"valueFrom": map[string]interface{}{
			"parameter": map[string]interface{}{
				"name":    "config-yaml",
				"default": "",
			},
		},
	})

	envVars = append(envVars, map[string]interface{}{
		"name": "INSTRUCTIONS_MD",
		"valueFrom": map[string]interface{}{
			"parameter": map[string]interface{}{
				"name":    "instructions-md",
				"default": "",
			},
		},
	})

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
			"name":      "git-credentials",
			"mountPath": "/secrets/git",
			"readOnly":  true,
		},
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
	for _, cm := range cfg.Workflow.ConfigMaps {
		mounts = append(mounts, map[string]interface{}{
			"name":      sanitizeName(cm),
			"mountPath": fmt.Sprintf("/configmaps/%s", cm),
			"readOnly":  true,
		})
	}

	// Add user-specified secrets
	for _, secret := range cfg.Workflow.Secrets {
		mounts = append(mounts, map[string]interface{}{
			"name":      sanitizeName(secret),
			"mountPath": fmt.Sprintf("/secrets/%s", secret),
			"readOnly":  true,
		})
	}

	return mounts
}

// buildVolumes builds volumes for secrets and configMaps
func buildVolumes(cfg *config.RalphConfig) []map[string]interface{} {
	volumes := []map[string]interface{}{
		{
			"name": "git-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.GitSecretName,
			},
		},
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
	for _, cm := range cfg.Workflow.ConfigMaps {
		volumes = append(volumes, map[string]interface{}{
			"name": sanitizeName(cm),
			"configMap": map[string]interface{}{
				"name": cm,
			},
		})
	}

	// Add user-specified secrets
	for _, secret := range cfg.Workflow.Secrets {
		volumes = append(volumes, map[string]interface{}{
			"name": sanitizeName(secret),
			"secret": map[string]interface{}{
				"secretName": secret,
			},
		})
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

// SubmitWorkflow submits a workflow to Argo
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

	return string(output), nil
}
