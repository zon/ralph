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
		logger.Infof("Would start service: %s with command: %s", svc.Name, cmdStr)
		return &Process{
			Name:    svc.Name,
			Service: svc,
			PID:     -1, // Sentinel value for dry-run
		}, nil
	}

	// Create the command
	cmd := exec.Command(svc.Command, svc.Args...)

	// Redirect stdout/stderr to discard output
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

	logger.Successf("Started service: %s (PID: %d)", svc.Name, cmd.Process.Pid)

	return &Process{
		Name:    svc.Name,
		Service: svc,
		Cmd:     cmd,
		PID:     cmd.Process.Pid,
	}, nil
}

// Stop gracefully stops a service process
// It sends SIGTERM, waits for graceful shutdown, then sends SIGKILL if still running
func (p *Process) Stop() error {
	return p.StopWithTimeout(5 * time.Second)
}

// StopWithTimeout gracefully stops a service process with a custom timeout
// It sends SIGTERM, waits up to timeout for graceful shutdown, then sends SIGKILL if still running
func (p *Process) StopWithTimeout(timeout time.Duration) error {
	if p.PID == -1 {
		// Dry-run sentinel - nothing to stop
		logger.Infof("Would stop service: %s", p.Name)
		return nil
	}

	if p.Cmd == nil || p.Cmd.Process == nil {
		return fmt.Errorf("no process to stop for service: %s", p.Name)
	}

	logger.Infof("Stopping service: %s (PID: %d)", p.Name, p.PID)

	// Send SIGTERM for graceful shutdown
	if err := p.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// Process may have already exited
		logger.Warningf("Failed to send SIGTERM to %s: %v", p.Name, err)
		return nil // Not necessarily an error if process already exited
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- p.Cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited gracefully
		logger.Successf("Stopped service: %s", p.Name)
		return nil
	case <-time.After(timeout):
		// Timeout reached, force kill
		logger.Warningf("Service %s did not stop gracefully, sending SIGKILL", p.Name)
		if err := p.Cmd.Process.Kill(); err != nil {
			logger.Errorf("Failed to kill service %s: %v", p.Name, err)
			return fmt.Errorf("failed to kill service %s: %w", p.Name, err)
		}
		// Wait for kill to complete
		<-done
		logger.Successf("Forcefully stopped service: %s", p.Name)
		return nil
	}
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
// It stops services gracefully with SIGTERM, waiting for clean shutdown
// Services that don't stop within timeout are force-killed with SIGKILL
func StopAllServices(processes []*Process) {
	if len(processes) == 0 {
		return
	}

	logger.Infof("Stopping %d service(s)...", len(processes))

	// Stop services in reverse order (LIFO - last started, first stopped)
	for i := len(processes) - 1; i >= 0; i-- {
		if err := processes[i].Stop(); err != nil {
			logger.Errorf("Error stopping service %s: %v", processes[i].Name, err)
			// Continue stopping other services even if one fails
		}
	}

	logger.Infof("All services stopped")
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

	logger.Infof("Waiting for port %d to be ready...", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			logger.Successf("Port %d is ready", port)
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
			logger.Infof("Would wait for port %d (service: %s)", p.Service.Port, p.Name)
		} else {
			logger.Infof("Would verify process is running (service: %s)", p.Name)
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

	logger.Successf("Service %s is running", p.Name)
	return nil
}
