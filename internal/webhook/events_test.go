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
// renderInstructions tests
// ──────────────────────────────────────────────────────────────────────────────

func TestRenderInstructions_SubstitutesCommentBody(t *testing.T) {
	tmpl := "Comment: {{.CommentBody}}"
	result := renderInstructions(tmpl, Event{Body: "please fix this"})
	assert.Equal(t, "Comment: please fix this", result)
}

func TestRenderInstructions_SubstitutesPRDetails(t *testing.T) {
	tmpl := "{{.RepoOwner}}/{{.RepoName}} #{{.PRNumber}} branch={{.PRBranch}}"
	result := renderInstructions(tmpl, Event{
		RepoOwner: "acme",
		RepoName:  "myrepo",
		PRNumber:  "42",
		PRBranch:  "ralph/my-feature",
	})
	assert.Equal(t, "acme/myrepo #42 branch=ralph/my-feature", result)
}

func TestRenderInstructions_DefaultTemplate_ContainsRequiredDirectives(t *testing.T) {
	result := renderInstructions(webhookconfig.DefaultCommentInstructions, Event{
		Body:      "do something",
		PRNumber:  "7",
		PRBranch:  "ralph/test",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	})
	assert.Contains(t, result, "do something")
	assert.Contains(t, result, "commit and push")
	assert.Contains(t, result, "GitHub PR comment")
	assert.Contains(t, result, "acme/myrepo")
	assert.Contains(t, result, "#7")
	assert.Contains(t, result, "ralph/test")
}

func TestRenderInstructions_InvalidTemplate_FallsBackToConcat(t *testing.T) {
	result := renderInstructions("{{.Invalid", Event{Body: "my comment"})
	assert.Contains(t, result, "my comment")
}

func TestRenderInstructions_DefaultMergeTemplate_ContainsRequiredDirectives(t *testing.T) {
	result := renderInstructions(webhookconfig.DefaultMergeInstructions, Event{
		PRNumber:  "12",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	})
	assert.Contains(t, result, "acme/myrepo")
	assert.Contains(t, result, "#12")
	assert.Contains(t, result, "ralph/my-feature")
	assert.Contains(t, result, "merge")
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
		App: webhookconfig.AppConfig{
			CommentInstructions: webhookconfig.DefaultCommentInstructions,
		},
	}
	e := Event{
		Body:      "please add a test",
		PRBranch:  "ralph/my-feature",
		RepoOwner: "acme",
		RepoName:  "myrepo",
	}

	result, err := e.ToWorkflow(cfg)
	require.NoError(t, err)
	require.NotNil(t, result.Run)
	assert.Nil(t, result.Merge)
	assert.Contains(t, result.Run.Instructions, "please add a test")
	assert.Equal(t, "acme", result.Run.RepoOwner)
	assert.Equal(t, "myrepo", result.Run.RepoName)
	assert.Equal(t, "ralph/my-feature", result.Run.CloneBranch)
}

func TestToWorkflow_ApprovalEvent_ReturnsMergeWorkflow(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{App: webhookconfig.AppConfig{
		CommentInstructions: webhookconfig.DefaultCommentInstructions,
		MergeInstructions:   webhookconfig.DefaultMergeInstructions,
	}}
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
	assert.Contains(t, result.Merge.Instructions, "acme/myrepo")
	assert.Contains(t, result.Merge.Instructions, "#99")
	assert.Contains(t, result.Merge.Instructions, "ralph/my-feature")
}

func TestToWorkflow_RunWorkflow_RendersToYAML(t *testing.T) {
	workflowTestDir(t)

	cfg := &webhookconfig.Config{App: webhookconfig.AppConfig{CommentInstructions: webhookconfig.DefaultCommentInstructions}}
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
			CommentInstructions: webhookconfig.DefaultCommentInstructions,
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
			CommentInstructions: webhookconfig.DefaultCommentInstructions,
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

	cfg := &webhookconfig.Config{App: webhookconfig.AppConfig{CommentInstructions: webhookconfig.DefaultCommentInstructions}}
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
