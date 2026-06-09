package cleanup

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
	out       *output.Client
}

func NewManager(out *output.Client) *Manager {
	return &Manager{
		cleanupFn: make([]func(), 0),
		exitFn:    os.Exit,
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
	defer m.mu.Unlock()

	for i := len(m.cleanupFn) - 1; i >= 0; i-- {
		m.cleanupFn[i]()
	}
	m.cleanupFn = nil
}

func (m *Manager) SetupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go m.handleSignal(sigChan)
}

func (m *Manager) handleSignal(sigChan <-chan os.Signal) {
	sig := <-sigChan
	m.out.Infof("Received signal: %v", sig)
	m.out.Info("Cleaning up...")
	m.Cleanup()
	m.exitFn(0)
}
