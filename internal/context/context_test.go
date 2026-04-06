package context

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldNotify(t *testing.T) {
	tests := []struct {
		name         string
		noNotify     bool
		local        bool
		follow       bool
		expectNotify bool
		description  string
	}{
		{
			name:         "default settings should not notify (remote workflow without follow)",
			noNotify:     false,
			local:        false,
			follow:       false,
			expectNotify: false,
			description:  "remote workflow without follow should not notify",
		},
		{
			name:         "local mode notifies by default",
			noNotify:     false,
			local:        true,
			follow:       false,
			expectNotify: true,
			description:  "local mode with notifications enabled",
		},
		{
			name:         "no-notify flag disables notifications",
			noNotify:     true,
			local:        true,
			follow:       false,
			expectNotify: false,
			description:  "user explicitly disabled notifications",
		},
		{
			name:         "remote workflow with follow enables notifications",
			noNotify:     false,
			local:        false,
			follow:       true,
			expectNotify: true,
			description:  "remote workflow with follow should notify",
		},
		{
			name:         "remote with follow but no-notify flag disables notifications",
			noNotify:     true,
			local:        false,
			follow:       true,
			expectNotify: false,
			description:  "explicit no-notify flag overrides follow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetNoNotify(tt.noNotify)
			ctx.SetLocal(tt.local)
			ctx.SetFollow(tt.follow)

			result := ctx.ShouldNotify()
			assert.Equal(t, tt.expectNotify, result, tt.description)
		})
	}
}

func TestIsLocal(t *testing.T) {
	tests := []struct {
		name  string
		local bool
	}{
		{
			name:  "local mode enabled",
			local: true,
		},
		{
			name:  "local mode disabled",
			local: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetLocal(tt.local)

			result := ctx.IsLocal()
			assert.Equal(t, tt.local, result, "IsLocal should match the set value")
		})
	}
}

func TestShouldFollow(t *testing.T) {
	tests := []struct {
		name   string
		follow bool
	}{
		{
			name:   "follow mode enabled",
			follow: true,
		},
		{
			name:   "follow mode disabled",
			follow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetFollow(tt.follow)

			result := ctx.ShouldFollow()
			assert.Equal(t, tt.follow, result, "ShouldFollow should match the set value")
		})
	}
}

func TestIsWorkflowExecution(t *testing.T) {
	tests := []struct {
		name              string
		workflowExecution bool
	}{
		{
			name:              "workflow execution enabled",
			workflowExecution: true,
		},
		{
			name:              "workflow execution disabled",
			workflowExecution: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetWorkflowExecution(tt.workflowExecution)

			result := ctx.IsWorkflowExecution()
			assert.Equal(t, tt.workflowExecution, result, "IsWorkflowExecution should match the set value")
		})
	}
}

func TestAddNote(t *testing.T) {
	ctx := &Context{}

	assert.False(t, ctx.HasNotes(), "New context should not have notes")

	ctx.AddNote("First note")
	assert.True(t, ctx.HasNotes(), "Context should have notes after adding one")
	assert.Len(t, ctx.Notes(), 1, "Should have 1 note")
	assert.Equal(t, "First note", ctx.Notes()[0], "First note should match")

	ctx.AddNote("Second note")
	assert.Len(t, ctx.Notes(), 2, "Should have 2 notes")
	assert.Equal(t, "Second note", ctx.Notes()[1], "Second note should match")
}

func TestHasNotes(t *testing.T) {
	tests := []struct {
		name      string
		notes     []string
		expectHas bool
	}{
		{
			name:      "no notes",
			notes:     nil,
			expectHas: false,
		},
		{
			name:      "empty slice",
			notes:     []string{},
			expectHas: false,
		},
		{
			name:      "one note",
			notes:     []string{"note"},
			expectHas: true,
		},
		{
			name:      "multiple notes",
			notes:     []string{"note1", "note2"},
			expectHas: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			for _, note := range tt.notes {
				ctx.AddNote(note)
			}

			result := ctx.HasNotes()
			assert.Equal(t, tt.expectHas, result, "HasNotes should match expected value")
		})
	}
}

