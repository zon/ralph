package project

import "fmt"

type RunAdapter struct{}

func (a *RunAdapter) AllRequirementsPassing(proj *Project) bool {
	allComplete, _, _ := CheckCompletion(proj)
	return allComplete
}

func (a *RunAdapter) MaxIterationsError(proj *Project) error {
	_, _, failingCount := CheckCompletion(proj)
	return fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
}
