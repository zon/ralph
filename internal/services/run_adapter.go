package services

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
)

type RunAdapter struct{}

func (a *RunAdapter) RunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := RunBefore(cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}