func TestBaseBranch(t *testing.T) {
	tests := []struct {
		name          string
		baseBranch    string
		expectDefault bool
	}{
		{
			name:          "default empty base branch",
			baseBranch:    "",
			expectDefault: true,
		},
		{
			name:          "custom base branch",
			baseBranch:    "develop",
			expectDefault: false,
		},
		{
			name:          "main branch",
			baseBranch:    "main",
			expectDefault: false,
		},
		{
			name:          "feature branch",
			baseBranch:    "feature/my-feature",
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetBaseBranch(tt.baseBranch)

			result := ctx.BaseBranch()
			assert.Equal(t, tt.baseBranch, result, "BaseBranch should match the set value")

			if tt.expectDefault {
				assert.Empty(t, result, "Default base branch should be empty")
			} else {
				assert.NotEmpty(t, result, "Custom base branch should not be empty")
			}
		})
	}
}

func TestRepoOwnerAndName(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		repoOwner   string
		repoName    string
		expectOwner string
		expectName  string
	}{
		{
			name:        "SetRepo populates owner and name",
			repo:        "owner/repo",
			expectOwner: "owner",
			expectName:  "repo",
		},
		{
			name:        "SetRepoOwner and SetRepoName populate fields",
			repoOwner:   "field-owner",
			repoName:    "field-repo",
			expectOwner: "field-owner",
			expectName:  "field-repo",
		},
		{
			name:        "SetRepo overrides previous SetRepoOwner/Name",
			repoOwner:   "old-owner",
			repoName:    "old-repo",
			repo:        "new-owner/new-repo",
			expectOwner: "new-owner",
			expectName:  "new-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			if tt.repoOwner != "" {
				ctx.SetRepoOwner(tt.repoOwner)
			}
			if tt.repoName != "" {
				ctx.SetRepoName(tt.repoName)
			}
			if tt.repo != "" {
				ctx.SetRepo(tt.repo)
			}

			owner, name := ctx.RepoOwnerAndName()
			assert.Equal(t, tt.expectOwner, owner)
			assert.Equal(t, tt.expectName, name)
		})
	}
}

func TestKubeContext(t *testing.T) {
	tests := []struct {
		name          string
		kubeContext   string
		expectDefault bool
	}{
		{
			name:          "default empty kube context",
			kubeContext:   "",
			expectDefault: true,
		},
		{
			name:          "custom kube context",
			kubeContext:   "my-cluster-context",
			expectDefault: false,
		},
		{
			name:          "minikube context",
			kubeContext:   "minikube",
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetKubeContext(tt.kubeContext)

			result := ctx.KubeContext()
			assert.Equal(t, tt.kubeContext, result, "KubeContext should match the set value")

			if tt.expectDefault {
				assert.Empty(t, result, "Default kube context should be empty")
			} else {
				assert.NotEmpty(t, result, "Custom kube context should not be empty")
			}
		})
	}
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name          string
		filter        string
		expectDefault bool
	}{
		{
			name:          "default empty filter",
			filter:        "",
			expectDefault: true,
		},
		{
			name:          "custom filter",
			filter:        "my-filter-string",
			expectDefault: false,
		},
		{
			name:          "filter with special characters",
			filter:        "filter-with-dashes_and_underscores",
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetFilter(tt.filter)

			result := ctx.Filter()
			assert.Equal(t, tt.filter, result, "Filter should match the set value")

			if tt.expectDefault {
				assert.Empty(t, result, "Default filter should be empty")
			} else {
				assert.NotEmpty(t, result, "Custom filter should not be empty")
			}
		})
	}
}

