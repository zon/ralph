package pass

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/project"
)

func TestPassCmd_MarkPassing(t *testing.T) {
	path := project.FileWithRequirement(t, "my-req", false)
	cmd := &PassCmd{ProjectFile: path, Slug: "my-req"}
	proj, err := cmd.Run()
	require.NoError(t, err)
	require.NotNil(t, proj)
	assert.True(t, project.RequirementStatus(t, path, "my-req"))
}

func TestPassCmd_MarkFailing(t *testing.T) {
	path := project.FileWithRequirement(t, "my-req", true)
	cmd := &PassCmd{ProjectFile: path, Slug: "my-req", False: true}
	proj, err := cmd.Run()
	require.NoError(t, err)
	require.NotNil(t, proj)
	assert.False(t, project.RequirementStatus(t, path, "my-req"))
}

func TestPassCmd_FileNotFound(t *testing.T) {
	cmd := &PassCmd{ProjectFile: project.NonExistentFile(t), Slug: "my-req"}
	proj, err := cmd.Run()
	assert.Error(t, err)
	assert.Nil(t, proj)
}

func TestPassCmd_SlugNotFound(t *testing.T) {
	path := project.FileWithRequirement(t, "my-req", false)
	cmd := &PassCmd{ProjectFile: path, Slug: "unknown-slug"}
	proj, err := cmd.Run()
	assert.Error(t, err)
	assert.Nil(t, proj)
}
