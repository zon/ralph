package workflow

import (
	"fmt"

	githubpkg "github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"gopkg.in/yaml.v3"
)

// MergeWorkflow holds all inputs required to generate and submit an Argo Workflow for a ralph merge.
type MergeWorkflow struct {
	// Repo is the GitHub repository.
	Repo githubpkg.Repo
	// CloneBranch is the branch the container clones initially (typically the base branch).
	CloneBranch string
	// PRBranch is the PR branch to merge.
	PRBranch string
	// PRNumber is the pull request number, passed to ralph merge --local.
	PRNumber string
	// Image is the container image for the workflow.
	Image Image
	// KubeContext is the Argo workflow context label.
	KubeContext string
	// Namespace is the Kubernetes namespace for workflow submission.
	Namespace string
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
			"synchronization": map[string]interface{}{
				"mutexes": []interface{}{
					map[string]interface{}{
						"name": sanitizeName(m.PRBranch),
					},
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
func (m *MergeWorkflow) Submit() (string, error) {
	workflowYAML, err := m.Render()
	if err != nil {
		return "", err
	}
	return submitYAML(workflowYAML, m.KubeContext, m.Namespace)
}

func (m *MergeWorkflow) buildMergeTemplate() map[string]interface{} {
	return map[string]interface{}{
		"name": "ralph-merger",
		"container": map[string]interface{}{
			"image":   resolveImage(m.Image.Repository, m.Image.Tag),
			"command": []string{"/bin/sh", "-c"},
			"args":    []string{buildMergeScript()},
			"env": []map[string]interface{}{
				{"name": "GIT_REPO_URL", "value": m.Repo.CloneURL()},
				{"name": "GITHUB_REPO_OWNER", "value": m.Repo.Owner},
				{"name": "GITHUB_REPO_NAME", "value": m.Repo.Name},
				{"name": "GIT_BRANCH", "value": m.CloneBranch},
				{"name": "PR_BRANCH", "value": m.PRBranch},
				{"name": "PR_NUMBER", "value": m.PRNumber},
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
