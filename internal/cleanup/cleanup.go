package cleanup

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/zon/ralph/internal/services"
)

// Manager manages cleanup operations for graceful shutdown
type Manager struct {
	mu        sync.Mutex
	processes []*services.Process
	cleanupFn []func()
}

// NewManager creates a new cleanup manager
func NewManager() *Manager {
	return &Manager{
		processes: make([]*services.Process, 0),
		cleanupFn: make([]func(), 0),
	}
}

// RegisterProcesses adds processes to be cleaned up on shutdown
func (m *Manager) RegisterProcesses(procs []*services.Process) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processes = append(m.processes, procs...)
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

	// Stop all registered services
	if len(m.processes) > 0 {
		services.StopAllServices(m.processes)
		m.processes = nil
	}

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

	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %v\n", sig)
		fmt.Println("Cleaning up...")
		m.Cleanup()
		os.Exit(0)
	}()
}
