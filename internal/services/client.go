package services

import (
	"fmt"
	"os"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/output"
)

type Client struct {
	out *output.Client
}

func NewClient(out *output.Client) *Client {
	return &Client{out: out}
}

func (a *Client) RunBeforeCommands(cfg *config.RalphConfig) error {
	if len(cfg.Before) > 0 {
		if err := RunBefore(a.out, cfg.Before); err != nil {
			return fmt.Errorf("failed to run before commands: %w", err)
		}
	}
	return nil
}

func (a *Client) Start(cfg *config.RalphConfig) (*Manager, error) {
	mgr := NewManager(a.out)
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
		// An invalid name never produced a log file, so skip it rather than
		// letting the path escape the repository root.
		logPath, err := safeLogPath(svc.Name)
		if err != nil {
			continue
		}
		os.Remove(logPath)
	}
}
