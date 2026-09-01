package workflow

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/zon/ralph/internal/argo"
	"github.com/zon/ralph/internal/config"
	githubpkg "github.com/zon/ralph/internal/github"
	"gopkg.in/yaml.v3"
)

// Workflow holds all inputs required to generate and submit an Argo Workflow for a ralph run.
type Workflow struct {
	// ProjectName is used in the workflow's generateName field (e.g. "my-feature").
	ProjectName string
	// Repo is the GitHub repository.
	Repo githubpkg.Repo
	// CloneBranch is the branch the container clones initially (typically the base/current branch).
	CloneBranch string
	// ProjectBranch is the branch the container creates/checks-out to do its work.
	ProjectBranch string
	// ProjectPath is the relative path to the project YAML file inside the repo.
	ProjectPath string
	// Instructions is the contents of the instructions file to inject into the container (may be empty).
	Instructions string
	// Verbose controls whether the ralph command inside the container runs with --verbose.
	Verbose bool
	// DebugBranch, when non-empty, causes the workflow to checkout that branch of the ralph repo
	// into /workspace/ralph and invoke ralph via `go run` instead of the built binary.
	DebugBranch string
	// Image is the container image for the workflow.
	Image Image
	// ConfigMaps are the ConfigMaps to mount into the container.
	ConfigMaps []config.ConfigMapMount
	// Secrets are the Secrets to mount into the container.
	Secrets []config.SecretMount
	// Env is the environment variables to set in the container.
	Env map[string]config.EnvVar
	// KubeContext is the Argo workflow context label.
	KubeContext string
	// Namespace is the Kubernetes namespace for workflow submission.
	Namespace string
	// BaseBranch is the base branch for PR creation, already resolved by the caller.
	BaseBranch string
	// Items is the item query selecting the item array, already resolved by the
	// caller. The manifest always carries it as an explicit --items argument. An
	// empty value falls back to ".", so the container never re-resolves it from config.
	Items string
	// Cleanup reports whether the project file should be deleted once every item is complete.
	Cleanup bool
	// NoServices controls whether the ralph command inside the container runs with --no-services.
	NoServices bool
	// Model overrides the AI model from config.
	Model string
	// Agent overrides the opencode agent from config.
	Agent string
	// Variant overrides the model variant from config.
	Variant string
	// Labels are the Kubernetes labels to apply to the workflow pod.
	Labels map[string]string
	// Resources holds the CPU and memory requests and limits for the executor container.
	Resources config.WorkflowResources
	// Command is the command tokens to pass to `ralph workflow --command -- <tokens>`.
	Command []string
	// Loop carries the loop invocation the container runs when set. When set,
	// the container script calls `ralph workflow loop` instead of `ralph workflow run`.
	Loop *LoopSpec
}

// LoopSpec carries the loop invocation a loop workflow runs: the slug, steps,
// and maximum iterations.
type LoopSpec struct {
	Slug  string
	Steps []string
	Max   int
}

