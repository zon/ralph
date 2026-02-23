package workflow

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
	"gopkg.in/yaml.v3"
)

// MergeWorkflow holds all inputs required to generate and submit an Argo Workflow for a ralph merge.
type MergeWorkflow struct {
	// RepoURL is the HTTPS URL of the git repository.
	RepoURL string
	// RepoOwner is the GitHub organisation or user.
	RepoOwner string
	// RepoName is the repository name.
	RepoName string
	// CloneBranch is the branch the container clones initially (typically the base branch).
	CloneBranch string
	// PRBranch is the PR branch to merge.
	PRBranch string
	// ProjectPath is the relative path to the project YAML file inside the repo.
	ProjectPath string
	// Watch controls whether argo submit is called with --watch.
	Watch bool
	// RalphConfig supplies workflow-level configuration.
	RalphConfig *config.RalphConfig
}

// Render produces the Argo Workflow YAML string for this MergeWorkflow.
func (m *MergeWorkflow) Render() (string, error) {
	wf := map[string]interface{}{
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
				"parameters": []map[string]interface{}{
					{"name": "project-path", "value": m.ProjectPath},
				},
			},
			"templates": []interface{}{
				m.buildMergeTemplate(),
			},
		},
	}

	yamlData, err := yaml.Marshal(wf)
	if err != nil {
		return "", fmt.Errorf("failed to marshal merge workflow to YAML: %w", err)
	}
	return string(yamlData), nil
}

// Submit renders and submits this MergeWorkflow to Argo, returning the workflow name.
// namespace is required and determines the Kubernetes namespace for the workflow.
func (m *MergeWorkflow) Submit(namespace string) (string, error) {
	workflowYAML, err := m.Render()
	if err != nil {
		return "", err
	}
	return submitYAML(workflowYAML, m.RalphConfig, m.Watch, namespace)
}

func (m *MergeWorkflow) buildMergeTemplate() map[string]interface{} {
	return map[string]interface{}{
		"name": "ralph-merger",
		"container": map[string]interface{}{
			"image":   resolveImage(m.RalphConfig),
			"command": []string{"/bin/sh", "-c"},
			"args":    []string{buildMergeScript()},
			"env": []map[string]interface{}{
				{"name": "GIT_REPO_URL", "value": m.RepoURL},
				{"name": "GITHUB_REPO_OWNER", "value": m.RepoOwner},
				{"name": "GITHUB_REPO_NAME", "value": m.RepoName},
				{"name": "GIT_BRANCH", "value": m.CloneBranch},
				{"name": "PR_BRANCH", "value": m.PRBranch},
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
}
