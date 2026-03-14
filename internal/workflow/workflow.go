package workflow

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
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
	// CommentBody is the raw PR comment body for comment-triggered workflows.
	// When set, the container script calls `ralph comment` instead of `ralph run`.
	CommentBody string
	// PRNumber is the pull request number, used with CommentBody for ralph comment invocations.
	PRNumber string
	// Verbose controls whether the ralph command inside the container runs with --verbose.
	Verbose bool
	// DebugBranch, when non-empty, causes the workflow to checkout that branch of the ralph repo
	// into /workspace/ralph and invoke ralph via `go run` instead of the built binary.
	DebugBranch string
	// RalphConfig supplies workflow-level configuration (image overrides, secrets, configmaps, env).
	RalphConfig *config.RalphConfig
	// BaseBranch overrides the base branch for PR creation (overrides RalphConfig.DefaultBranch when set).
	BaseBranch string
}

// Render produces the Argo Workflow YAML string for this Workflow.
func (w *Workflow) Render() (string, error) {
	params := map[string]string{
		"project-path":    w.ProjectPath,
		"instructions-md": w.Instructions,
		"comment-body":    w.CommentBody,
		"pr-number":       w.PRNumber,
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
			"synchronization": map[string]interface{}{
				"mutexes": []interface{}{
					map[string]interface{}{
						"name": sanitizeName(w.ProjectBranch),
					},
				},
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
	return submitYAML(workflowYAML, w.RalphConfig, namespace)
}

// buildScript returns the appropriate shell script for this workflow type.
func (w *Workflow) buildScript() string {
	if w.CommentBody != "" {
		return buildCommentScript(w.Verbose)
	}
	return buildRunScript(w.Verbose, w.DebugBranch, w.RalphConfig)
}

func (w *Workflow) buildMainTemplate() map[string]interface{} {
	return map[string]interface{}{
		"name": "ralph-executor",
		"container": map[string]interface{}{
			"image":        resolveImage(w.RalphConfig),
			"command":      []string{"/bin/sh", "-c"},
			"args":         []string{w.buildScript()},
			"env":          w.buildEnvVars(),
			"volumeMounts": buildVolumeMounts(w.RalphConfig),
			"workingDir":   "/workspace",
		},
		"volumes": buildVolumes(w.RalphConfig),
	}
}

func (w *Workflow) buildEnvVars() []map[string]interface{} {
	baseBranch := w.BaseBranch
	if baseBranch == "" {
		baseBranch = w.RalphConfig.DefaultBranch
	}

	envVars := []map[string]interface{}{
		{"name": "GIT_REPO_URL", "value": w.RepoURL},
		{"name": "GITHUB_REPO_OWNER", "value": w.RepoOwner},
		{"name": "GITHUB_REPO_NAME", "value": w.RepoName},
		{"name": "GIT_BRANCH", "value": w.CloneBranch},
		{"name": "PROJECT_BRANCH", "value": w.ProjectBranch},
		{"name": "PROJECT_PATH", "value": "{{workflow.parameters.project-path}}"},
		{"name": "INSTRUCTIONS_MD", "value": "{{workflow.parameters.instructions-md}}"},
		{"name": "COMMENT_BODY", "value": "{{workflow.parameters.comment-body}}"},
		{"name": "PR_NUMBER", "value": "{{workflow.parameters.pr-number}}"},
		{"name": "RALPH_WORKFLOW_EXECUTION", "value": "true"},
		{"name": "BASE_BRANCH", "value": baseBranch},
	}

	hasPulumiToken := false
	if w.RalphConfig.Workflow.Env != nil {
		_, hasPulumiToken = w.RalphConfig.Workflow.Env["PULUMI_ACCESS_TOKEN"]
	}

	if !hasPulumiToken {
		envVars = append(envVars, map[string]interface{}{
			"name": "PULUMI_ACCESS_TOKEN",
			"valueFrom": map[string]interface{}{
				"secretKeyRef": map[string]interface{}{
					"name":     k8s.PulumiSecretName,
					"key":      "PULUMI_ACCESS_TOKEN",
					"optional": true,
				},
			},
		})
	}

	for key, value := range w.RalphConfig.Workflow.Env {
		envVars = append(envVars, map[string]interface{}{
			"name":  key,
			"value": value,
		})
	}

	return envVars
}
