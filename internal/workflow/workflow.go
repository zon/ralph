package workflow

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"gopkg.in/yaml.v3"
)

// Workflow holds all inputs required to generate and submit an Argo Workflow for a ralph run.
type Workflow struct {
	// ProjectName is used in the workflow's generateName field (e.g. "my-feature").
	ProjectName string
	// RepoURL is the HTTPS URL of the git repository (e.g. "https://github.com/owner/repo.git").
	RepoURL string
	// RepoOwner is the GitHub organisation or user (e.g. "zon").
	RepoOwner string
	// RepoName is the repository name (e.g. "ralph").
	RepoName string
	// CloneBranch is the branch the container clones initially (typically the base/current branch).
	CloneBranch string
	// ProjectBranch is the branch the container creates/checks-out to do its work.
	ProjectBranch string
	// ProjectPath is the relative path to the project YAML file inside the repo.
	ProjectPath string
	// Instructions is the contents of the instructions file to inject into the container (may be empty).
	Instructions string
	// DryRun controls whether the ralph command inside the container runs with --dry-run.
	DryRun bool
	// Verbose controls whether the ralph command inside the container runs with --verbose.
	Verbose bool
	// Watch controls whether argo submit is called with --watch.
	Watch bool
	// RalphConfig supplies workflow-level configuration (image overrides, secrets, configmaps, env).
	RalphConfig *config.RalphConfig
}

// Render produces the Argo Workflow YAML string for this Workflow.
func (w *Workflow) Render() (string, error) {
	params := map[string]string{
		"project-path":    w.ProjectPath,
		"instructions-md": w.Instructions,
	}

	wf := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]interface{}{
			"generateName": fmt.Sprintf("ralph-%s-", w.ProjectName),
		},
		"spec": map[string]interface{}{
			"entrypoint": "ralph-executor",
			"ttlStrategy": map[string]interface{}{
				"secondsAfterCompletion": 86400,
			},
			"podGC": map[string]interface{}{
				"strategy":            "OnWorkflowCompletion",
				"deleteDelayDuration": "10m",
			},
			"arguments": map[string]interface{}{
				"parameters": buildParameters(params),
			},
			"templates": []interface{}{
				w.buildMainTemplate(),
			},
		},
	}

	yamlData, err := yaml.Marshal(wf)
	if err != nil {
		return "", fmt.Errorf("failed to marshal workflow to YAML: %w", err)
	}
	return string(yamlData), nil
}

// Submit renders and submits this Workflow to Argo, returning the workflow name.
// namespace is required and determines the Kubernetes namespace for the workflow.
func (w *Workflow) Submit(namespace string) (string, error) {
	workflowYAML, err := w.Render()
	if err != nil {
		return "", err
	}
	return submitYAML(workflowYAML, w.RalphConfig, w.Watch, namespace)
}

func (w *Workflow) buildMainTemplate() map[string]interface{} {
	return map[string]interface{}{
		"name": "ralph-executor",
		"container": map[string]interface{}{
			"image":        resolveImage(w.RalphConfig),
			"command":      []string{"/bin/sh", "-c"},
			"args":         []string{buildExecutionScript(w.DryRun, w.Verbose, w.RalphConfig)},
			"env":          w.buildEnvVars(),
			"volumeMounts": buildVolumeMounts(w.RalphConfig),
			"workingDir":   "/workspace",
		},
		"volumes": buildVolumes(w.RalphConfig),
	}
}

func (w *Workflow) buildEnvVars() []map[string]interface{} {
	envVars := []map[string]interface{}{
		{"name": "GIT_REPO_URL", "value": w.RepoURL},
		{"name": "GITHUB_REPO_OWNER", "value": w.RepoOwner},
		{"name": "GITHUB_REPO_NAME", "value": w.RepoName},
		{"name": "GIT_BRANCH", "value": w.CloneBranch},
		{"name": "PROJECT_BRANCH", "value": w.ProjectBranch},
		{"name": "PROJECT_PATH", "value": "{{workflow.parameters.project-path}}"},
		{"name": "INSTRUCTIONS_MD", "value": "{{workflow.parameters.instructions-md}}"},
		{"name": "RALPH_WORKFLOW_EXECUTION", "value": "true"},
		{"name": "BASE_BRANCH", "value": w.RalphConfig.BaseBranch},
	}

	for key, value := range w.RalphConfig.Workflow.Env {
		envVars = append(envVars, map[string]interface{}{
			"name":  key,
			"value": value,
		})
	}

	return envVars
}
