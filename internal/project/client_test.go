package project_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/config"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
)

func TestProjectAdapterAllRequirementsPassing(t *testing.T) {
	client := &project.Client{}

	t.Run("returns true when all requirements pass", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
			Requirements: []project.Requirement{
				{Slug: "req-1", Items: []string{"a"}, Passing: true},
				{Slug: "req-2", Items: []string{"b"}, Passing: true},
			},
		}
		assert.True(t, client.AllRequirementsPassing(proj))
	})

	t.Run("returns false when some requirements fail", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
			Requirements: []project.Requirement{
				{Slug: "req-1", Items: []string{"a"}, Passing: true},
				{Slug: "req-2", Items: []string{"b"}, Passing: false},
			},
		}
		assert.False(t, client.AllRequirementsPassing(proj))
	})

	t.Run("returns false when all requirements fail", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
			Requirements: []project.Requirement{
				{Slug: "req-1", Items: []string{"a"}, Passing: false},
				{Slug: "req-2", Items: []string{"b"}, Passing: false},
			},
		}
		assert.False(t, client.AllRequirementsPassing(proj))
	})
}

func TestExtraIterationsDefaultTwentyPercent(t *testing.T) {
	cfg := &config.RalphConfig{}
	proj := &project.Project{
		Requirements: make([]project.Requirement, 10),
	}
	c := &project.Client{}
	extra := c.ExtraIterations(proj, cfg)
	assert.Equal(t, 2, extra)
}

func TestExtraIterationsRoundsUp(t *testing.T) {
	cfg := &config.RalphConfig{}
	proj := &project.Project{
		Requirements: make([]project.Requirement, 3),
	}
	c := &project.Client{}
	extra := c.ExtraIterations(proj, cfg)
	assert.Equal(t, 1, extra)
}

func TestExtraIterationsUsesConfigValue(t *testing.T) {
	v := 5
	cfg := &config.RalphConfig{ExtraIterations: &v}
	proj := &project.Project{
		Requirements: make([]project.Requirement, 10),
	}
	c := &project.Client{}
	extra := c.ExtraIterations(proj, cfg)
	assert.Equal(t, 5, extra)
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
