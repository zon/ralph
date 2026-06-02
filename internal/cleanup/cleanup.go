package cleanup

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/zon/ralph/internal/logger"
	"github.com/zon/ralph/internal/output"
)

// Manager manages cleanup operations for graceful shutdown
type Manager struct {
	mu        sync.Mutex
	cleanupFn []func()
	exitFn    func(int)
	out       *output.Client
}

// NewManager creates a new cleanup manager
func NewManager(out *output.Client) *Manager {
	return &Manager{
		cleanupFn: make([]func(), 0),
		exitFn:    os.Exit,
		out:       out,
	}
}

// RegisterCleanup adds a cleanup function to be called on shutdown
func (m *Manager) RegisterCleanup(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanupFn = append(m.cleanupFn, fn)
}

// Cleanup executes all cleanup operations
func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Execute cleanup functions in reverse order
	for i := len(m.cleanupFn) - 1; i >= 0; i-- {
		m.cleanupFn[i]()
	}
	m.cleanupFn = nil
}

// SetupSignalHandlers configures handlers for SIGINT and SIGTERM
// When a signal is received, it executes cleanup and exits
func (m *Manager) SetupSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go m.handleSignal(sigChan)
}

func (m *Manager) handleSignal(sigChan <-chan os.Signal) {
	sig := <-sigChan
	logger.Infof("Received signal: %v", sig)
	logger.Info("Cleaning up...")
	m.Cleanup()
	m.exitFn(0)
}
