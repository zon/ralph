package main

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/zon/ralph/internal/output"
)

type Manager struct {
	mu        sync.Mutex
	cleanupFn []func()
	exitFn    func(int)
	stopFn    func(chan<- os.Signal)
	out       *output.Client
}

func NewManager(out *output.Client) *Manager {
	return &Manager{
		cleanupFn: make([]func(), 0),
		exitFn:    os.Exit,
		stopFn:    signal.Stop,
		out:       out,
	}
}

func (m *Manager) RegisterCleanup(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupFn = append(m.cleanupFn, fn)
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	fns := m.cleanupFn
	m.cleanupFn = nil
	m.mu.Unlock()

	for i := len(fns) - 1; i >= 0; i-- {
		fns[i]()
	}
}

func (m *Manager) SetupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go m.handleSignal(sigChan)
}

func (m *Manager) handleSignal(sigChan chan os.Signal) {
	sig := <-sigChan
	// Stop the registration so a second signal takes its default action and
	// terminates the process even if cleanup is still running.
	m.stopFn(sigChan)
	m.out.Infof("Received signal: %v", sig)
	m.out.Info("Cleaning up...")
	m.Cleanup()
	m.exitFn(signalExitCode(sig))
}

// signalExitCode returns the conventional exit code for an interrupted
// process: 128 plus the signal number, for example 130 for SIGINT and 143 for
// SIGTERM.
func signalExitCode(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 1
}
