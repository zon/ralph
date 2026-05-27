package cmd

import "fmt"

type CommandFlags struct {
	Follow bool
	Local  bool
	Debug  string
}

func (f CommandFlags) Validate() error {
	if f.Follow && f.Local {
		return fmt.Errorf("--follow flag is not applicable with --local flag")
	}

	if f.Debug != "" && f.Local {
		return fmt.Errorf("--debug flag is not applicable with --local flag")
	}
	return nil
}