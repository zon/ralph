package validate

import (
	"bytes"
	"fmt"

	"github.com/zon/ralph/internal/project"
)

const MaxAttempts = 10

var ErrUnreachable = fmt.Errorf("unreachable: validate loop exited without returning")
var ErrNoChange = fmt.Errorf("agent made no changes to the project file")

type Validator struct {
	project ProjectClient
	agent   AgentClient
}

func (v *Validator) Validate(path string) (*project.Project, error) {
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		proj, loadErr := v.project.Load(path)
		if loadErr == nil {
			if saveErr := v.project.Save(path, proj); saveErr != nil {
				return nil, saveErr
			}
			return proj, nil
		}
		if attempt == MaxAttempts {
			return nil, loadErr
		}
		before, err := v.project.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if fixErr := v.agent.FixProject(path, loadErr); fixErr != nil {
			return nil, fixErr
		}
		after, err := v.project.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(before, after) {
			return nil, fmt.Errorf("%w: %w", ErrNoChange, loadErr)
		}
	}
	return nil, ErrUnreachable
}

type ProjectClient interface {
	Load(path string) (*project.Project, error)
	Save(path string, proj *project.Project) error
	ReadFile(path string) ([]byte, error)
}

type AgentClient interface {
	FixProject(path string, loadErr error) error
}