package context

import (
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
