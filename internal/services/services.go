package services

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
)

// RunBefore executes commands sequentially before starting services
// Commands are run with connected output and expected to exit
func RunBefore(cmds []config.Before) error {
	if len(cmds) == 0 {
		return nil
	}

	logger.Verbosef("Running %d before command(s)...", len(cmds))

	for _, cmd := range cmds {
		cmdStr := fmt.Sprintf("%s %s", cmd.Command, strings.Join(cmd.Args, " "))

		logger.Infof("Running before: %s", cmd.Name)
		logger.Verbosef("Command: %s", cmdStr)

		c := exec.Command(cmd.Command, cmd.Args...)

		if cmd.WorkDir != "" {
			c.Dir = cmd.WorkDir
		}

		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			if cmd.Optional {
				logger.Warningf("Optional before %s failed: %v", cmd.Name, err)
				continue
			}
			return fmt.Errorf("before %s failed: %w", cmd.Name, err)
		}

		logger.Successf("Before %s completed successfully", cmd.Name)
	}

	return nil
}

// Process represents a running service process
type Process struct {
	Name    string
	service config.Service
	cmd     *exec.Cmd
	pid     int
	logFile *os.File // {service}.log in the repo root, capturing output during startup
}

// Manager manages a collection of running services
// It tracks all running processes and ensures they're only stopped once
type Manager struct {
	mu        sync.Mutex
	processes []*Process
}

// NewManager creates a new service manager
func NewManager() *Manager {
	return &Manager{
		processes: make([]*Process, 0),
	}
}

// Start starts all configured services and tracks them.
// On failure, it returns the failing service alongside the error.
func (m *Manager) Start(services []config.Service) (config.Service, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	processes, failedSvc, err := startAllServices(services)
	if err != nil {
		return failedSvc, err
	}

	m.processes = processes
	return config.Service{}, nil
}

// Stop stops all tracked services and clears the process list
// Can be called multiple times safely - subsequent calls are no-ops
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If no processes, nothing to do
	if len(m.processes) == 0 {
		return
	}

	// Stop all services
	stopAllServices(m.processes)

	// Clear the process list so subsequent calls are no-ops
	m.processes = nil
}

// startService starts a service and returns a Process handle
func startService(svc config.Service) (*Process, error) {
	cmdStr := fmt.Sprintf("%s %s", svc.Command, joinArgs(svc.Args))
	return createAndStartProcess(svc, cmdStr)
}

func createAndStartProcess(svc config.Service, cmdStr string) (*Process, error) {
	cmd := exec.Command(svc.Command, svc.Args...)

	if svc.WorkDir != "" {
		cmd.Dir = svc.WorkDir
	}

	logFile, err := openLogFile(svc)
	if err != nil {
		return nil, err
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.Remove(logFile.Name())
		return nil, fmt.Errorf("failed to start service %s (command: %s): %w", svc.Name, cmdStr, err)
	}

	return &Process{
		Name:    svc.Name,
		service: svc,
		cmd:     cmd,
		pid:     cmd.Process.Pid,
		logFile: logFile,
	}, nil
}

func openLogFile(svc config.Service) (*os.File, error) {
	logPath := fmt.Sprintf("%s.log", svc.Name)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file for service %s: %w", svc.Name, err)
	}
	return logFile, nil
}

// Stop gracefully stops a service process
// It sends SIGTERM, waits for graceful shutdown, then sends SIGKILL if still running
func (p *Process) Stop() error {
	return p.StopWithTimeout(5 * time.Second)
}

// StopWithTimeout gracefully stops a service process with a custom timeout
// It sends SIGTERM, waits up to timeout for graceful shutdown, then sends SIGKILL if still running
func (p *Process) StopWithTimeout(timeout time.Duration) error {
	if p.cmd == nil || p.cmd.Process == nil {
		return fmt.Errorf("no process to stop for service: %s", p.Name)
	}

	if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		logger.Warningf("Failed to send SIGTERM to %s: %v", p.Name, err)
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case <-done:
		logger.Infof("Service %s stopped", p.Name)
		return nil
	case <-time.After(timeout):
		logger.Warningf("Service %s did not stop gracefully, sending SIGKILL", p.Name)
		if err := p.cmd.Process.Kill(); err != nil {
			logger.Errorf("Failed to kill service %s: %v", p.Name, err)
			return fmt.Errorf("failed to kill service %s: %w", p.Name, err)
		}
		<-done
		logger.Infof("Service %s stopped", p.Name)
		return nil
	}
}

