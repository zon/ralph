package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAny_HasOneFailingRequirement(t *testing.T) {
	proj := Any()
	assert.Equal(t, 1, len(proj.Requirements))
	assert.False(t, proj.Requirements[0].Passing)
}

func TestAny_HasMaxIterationsOfOne(t *testing.T) {
	proj := Any()
	assert.Equal(t, 1, proj.MaxIterations)
}

func TestAny_ReturnsValidProject(t *testing.T) {
	proj := Any()
	assert.NotNil(t, proj)
	assert.Equal(t, "test-project", proj.Slug)
}

func TestWithAllPassing_AllRequirementsPassing(t *testing.T) {
	proj := WithAllPassing()
	for _, req := range proj.Requirements {
		assert.True(t, req.Passing)
	}
}

func TestWithAllPassing_HasAtLeastOneRequirement(t *testing.T) {
	proj := WithAllPassing()
	assert.GreaterOrEqual(t, len(proj.Requirements), 1)
}

func TestWithFailingRequirements_HasAtLeastOneFailing(t *testing.T) {
	proj := WithFailingRequirements()
	found := false
	for _, req := range proj.Requirements {
		if !req.Passing {
			found = true
			break
		}
	}
	assert.True(t, found, "expected at least one failing requirement")
}

func TestWithMaxIterations_SetsMaxIterations(t *testing.T) {
	proj := WithMaxIterations(5)
	assert.Equal(t, 5, proj.MaxIterations)
}

func TestWithMaxIterations_Zero(t *testing.T) {
	proj := WithMaxIterations(0)
	assert.Equal(t, 0, proj.MaxIterations)
}

func TestWithMaxIterations_PreservesRequirements(t *testing.T) {
	proj := WithMaxIterations(3)
	assert.GreaterOrEqual(t, len(proj.Requirements), 1)
	assert.NotEmpty(t, proj.Requirements[0].Slug)
}
