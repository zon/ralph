package project

import "fmt"

type Client struct{}

func (a *Client) AllRequirementsPassing(proj *Project) bool {
	allComplete, _, _ := CheckCompletion(proj)
	return allComplete
}

func (a *Client) MaxIterationsError(proj *Project) error {
	_, _, failingCount := CheckCompletion(proj)
	return fmt.Errorf("%w: %d requirements still failing", ErrMaxIterationsReached, failingCount)
}