// IsRunning checks if the process is still running
func (p *Process) IsRunning() bool {
	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	err := p.cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// joinArgs joins command arguments into a string for display
func joinArgs(ctx []string) string {
	if len(ctx) == 0 {
		return ""
	}

	result := ""
	for i, arg := range ctx {
		if i > 0 {
			result += " "
		}
		// Quote ctx that contain spaces
		if containsSpace(arg) {
			result += fmt.Sprintf("'%s'", arg)
		} else {
			result += arg
		}
	}
	return result
}

// containsSpace checks if a string contains spaces
func containsSpace(s string) bool {
	for _, r := range s {
		if r == ' ' {
			return true
		}
	}
	return false
}

// logServiceOutput logs the contents of the service's log file, if any
func logServiceOutput(p *Process) {
	if p.logFile == nil {
		return
	}
	if _, err := p.logFile.Seek(0, 0); err != nil {
		return
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(p.logFile); err != nil || buf.Len() == 0 {
		return
	}
	logger.Infof("Service %s output:\n%s", p.Name, buf.String())
}

// closeLogFile closes the log file handle without removing the file,
// so the agent can still read it after the service is healthy
func closeLogFile(p *Process) {
	if p.logFile != nil {
		p.logFile.Close()
		p.logFile = nil
	}
}

// cleanupOutput closes and removes the service log file
func cleanupOutput(p *Process) {
	if p.logFile != nil {
		name := p.logFile.Name()
		p.logFile.Close()
		os.Remove(name)
		p.logFile = nil
	}
}

// LogFileName returns the log file path for a service name
func LogFileName(serviceName string) string {
	return fmt.Sprintf("%s.log", serviceName)
}

// startAllServices starts all services and waits for them to become healthy
// Returns a slice of started processes and any error encountered
func startAllServices(services []config.Service) ([]*Process, config.Service, error) {
	processes := []*Process{}

	for _, svc := range services {
		proc, err := startService(svc)
		if err != nil {
			stopAllServices(processes)
			return nil, svc, fmt.Errorf("failed to start service %s: %w", svc.Name, err)
		}
		processes = append(processes, proc)

		timeout := time.Duration(svc.Timeout) * time.Second

		if err := WaitForHealth(proc, timeout); err != nil {
			logServiceOutput(proc)
			cleanupOutput(proc)
			stopAllServices(processes)
			return nil, svc, fmt.Errorf("health check failed for service %s: %w", svc.Name, err)
		}

		closeLogFile(proc)
		logger.Infof("Service %s is ready", svc.Name)
	}

	return processes, config.Service{}, nil
}

// stopAllServices stops all services in reverse order
// It stops services gracefully with SIGTERM, waiting for clean shutdown
// Services that don't stop within timeout are force-killed with SIGKILL
func stopAllServices(processes []*Process) {
	if len(processes) == 0 {
		return
	}

	// Stop services in reverse order (LIFO - last started, first stopped)
	for i := len(processes) - 1; i >= 0; i-- {
		p := processes[i]
		if err := p.Stop(); err != nil {
			logger.Warningf("Error stopping service %s: %v", p.Name, err)
			// Continue stopping other services even if one fails
		}
		if p.service.Port > 0 {
			if err := waitForPortRelease(p.service.Port, 5*time.Second); err != nil {
				logger.Warningf("Port %d may still be in use after stopping %s: %v", p.service.Port, p.Name, err)
			}
		}
	}
}

// checkPort checks if a TCP port is open and accepting connections
func checkPort(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// waitForPort waits for a TCP port to become available with a timeout
// Returns nil if port becomes available, error if timeout is reached
func waitForPort(port int, timeout time.Duration) error {
	address := fmt.Sprintf("localhost:%d", port)
	deadline := time.Now().Add(timeout)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}

		// Wait before retrying
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout waiting for port %d to become available", port)
}

// waitForPortRelease waits for a TCP port to stop accepting connections
// Returns nil when the port is released, error if timeout is reached
func waitForPortRelease(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 500 * time.Millisecond

	for time.Now().Before(deadline) {
		if !checkPort(port) {
			return nil
		}
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout waiting for port %d to be released", port)
}

// WaitForHealth waits for a service to become healthy
// For services with ports, it checks port availability
// For services without ports, it just verifies the process is running
func WaitForHealth(p *Process, timeout time.Duration) error {
	if p.service.Port > 0 {
		return waitForPort(p.service.Port, timeout)
	}

	if !p.IsRunning() {
		return fmt.Errorf("service %s is not running", p.Name)
	}

	return nil
}