func TestNewContextFromEnv(t *testing.T) {
	envVars := map[string]string{
		"RALPH_WORKFLOW_EXECUTION": "true",
		"GITHUB_REPO_OWNER":        "test-owner",
		"GITHUB_REPO_NAME":         "test-repo",
		"PROJECT_PATH":             "/path/to/project.yaml",
		"PROJECT_BRANCH":           "feature/test",
		"BASE_BRANCH":              "main",
		"RALPH_VERBOSE":            "true",
		"RALPH_NO_SERVICES":        "true",
		"RALPH_DEBUG_BRANCH":       "debug-branch",
		"INSTRUCTIONS_MD":          "# Test Instructions",
		"RALPH_MAX_ITERATIONS":     "42",
	}

	for key, val := range envVars {
		os.Setenv(key, val)
		defer os.Unsetenv(key)
	}

	ctx := NewContextFromEnv()

	assert.True(t, ctx.IsWorkflowExecution(), "workflowExecution should be true")
	owner, name := ctx.RepoOwnerAndName()
	assert.Equal(t, "test-owner", owner, "repo owner should match")
	assert.Equal(t, "test-repo", name, "repo name should match")
	assert.Equal(t, "/path/to/project.yaml", ctx.ProjectFile(), "project file should match")
	assert.Equal(t, "feature/test", ctx.Branch(), "branch should match")
	assert.Equal(t, "main", ctx.BaseBranch(), "base branch should match")
	assert.True(t, ctx.IsVerbose(), "verbose should be true")
	assert.True(t, ctx.NoServices(), "noServices should be true")
	assert.Equal(t, "debug-branch", ctx.DebugBranch(), "debug branch should match")
	assert.Equal(t, "# Test Instructions", ctx.InstructionsMD(), "instructionsMD should match")
	assert.Equal(t, 42, ctx.MaxIterations(), "maxIterations should match")
}

func TestNewContextFromEnvEmpty(t *testing.T) {
	envVars := []string{
		"RALPH_WORKFLOW_EXECUTION",
		"GITHUB_REPO_OWNER",
		"GITHUB_REPO_NAME",
		"PROJECT_PATH",
		"PROJECT_BRANCH",
		"BASE_BRANCH",
		"RALPH_VERBOSE",
		"RALPH_NO_SERVICES",
		"RALPH_DEBUG_BRANCH",
		"INSTRUCTIONS_MD",
		"RALPH_MAX_ITERATIONS",
	}

	for _, key := range envVars {
		os.Unsetenv(key)
	}

	ctx := NewContextFromEnv()

	assert.False(t, ctx.IsWorkflowExecution(), "workflowExecution should be false by default")
	owner, name := ctx.RepoOwnerAndName()
	assert.Empty(t, owner, "repo owner should be empty")
	assert.Empty(t, name, "repo name should be empty")
	assert.Empty(t, ctx.ProjectFile(), "project file should be empty")
	assert.Empty(t, ctx.Branch(), "branch should be empty")
	assert.Empty(t, ctx.BaseBranch(), "base branch should be empty")
	assert.False(t, ctx.IsVerbose(), "verbose should be false")
	assert.False(t, ctx.NoServices(), "noServices should be false")
	assert.Empty(t, ctx.DebugBranch(), "debug branch should be empty")
	assert.Empty(t, ctx.InstructionsMD(), "instructionsMD should be empty")
	assert.Equal(t, 0, ctx.MaxIterations(), "maxIterations should be 0 by default")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name         string
		workflowExec bool
		local        bool
		expectError  bool
	}{
		{
			name:         "both false is valid",
			workflowExec: false,
			local:        false,
			expectError:  false,
		},
		{
			name:         "workflowExecution true, local false is valid",
			workflowExec: true,
			local:        false,
			expectError:  false,
		},
		{
			name:         "workflowExecution false, local true is valid",
			workflowExec: false,
			local:        true,
			expectError:  false,
		},
		{
			name:         "both true is invalid",
			workflowExec: true,
			local:        true,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &Context{}
			ctx.SetWorkflowExecution(tt.workflowExec)
			ctx.SetLocal(tt.local)

			err := ctx.Validate()
			if tt.expectError {
				assert.Error(t, err, "Validate should return error when both workflowExecution and local are true")
				assert.Contains(t, err.Error(), "workflowExecution and local cannot both be true")
			} else {
				assert.NoError(t, err, "Validate should not return error")
			}
		})
	}
}
