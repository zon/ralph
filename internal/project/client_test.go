package project_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
)

func TestProjectClientAllRequirementsPassing(t *testing.T) {
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

func TestProjectClientMaxIterationsError(t *testing.T) {
	client := &project.Client{}

	t.Run("returns error wrapping ErrMaxIterationsReached", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
			Requirements: []project.Requirement{
				{Slug: "req-1", Items: []string{"a"}, Passing: false},
			},
		}
		err := client.MaxIterationsError(proj)
		require.Error(t, err)
		assert.True(t, errors.Is(err, project.ErrMaxIterationsReached))
	})

	t.Run("includes count of failing requirements", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
			Requirements: []project.Requirement{
				{Slug: "req-1", Items: []string{"a"}, Passing: false},
				{Slug: "req-2", Items: []string{"b"}, Passing: true},
				{Slug: "req-3", Items: []string{"c"}, Passing: false},
			},
		}
		err := client.MaxIterationsError(proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "2 requirements still failing")
	})

	t.Run("reports 0 when all requirements pass", func(t *testing.T) {
		proj := &project.Project{
			Slug: "test",
			Requirements: []project.Requirement{
				{Slug: "req-1", Items: []string{"a"}, Passing: true},
			},
		}
		err := client.MaxIterationsError(proj)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "0 requirements still failing")
	})
}

func TestProjectClientImplementsInterface(t *testing.T) {
	var _ orchestrationRun.ProjectClient = &project.Client{}
}
