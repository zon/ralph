package cmd

import (
	"github.com/zon/ralph/internal/setskills"
)

type SetSkillsCmd struct {
	Branch string `help:"Branch to install skills from" default:"main"`
}

func (s *SetSkillsCmd) Run() error {
	return setskills.SetSkills(s.Branch)
}