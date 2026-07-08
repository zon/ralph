package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	assert.False(t, ctx.IsVerbose(), "IsVerbose should be false by default")
	assert.True(t, ctx.NoNotify(), "NoNotify should be true by default")
	assert.False(t, ctx.NoServices(), "NoServices should be false when NoServices=false")
	assert.Empty(t, ctx.ProjectFile(), "ProjectFile should be empty by default")
}

func TestNewContext_WithOptions(t *testing.T) {
	ctx := NewContext(
		WithProjectFile("/path/to/project.yaml"),
		WithVerbose(true),
	)

	assert.Equal(t, "/path/to/project.yaml", ctx.ProjectFile(), "ProjectFile should match")
	assert.True(t, ctx.IsVerbose(), "IsVerbose should be true")
	assert.True(t, ctx.NoNotify(), "NoNotify should be true (default)")
}

func TestNewContext_MultipleOptions(t *testing.T) {
	ctx := NewContext(
		WithNoServices(true),
	)

	assert.True(t, ctx.NoServices(), "NoServices should be true when NoServices=true")
}
