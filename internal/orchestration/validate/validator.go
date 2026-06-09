package validate

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

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
	Remove(path string) error
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
			savePath := yamlPath(path)
			if err := v.project.Save(savePath, proj); err != nil {
				return nil, err
			}
			if savePath != path {
				if err := v.project.Remove(path); err != nil {
					return nil, err
				}
			}
			return proj, nil
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

func yamlPath(path string) string {
	if strings.ToLower(filepath.Ext(path)) == ".json" {
		return path[:len(path)-len(filepath.Ext(path))] + ".yaml"
	}
	return path
}
