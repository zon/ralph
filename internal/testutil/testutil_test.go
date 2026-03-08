package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	assert.Equal(t, 10, ctx.MaxIterations(), "MaxIterations should be 10 by default")
	assert.False(t, ctx.IsVerbose(), "IsVerbose should be false by default")
	assert.False(t, ctx.ShouldNotify(), "ShouldNotify should be false when NoNotify=true")
	assert.True(t, ctx.ShouldStartServices(), "ShouldStartServices should be true when NoServices=false")
	assert.Empty(t, ctx.ProjectFile(), "ProjectFile should be empty by default")
}

func TestNewContext_WithOptions(t *testing.T) {
	ctx := NewContext(
		WithProjectFile("/path/to/project.yaml"),
		WithMaxIterations(5),
		WithVerbose(true),
	)

	assert.Equal(t, "/path/to/project.yaml", ctx.ProjectFile(), "ProjectFile should match")
	assert.Equal(t, 5, ctx.MaxIterations(), "MaxIterations should be 5")
	assert.True(t, ctx.IsVerbose(), "IsVerbose should be true")
	assert.False(t, ctx.ShouldNotify(), "ShouldNotify should be false (default NoNotify=true)")
}

func TestNewContext_MultipleOptions(t *testing.T) {
	ctx := NewContext(
		WithMaxIterations(20),
		WithNoServices(true),
	)

	assert.Equal(t, 20, ctx.MaxIterations(), "MaxIterations should be 20")
	assert.False(t, ctx.ShouldStartServices(), "ShouldStartServices should be false when NoServices=true")
}
