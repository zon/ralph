package validate

import (
	"bytes"
	"fmt"

	"github.com/zon/ralph/internal/project"
)

const MaxAttempts = 10

var (
	ErrNoChange    = fmt.Errorf("agent made no changes to the project file")
	ErrUnreachable = fmt.Errorf("unreachable: validate loop exited without returning")
)

type ProjectClient interface {
	Load(path string) (*project.Project, error)
	Save(path string, proj *project.Project) error
	ReadFile(path string) ([]byte, error)
}

type AgentClient interface {
	FixProject(path string, loadErr error, model string) error
}

type Validator struct {
	project ProjectClient
	agent   AgentClient
	model   string
}

func (v *Validator) Validate(path string) (*project.Project, error) {
	for attempt := 1; attempt <= MaxAttempts; attempt++ {
		proj, loadErr := v.project.Load(path)
		if loadErr == nil {
			return proj, v.project.Save(path, proj)
		}
		if attempt == MaxAttempts {
			return nil, loadErr
		}
		before, _ := v.project.ReadFile(path)
		v.agent.FixProject(path, loadErr, v.model)
		after, _ := v.project.ReadFile(path)
		if bytes.Equal(before, after) {
			return nil, ErrNoChange
		}
	}
	return nil, ErrUnreachable
}
