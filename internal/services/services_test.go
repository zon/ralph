package services

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/zon/ralph/internal/config"
)

// cleanupLogs removes any service log files created during a test
func cleanupLogs(t *testing.T, services []config.Service) {
	t.Helper()
	for _, svc := range services {
		os.Remove(LogFileName(svc.Name))
	}
}

func TestCheckPort(t *testing.T) {
	// Start a listener on a random port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Port should be available
	if !CheckPort(port) {
		t.Errorf("CheckPort(%d) = false, want true", port)
	}

	// Close listener
	listener.Close()

	// Give it a moment to fully close
	time.Sleep(100 * time.Millisecond)

	// Port should not be available anymore
	if CheckPort(port) {
		t.Errorf("CheckPort(%d) = true after close, want false", port)
	}
}

func TestWaitForPort(t *testing.T) {
	// Start a listener on a random port after a delay
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Start listener after 500ms
	go func() {
		time.Sleep(500 * time.Millisecond)
		l, err := net.Listen("tcp", listener.Addr().String())
		if err != nil {
			t.Logf("Failed to restart listener: %v", err)
			return
		}
		defer l.Close()
		time.Sleep(2 * time.Second) // Keep it open
	}()

	// Wait for port to become available
	err = WaitForPort(port, 3*time.Second)
	if err != nil {
		t.Errorf("WaitForPort failed: %v", err)
	}
}

func TestWaitForPortTimeout(t *testing.T) {
	// Try to wait for a port that will never open
	// Use a high port number that's unlikely to be in use
	err := WaitForPort(54321, 1*time.Second)
	if err == nil {
		t.Error("WaitForPort should have timed out but didn't")
	}
}

func TestWaitForHealthDryRun(t *testing.T) {
	// Create a mock process with port
	proc := &Process{
		Name: "test-service",
		service: config.Service{
			Name:    "test-service",
			Command: "echo",
			Port:    8080,
		},
		pid: -1, // Dry-run sentinel
	}

	// Should succeed without actually checking port
	err := WaitForHealth(proc, 1*time.Second, true)
	if err != nil {
		t.Errorf("WaitForHealth in dry-run mode failed: %v", err)
	}

	// Test without port
	proc.service.Port = 0
	err = WaitForHealth(proc, 1*time.Second, true)
	if err != nil {
		t.Errorf("WaitForHealth in dry-run mode (no port) failed: %v", err)
	}
}

func TestStartServiceDryRun(t *testing.T) {
	svc := config.Service{
		Name:    "test-service",
		Command: "echo",
		Args:    []string{"hello"},
		Port:    8080,
	}

	proc, err := startService(svc, true)
	if err != nil {
		t.Fatalf("startService in dry-run mode failed: %v", err)
	}

	if !proc.isDryRun() {
		t.Errorf("Expected isDryRun() = true for dry-run, got false")
	}

	if proc.Name != "test-service" {
		t.Errorf("Expected Name = test-service, got %s", proc.Name)
	}
}

func TestStartAllServicesDryRun(t *testing.T) {
	services := []config.Service{
		{
			Name:    "service1",
			Command: "echo",
			Args:    []string{"service1"},
			Port:    8080,
		},
		{
			Name:    "service2",
			Command: "echo",
			Args:    []string{"service2"},
			Port:    8081,
		},
		{
			Name:    "service3",
			Command: "echo",
			Args:    []string{"service3"},
			// No port - process check only
		},
	}

	processes, _, err := startAllServices(services, true)
	if err != nil {
		t.Fatalf("startAllServices in dry-run mode failed: %v", err)
	}

	if len(processes) != 3 {
		t.Errorf("Expected 3 processes, got %d", len(processes))
	}

	for i, proc := range processes {
		if !proc.isDryRun() {
			t.Errorf("Process %d: Expected isDryRun() = true for dry-run, got false", i)
		}
	}
}

func TestGracefulShutdown(t *testing.T) {
	// Start a long-running process that responds to SIGTERM
	svc := config.Service{
		Name:    "sleep-service",
		Command: "sleep",
		Args:    []string{"30"},
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Verify process is running
	if !proc.IsRunning() {
		t.Fatal("Process should be running")
	}

	// Stop the process gracefully
	err = proc.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}

	// Verify process is no longer running
	time.Sleep(100 * time.Millisecond)
	if proc.IsRunning() {
		t.Error("Process should have stopped")
	}
}

