package validate

import (
	"github.com/zon/ralph/internal/project"
)

type Validator interface {
	Validate(path string) (*project.Project, error)
}

type ValidateCmd struct {
	validator Validator
}

func New(validator Validator) *ValidateCmd {
	return &ValidateCmd{validator: validator}
}

func (v *ValidateCmd) Run(path string) (*project.Project, error) {
	return v.validator.Validate(path)
}
