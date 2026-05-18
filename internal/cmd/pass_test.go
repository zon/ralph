package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/project"
)

func TestPassCmd_MarkPassing(t *testing.T) {
	path := project.FileWithRequirement(t, "my-req", false)
	cmd := &PassCmd{ProjectFile: path, Slug: "my-req"}
	require.NoError(t, cmd.Run())
	assert.True(t, project.RequirementStatus(t, path, "my-req"))
}

func TestPassCmd_MarkFailing(t *testing.T) {
	path := project.FileWithRequirement(t, "my-req", true)
	cmd := &PassCmd{ProjectFile: path, Slug: "my-req", False: true}
	require.NoError(t, cmd.Run())
	assert.False(t, project.RequirementStatus(t, path, "my-req"))
}

func TestPassCmd_FileNotFound(t *testing.T) {
	cmd := &PassCmd{ProjectFile: project.NonExistentFile(t), Slug: "my-req"}
	assert.Error(t, cmd.Run())
}

func TestPassCmd_SlugNotFound(t *testing.T) {
	path := project.FileWithRequirement(t, "my-req", false)
	cmd := &PassCmd{ProjectFile: path, Slug: "unknown-slug"}
	assert.Error(t, cmd.Run())
}