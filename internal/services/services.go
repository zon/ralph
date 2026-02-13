package services

import (
	"fmt"
	"os/exec"
	"syscall"

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

// StopAllServices stops all services in reverse order
func StopAllServices(processes []*Process) {
	// Stop services in reverse order (LIFO)
	for i := len(processes) - 1; i >= 0; i-- {
		if err := processes[i].Stop(); err != nil {
			logger.Error("Error stopping service %s: %v", processes[i].Name, err)
		}
	}
}