func TestForceKillAfterTimeout(t *testing.T) {
	// Start a process that ignores SIGTERM (trap in shell)
	// This simulates a process that doesn't gracefully shut down
	svc := config.Service{
		Name:    "stubborn-service",
		Command: "sh",
		Args:    []string{"-c", "trap '' TERM; sleep 30"},
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	// Verify process is running
	if !proc.IsRunning() {
		t.Fatal("Process should be running")
	}

	// Stop with a short timeout - should trigger SIGKILL
	err = proc.StopWithTimeout(500 * time.Millisecond)
	if err != nil {
		t.Errorf("StopWithTimeout failed: %v", err)
	}

	// Verify process was killed
	time.Sleep(100 * time.Millisecond)
	if proc.IsRunning() {
		t.Error("Process should have been killed")
	}
}

func TestStopAllServicesOrder(t *testing.T) {
	// Start multiple services
	services := []config.Service{
		{Name: "service1", Command: "sleep", Args: []string{"30"}},
		{Name: "service2", Command: "sleep", Args: []string{"30"}},
		{Name: "service3", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	processes := []*Process{}
	for _, svc := range services {
		proc, err := startService(svc, false)
		if err != nil {
			t.Fatalf("Failed to start service %s: %v", svc.Name, err)
		}
		processes = append(processes, proc)
	}

	// Verify all are running
	for _, proc := range processes {
		if !proc.IsRunning() {
			t.Errorf("Service %s should be running", proc.Name)
		}
	}

	// Stop all services
	stopAllServices(processes)

	// Verify all are stopped
	time.Sleep(200 * time.Millisecond)
	for _, proc := range processes {
		if proc.IsRunning() {
			t.Errorf("Service %s should have stopped", proc.Name)
		}
	}
}

func TestStopAllServicesEmpty(t *testing.T) {
	// Should handle empty slice gracefully
	stopAllServices([]*Process{})
	// If we get here without panic, test passes
}

func TestManagerStartStop(t *testing.T) {
	mgr := NewManager()

	services := []config.Service{
		{Name: "service1", Command: "sleep", Args: []string{"30"}},
		{Name: "service2", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	// Start services
	_, err := mgr.Start(services, false)
	if err != nil {
		t.Fatalf("Manager.Start failed: %v", err)
	}

	// Verify processes are tracked
	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	if processCount != 2 {
		t.Errorf("Expected 2 processes, got %d", processCount)
	}

	// Stop services
	mgr.Stop()

	// Verify process list is cleared
	mgr.mu.Lock()
	processCount = len(mgr.processes)
	mgr.mu.Unlock()

	if processCount != 0 {
		t.Errorf("Expected 0 processes after stop, got %d", processCount)
	}
}

func TestManagerMultipleStops(t *testing.T) {
	mgr := NewManager()

	services := []config.Service{
		{Name: "service1", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	// Start service
	_, err := mgr.Start(services, false)
	if err != nil {
		t.Fatalf("Manager.Start failed: %v", err)
	}

	// Stop multiple times - should be safe
	mgr.Stop()
	mgr.Stop()
	mgr.Stop()

	// Verify process list is still empty
	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	if processCount != 0 {
		t.Errorf("Expected 0 processes after multiple stops, got %d", processCount)
	}
}

func TestStartServiceDryRunWithWorkDir(t *testing.T) {
	svc := config.Service{
		Name:    "test-service",
		Command: "echo",
		Args:    []string{"hello"},
		WorkDir: "/tmp",
	}

	proc, err := startService(svc, true)
	if err != nil {
		t.Fatalf("startService with WorkDir in dry-run mode failed: %v", err)
	}

	if !proc.isDryRun() {
		t.Errorf("Expected isDryRun() = true for dry-run, got false")
	}
}

func TestStartServiceWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	svc := config.Service{
		Name:    "pwd-service",
		Command: "sh",
		Args:    []string{"-c", "echo hello"},
		WorkDir: tmpDir,
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	if err != nil {
		t.Fatalf("startService with WorkDir failed: %v", err)
	}
	defer proc.Stop()

	if proc.cmd.Dir != tmpDir {
		t.Errorf("cmd.Dir = %q, want %q", proc.cmd.Dir, tmpDir)
	}
}

func TestRunBeforeDryRunWithWorkDir(t *testing.T) {
	cmds := []config.Before{
		{
			Name:    "before-with-workdir",
			Command: "make",
			Args:    []string{"build"},
			WorkDir: "/tmp",
		},
	}

	// dry-run should succeed without executing
	err := RunBefore(cmds, true)
	if err != nil {
		t.Errorf("RunBefore with WorkDir in dry-run failed: %v", err)
	}
}

func TestManagerStopBeforeStart(t *testing.T) {
	mgr := NewManager()

	// Stop before starting - should be safe
	mgr.Stop()

	// Verify no panic occurred
	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	if processCount != 0 {
		t.Errorf("Expected 0 processes, got %d", processCount)
	}
}

func TestManagerDryRun(t *testing.T) {
	mgr := NewManager()

	services := []config.Service{
		{Name: "service1", Command: "echo", Args: []string{"test"}, Port: 8080},
	}

	// Start in dry-run mode
	_, err := mgr.Start(services, true)
	if err != nil {
		t.Fatalf("Manager.Start in dry-run failed: %v", err)
	}

	// Verify process is tracked
	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	if processCount != 1 {
		t.Errorf("Expected 1 process in dry-run, got %d", processCount)
	}

	// Stop should work in dry-run too
	mgr.Stop()

	mgr.mu.Lock()
	processCount = len(mgr.processes)
	mgr.mu.Unlock()

	if processCount != 0 {
		t.Errorf("Expected 0 processes after stop, got %d", processCount)
	}
}

func TestRunBeforeFailingOptional(t *testing.T) {
	cmds := []config.Before{
		{
			Name:     "failing-optional",
			Command:  "false",
			Optional: true,
		},
	}

	err := RunBefore(cmds, false)
	if err != nil {
		t.Errorf("RunBefore with failing optional command should return nil, got: %v", err)
	}
}

func TestRunBeforeFailingNonOptional(t *testing.T) {
	cmds := []config.Before{
		{
			Name:     "failing-required",
			Command:  "false",
			Optional: false,
		},
	}

	err := RunBefore(cmds, false)
	if err == nil {
		t.Error("RunBefore with failing non-optional command should return error, got nil")
	}
}

func TestRunBeforeSequentialExecution(t *testing.T) {
	cmds := []config.Before{
		{
			Name:    "first",
			Command: "sh",
			Args:    []string{"-c", "echo first"},
		},
		{
			Name:    "second",
			Command: "sh",
			Args:    []string{"-c", "echo second"},
		},
		{
			Name:    "third",
			Command: "sh",
			Args:    []string{"-c", "echo third"},
		},
	}

	err := RunBefore(cmds, false)
	if err != nil {
		t.Errorf("RunBefore with successful commands should return nil, got: %v", err)
	}
}

func TestRunBeforeWithWorkDir(t *testing.T) {
	tmpDir := t.TempDir()

	cmds := []config.Before{
		{
			Name:    "pwd-check",
			Command: "sh",
			Args:    []string{"-c", "pwd"},
			WorkDir: tmpDir,
		},
	}

	err := RunBefore(cmds, false)
	if err != nil {
		t.Errorf("RunBefore with WorkDir failed: %v", err)
	}
}

func TestWaitForHealthProcessRunningNoPort(t *testing.T) {
	svc := config.Service{
		Name:    "health-test",
		Command: "sleep",
		Args:    []string{"30"},
		Port:    0, // No port - process liveness check
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}
	defer proc.Stop()

	err = WaitForHealth(proc, 5*time.Second, false)
	if err != nil {
		t.Errorf("WaitForHealth should return nil when process is running and no port configured, got: %v", err)
	}
}

func TestWaitForHealthProcessExitsBeforeCheck(t *testing.T) {
	svc := config.Service{
		Name:    "short-lived",
		Command: "sh",
		Args:    []string{"-c", "exit 0"},
		Port:    0,
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	if err != nil {
		t.Fatalf("Failed to start service: %v", err)
	}

	proc.cmd.Wait()

	err = WaitForHealth(proc, 5*time.Second, false)
	if err == nil {
		t.Error("WaitForHealth should return error when process exits before health check passes")
	}
}

func TestStartAllServicesRollbackOnStartFailure(t *testing.T) {
	services := []config.Service{
		{Name: "rb-start-svc1", Command: "sleep", Args: []string{"30"}, Port: 17777},
		{Name: "rb-start-svc2", Command: "nonexistent-xyz", Args: []string{}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	_, _, err := startAllServices(services, false)
	if err == nil {
		t.Fatal("startAllServices should have failed but didn't")
	}

	time.Sleep(600 * time.Millisecond)

	if CheckPort(17777) {
		t.Error("Port 17777 should have been released as part of rollback but it's still in use")
	}
}

func TestStartAllServicesRollbackOnHealthCheckFailure(t *testing.T) {
	services := []config.Service{
		{Name: "rb-health-svc1", Command: "sleep", Args: []string{"30"}, Port: 17788},
		{Name: "rb-health-svc2", Command: "sleep", Args: []string{"30"}, Port: 17787},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	_, _, err := startAllServices(services, false)
	if err == nil {
		t.Fatal("startAllServices should have failed health check but didn't")
	}

	time.Sleep(600 * time.Millisecond)

	if CheckPort(17788) {
		t.Error("Port 17788 should have been released as part of rollback but it's still in use")
	}
	if CheckPort(17787) {
		t.Error("Port 17787 should have been released as part of rollback but it's still in use")
	}
}
