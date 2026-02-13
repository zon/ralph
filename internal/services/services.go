package services

import (
	"fmt"
	"net"
	"os/exec"
	"syscall"
	"time"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/logger"
)

// Process represents a running service process
type Process struct {
	Name    string
	Service config.Service
	Cmd     *exec.Cmd
	PID     int
}

// StartService starts a service and returns a Process handle
// In dry-run mode, it logs what would be done without executing
func StartService(svc config.Service, dryRun bool) (*Process, error) {
	cmdStr := fmt.Sprintf("%s %s", svc.Command, joinArgs(svc.Args))

	if dryRun {
		logger.Info("Would start service: %s with command: %s", svc.Name, cmdStr)
		return &Process{
			Name:    svc.Name,
			Service: svc,
			PID:     -1, // Sentinel value for dry-run
		}, nil
	}

	// Create the command
	cmd := exec.Command(svc.Command, svc.Args...)

	// Redirect stdout/stderr to /dev/null
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the process in a new process group
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the service
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start service %s: %w", svc.Name, err)
	}

	logger.Success("Started service: %s (PID: %d)", svc.Name, cmd.Process.Pid)

	return &Process{
		Name:    svc.Name,
		Service: svc,
		Cmd:     cmd,
		PID:     cmd.Process.Pid,
	}, nil
}

// Stop gracefully stops a service process
func (p *Process) Stop() error {
	if p.PID == -1 {
		// Dry-run sentinel - nothing to stop
		logger.Info("Would stop service: %s", p.Name)
		return nil
	}

	if p.Cmd == nil || p.Cmd.Process == nil {
		return fmt.Errorf("no process to stop for service: %s", p.Name)
	}

	logger.Info("Stopping service: %s (PID: %d)", p.Name, p.PID)

	// Send SIGTERM for graceful shutdown
	if err := p.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		logger.Warning("Failed to send SIGTERM to %s: %v", p.Name, err)
	}

	// Wait for process to exit (with timeout handled by caller if needed)
	_ = p.Cmd.Wait()

	logger.Success("Stopped service: %s", p.Name)
	return nil
}

// IsRunning checks if the process is still running
func (p *Process) IsRunning() bool {
	if p.PID == -1 {
		// Dry-run sentinel
		return false
	}

	if p.Cmd == nil || p.Cmd.Process == nil {
		return false
	}

	// Check if process still exists
	err := p.Cmd.Process.Signal(syscall.Signal(0))
	return err == nil
}

// joinArgs joins command arguments into a string for display
func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}

	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		// Quote args that contain spaces
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

// StartAllServices starts all services and waits for them to become healthy
// Returns a slice of started processes and any error encountered
func StartAllServices(services []config.Service, dryRun bool) ([]*Process, error) {
	processes := []*Process{}
	healthTimeout := 30 * time.Second

	for _, svc := range services {
		// Start the service
		proc, err := StartService(svc, dryRun)
		if err != nil {
			// If a service fails to start, stop all previously started services
			StopAllServices(processes)
			return nil, fmt.Errorf("failed to start service %s: %w", svc.Name, err)
		}
		processes = append(processes, proc)

		// Wait for health check
		if err := WaitForHealth(proc, healthTimeout, dryRun); err != nil {
			// If health check fails, stop all services
			StopAllServices(processes)
			return nil, fmt.Errorf("health check failed for service %s: %w", svc.Name, err)
		}
	}

	return processes, nil
}

// StopAllServices stops all services in reverse order
func StopAllServices(processes []*Process) {
	// Stop services in reverse order (LIFO)
	for i := len(processes) - 1; i >= 0; i-- {
		if err := processes[i].Stop(); err != nil {
			logger.Error("Error stopping service %s: %v", processes[i].Name, err)
		}
	}
}

// CheckPort checks if a TCP port is open and accepting connections
func CheckPort(port int) bool {
	address := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", address, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// WaitForPort waits for a TCP port to become available with a timeout
// Returns nil if port becomes available, error if timeout is reached
func WaitForPort(port int, timeout time.Duration) error {
	address := fmt.Sprintf("localhost:%d", port)
	deadline := time.Now().Add(timeout)
	interval := 500 * time.Millisecond

	logger.Info("Waiting for port %d to be ready...", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			logger.Success("Port %d is ready", port)
			return nil
		}

		// Wait before retrying
		time.Sleep(interval)
	}

	return fmt.Errorf("timeout waiting for port %d to become available", port)
}

// WaitForHealth waits for a service to become healthy
// For services with ports, it checks port availability
// For services without ports, it just verifies the process is running
func WaitForHealth(p *Process, timeout time.Duration, dryRun bool) error {
	if dryRun {
		if p.Service.Port > 0 {
			logger.Info("Would wait for port %d (service: %s)", p.Service.Port, p.Name)
		} else {
			logger.Info("Would verify process is running (service: %s)", p.Name)
		}
		return nil
	}

	// If service has a port, wait for it
	if p.Service.Port > 0 {
		return WaitForPort(p.Service.Port, timeout)
	}

	// For services without ports, just verify the process is still running
	if !p.IsRunning() {
		return fmt.Errorf("service %s is not running", p.Name)
	}

	logger.Success("Service %s is running", p.Name)
	return nil
}
