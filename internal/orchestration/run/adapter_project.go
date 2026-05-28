package run

import (
	"fmt"

	"github.com/zon/ralph/internal/project"
)

// ProjectClientAdapter adapts project.Project functions to the ProjectClient interface.
type ProjectClientAdapter struct{}

func (a *ProjectClientAdapter) AllRequirementsPassing(proj *project.Project) bool {
	allComplete, _, _ := project.CheckCompletion(proj)
	return allComplete
}

func (a *ProjectClientAdapter) MaxIterationsError(proj *project.Project) error {
	_, _, failingCount := project.CheckCompletion(proj)
	return fmt.Errorf("%w: %d requirements still failing", project.ErrMaxIterationsReached, failingCount)
}