// Render produces the Argo Workflow YAML string for this Workflow.
func (w *Workflow) Render() (string, error) {
	params := map[string]string{
		"project-path":    w.ProjectPath,
		"instructions-md": w.Instructions,
		"base-branch":     w.BaseBranch,
	}

	wfLabels := map[string]string{
		"app.kubernetes.io/managed-by": "ralph",
	}
	for k, v := range w.Labels {
		wfLabels[k] = v
	}

	wf := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"metadata": map[string]interface{}{
			"generateName": fmt.Sprintf("ralph-%s-", w.ProjectName),
			"labels":       wfLabels,
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
// Malformed resource values are rejected here, before anything is handed to Argo.
func (w *Workflow) Submit(ctx context.Context, client argo.Client) (string, error) {
	if err := config.ValidateWorkflowResources(w.Resources); err != nil {
		return "", fmt.Errorf("invalid workflow resources: %w", err)
	}
	workflowYAML, err := w.Render()
	if err != nil {
		return "", err
	}
	return client.SubmitYAML(ctx, workflowYAML, argo.K8sContext{Name: w.KubeContext, Namespace: w.Namespace})
}

func (w *Workflow) buildMainTemplate() map[string]interface{} {
	var command []string
	var args []string

	switch {
	case len(w.Command) > 0:
		command = []string{"ralph"}
		args = []string{"workflow", "--command", "--"}
		args = append(args, w.Command...)
		if w.Verbose {
			args = append(args, "--verbose")
		}
		if w.Model != "" {
			args = append(args, "--model", w.Model)
		}
		if w.Agent != "" {
			args = append(args, "--agent", w.Agent)
		}

	case w.Loop != nil:
		command = []string{"ralph"}
		args = []string{"workflow", "loop"}
		if w.Loop.Slug != "" {
			args = append(args, "--slug", w.Loop.Slug)
		}
		args = append(args,
			"--repo", w.Repo.Owner+"/"+w.Repo.Name,
			"--clone-branch", w.CloneBranch,
			"--bot-name", config.DefaultAppName+"[bot]",
			"--bot-email", config.DefaultAppName+"[bot]@users.noreply.github.com",
		)
		if w.Loop.Max > 0 {
			args = append(args, "--max", strconv.Itoa(w.Loop.Max))
		}
		for _, step := range w.Loop.Steps {
			args = append(args, "--step", step)
		}
		if w.Verbose {
			args = append(args, "--verbose")
		}
		if w.Model != "" {
			args = append(args, "--model", w.Model)
		}
		if w.Variant != "" {
			args = append(args, "--variant", w.Variant)
		}
		if w.Agent != "" {
			args = append(args, "--agent", w.Agent)
		}

	default:
		command = []string{"ralph"}
		args = []string{
			"workflow", "run",
			"--repo", w.Repo.Owner + "/" + w.Repo.Name,
			"--project-path", "{{workflow.parameters.project-path}}",
			"--project-branch", w.ProjectBranch,
			"--base", w.BaseBranch,
		}
		if w.DebugBranch != "" {
			args = append(args, "--debug", w.DebugBranch)
		}
		itemsQuery := w.Items
		if itemsQuery == "" {
			itemsQuery = "."
		}
		args = append(args, "--items", itemsQuery)
		if w.Cleanup {
			args = append(args, "--cleanup")
		}
		if w.NoServices {
			args = append(args, "--no-services")
		}
		if w.Verbose {
			args = append(args, "--verbose")
		}
		if w.Model != "" {
			args = append(args, "--model", w.Model)
		}
		if w.Variant != "" {
			args = append(args, "--variant", w.Variant)
		}
		if w.Agent != "" {
			args = append(args, "--agent", w.Agent)
		}
	}

	container := map[string]interface{}{
		"image":        resolveImage(w.Image.Repository, w.Image.Tag),
		"command":      command,
		"args":         args,
		"env":          w.buildEnvVars(),
		"volumeMounts": buildVolumeMounts(w.ConfigMaps, w.Secrets),
		"workingDir":   "/workspace",
	}
	if resources := buildResources(w.Resources); len(resources) > 0 {
		container["resources"] = resources
	}

	template := map[string]interface{}{
		"name":      "ralph-executor",
		"container": container,
		"volumes":   buildVolumes(w.ConfigMaps, w.Secrets),
	}

	if len(w.Labels) > 0 {
		template["metadata"] = map[string]interface{}{
			"labels": w.Labels,
		}
	}

	return template
}

func (w *Workflow) buildEnvVars() []map[string]interface{} {
	envVars := []map[string]interface{}{
		{"name": "GIT_REPO_URL", "value": w.Repo.CloneURL()},
		{"name": "GITHUB_REPO_OWNER", "value": w.Repo.Owner},
		{"name": "GITHUB_REPO_NAME", "value": w.Repo.Name},
		{"name": "GIT_BRANCH", "value": w.CloneBranch},
		{"name": "PROJECT_BRANCH", "value": w.ProjectBranch},
		{"name": "PROJECT_PATH", "value": "{{workflow.parameters.project-path}}"},
		{"name": "INSTRUCTIONS_MD", "value": "{{workflow.parameters.instructions-md}}"},
		{"name": "RALPH_WORKFLOW_EXECUTION", "value": "true"},
		{"name": "RALPH_DEBUG_BRANCH", "value": w.DebugBranch},
		{"name": "RALPH_VERBOSE", "value": fmt.Sprintf("%t", w.Verbose)},
		{"name": "RALPH_NO_SERVICES", "value": fmt.Sprintf("%t", w.NoServices)},
	}

	for _, key := range slices.Sorted(maps.Keys(w.Env)) {
		envVar := w.Env[key]
		if envVar.SecretKeyRef != nil {
			envVars = append(envVars, map[string]interface{}{
				"name": key,
				"valueFrom": map[string]interface{}{
					"secretKeyRef": map[string]interface{}{
						"name": envVar.SecretKeyRef.Name,
						"key":  envVar.SecretKeyRef.Key,
					},
				},
			})
			continue
		}
		envVars = append(envVars, map[string]interface{}{
			"name":  key,
			"value": envVar.Value,
		})
	}

	return envVars
}
