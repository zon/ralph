package project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/config"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
)

func withItems(n int) *project.Project {
	return &project.Project{Items: project.NewItems(make([]any, n))}
}

func TestExtraIterationsDefaultTwentyPercent(t *testing.T) {
	cfg := &config.RalphConfig{}
	c := &project.Client{}
	assert.Equal(t, 2, c.ExtraIterations(withItems(10), cfg))
}

func TestExtraIterationsRoundsUp(t *testing.T) {
	cfg := &config.RalphConfig{}
	c := &project.Client{}
	assert.Equal(t, 1, c.ExtraIterations(withItems(3), cfg))
}

func TestExtraIterationsUsesConfigValue(t *testing.T) {
	v := 5
	cfg := &config.RalphConfig{ExtraIterations: &v}
	c := &project.Client{}
	assert.Equal(t, 5, c.ExtraIterations(withItems(10), cfg))
}

func TestProjectAdapterHasSpec(t *testing.T) {
	client := &project.Client{}

	t.Run("returns true when feature is set", func(t *testing.T) {
		proj := &project.Project{
			Slug:    "test",
			Feature: "specs/my-feature",
		}
		assert.True(t, client.HasSpec(proj))
	})

	t.Run("returns false when feature is empty", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
		}
		assert.False(t, client.HasSpec(proj))
	})
}

func TestProjectAdapterImplementsInterfaces(t *testing.T) {
	var _ orchestrationRun.ProjectClient = &project.Client{}
	var _ orchestrationRun.ProjectRepo = &project.Client{}
}
