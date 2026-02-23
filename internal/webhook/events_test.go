package webhook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	internalconfig "github.com/zon/ralph/internal/config"
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
	result := renderInstructions(tmpl, "please fix this")
	assert.Equal(t, "Comment: please fix this", result)
}

func TestRenderInstructions_DefaultTemplate_ContainsRequiredDirectives(t *testing.T) {
	result := renderInstructions(internalconfig.DefaultCommentInstructions, "do something")
	assert.Contains(t, result, "do something")
	assert.Contains(t, result, "commit and push")
	assert.Contains(t, result, "GitHub PR comment")
}

func TestRenderInstructions_InvalidTemplate_FallsBackToConcat(t *testing.T) {
	result := renderInstructions("{{.Invalid", "my comment")
	assert.Contains(t, result, "my comment")
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

	cfg := &Config{
		App: AppConfig{
			CommentInstructions: internalconfig.DefaultCommentInstructions,
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

	cfg := &Config{App: AppConfig{CommentInstructions: internalconfig.DefaultCommentInstructions}}
	e := Event{
		Approved:  true,
		PRBranch:  "ralph/my-feature",
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
}

func TestToWorkflow_RunWorkflow_RendersToYAML(t *testing.T) {
	workflowTestDir(t)

	cfg := &Config{App: AppConfig{CommentInstructions: internalconfig.DefaultCommentInstructions}}
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

func TestToWorkflow_MergeWorkflow_RendersToYAML(t *testing.T) {
	workflowTestDir(t)

	cfg := &Config{App: AppConfig{CommentInstructions: internalconfig.DefaultCommentInstructions}}
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
