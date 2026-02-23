package webhook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/webhookconfig"
)

// ──────────────────────────────────────────────────────────────────────────────
// projectFileFromBranch tests
// ──────────────────────────────────────────────────────────────────────────────

func TestProjectFileFromBranch(t *testing.T) {
	tests := []struct {
		branch string
		want   string
	}{
		{"ralph/my-feature", "projects/my-feature.yaml"},
		{"ralph/github-webhook-service", "projects/github-webhook-service.yaml"},
		{"feature/something", "projects/feature-something.yaml"},
		{"", "projects/.yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.branch, func(t *testing.T) {
			assert.Equal(t, tc.want, projectFileFromBranch(tc.branch))
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// ToWorkflow tests
// ──────────────────────────────────────────────────────────────────────────────

// workflowTestDir sets up a temp dir with the minimal .ralph/config.yaml that
// GenerateWorkflowWithGitInfo requires, changes into it for the test, and
// restores the original working directory on cleanup.
func workflowTestDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	ralphDir := filepath.Join(tmp, ".ralph")
	require.NoError(t, os.MkdirAll(ralphDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ralphDir, "config.yaml"), []byte("maxIterations: 3\n"), 0644))

	orig, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestToWorkflow_CommentEvent_ReturnsRunWorkflow(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{
		App: webhookconfig.AppConfig{},
	}
	e := Event{
		Body:      "please add a test",
		PRBranch:  "ralph/my-feature",
		PRNumber:  "5",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)
	require.NotNil(t, result.Run)
	assert.Nil(t, result.Merge)
	assert.Equal(t, "please add a test", result.Run.CommentBody)
	assert.Equal(t, "5", result.Run.PRNumber)
	assert.Equal(t, "acme", result.Run.RepoOwner)
	assert.Equal(t, "myrepo", result.Run.RepoName)
	assert.Equal(t, "ralph/my-feature", result.Run.CloneBranch)
}

func TestToWorkflow_ApprovalEvent_ReturnsMergeWorkflow(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{App: webhookconfig.AppConfig{}}
	e := Event{
		Approved:  true,
		PRBranch:  "ralph/my-feature",
		PRNumber:  "99",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)
	require.NotNil(t, result.Merge)
	assert.Nil(t, result.Run)
	assert.Equal(t, "acme", result.Merge.RepoOwner)
	assert.Equal(t, "myrepo", result.Merge.RepoName)
	assert.Equal(t, "ralph/my-feature", result.Merge.PRBranch)
	assert.Equal(t, "99", result.Merge.PRNumber)
}

func TestToWorkflow_RunWorkflow_RendersToYAML(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{App: webhookconfig.AppConfig{}}
	e := Event{
		Body:      "fix the bug",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)

	yaml, err := result.Run.Render()
	require.NoError(t, err)
	assert.Contains(t, yaml, "argoproj.io/v1alpha1")
	assert.Contains(t, yaml, "acme")
	assert.Contains(t, yaml, "myrepo")
	assert.Contains(t, yaml, "ralph/my-feature")
}

func TestToWorkflow_NamespacePropagated(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{
		App: webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "myrepo", Namespace: "team-ns"},
			},
		},
	}
	e := Event{
		Body:      "fix something",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)
	assert.Equal(t, "team-ns", result.Namespace)
}

func TestToWorkflow_Namespace_EmptyWhenNotConfigured(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{
		App: webhookconfig.AppConfig{
			Repos: []webhookconfig.RepoConfig{
				{Owner: "acme", Name: "myrepo"},
			},
		},
	}
	e := Event{
		Body:      "fix something",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)
	assert.Equal(t, "", result.Namespace)
}

func TestToWorkflow_MergeWorkflow_RendersToYAML(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{App: webhookconfig.AppConfig{}}
	e := Event{
		Approved:  true,
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)

	yaml, err := result.Merge.Render()
	require.NoError(t, err)
	assert.Contains(t, yaml, "argoproj.io/v1alpha1")
	assert.Contains(t, yaml, "ralph-merge-")
}
