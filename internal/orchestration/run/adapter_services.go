package run

import (
	"fmt"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/services"
)

// ServicesClientAdapter adapts services functions to the ServicesClient interface.
type ServicesClientAdapter struct{}

func (a *ServicesClientAdapter) RunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := services.RunBefore(cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}
