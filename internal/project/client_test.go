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

func TestExtraIterationsDefaultThirtyPercent(t *testing.T) {
	cfg := &config.RalphConfig{}
	c := &project.Client{}
	assert.Equal(t, 3, c.ExtraIterations(withItems(10), cfg))
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

func TestProjectAdapterImplementsInterfaces(t *testing.T) {
	var _ orchestrationRun.ProjectClient = &project.Client{}
	var _ orchestrationRun.ProjectRepo = &project.Client{}
}
