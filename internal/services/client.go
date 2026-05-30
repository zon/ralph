package services

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
)

type Client struct{}

func (a *Client) RunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := RunBefore(cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}

func (a *Client) Start(cfg *config.RalphConfig) (*Manager, error) {
	mgr := NewManager()
	if _, err := mgr.Start(cfg.Services); err != nil {
		return nil, err
	}
	return mgr, nil
}

func (a *Client) Stop(svc *Manager) {
	if svc != nil {
		svc.Stop()
	}
}

func (a *Client) RemoveLogs(cfg *config.RalphConfig) {
	for _, svc := range cfg.Services {
		logPath := LogFileName(svc.Name)
		os.Remove(logPath)
	}
}
