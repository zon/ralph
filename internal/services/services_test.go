package services

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

func cleanupLogs(t *testing.T, services []config.Service) {
	t.Helper()
	for _, svc := range services {
		os.Remove(LogFileName(svc.Name))
	}
}

func TestCheckPort(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to start listener")
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	assert.True(t, checkPort(port), "checkPort should return true for available port")

	listener.Close()

	time.Sleep(100 * time.Millisecond)

	assert.False(t, checkPort(port), "checkPort should return false after port is closed")
}

func TestWaitForPort(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err, "Failed to start listener")
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	go func() {
		time.Sleep(500 * time.Millisecond)
		l, err := net.Listen("tcp", listener.Addr().String())
		if err != nil {
			t.Logf("Failed to restart listener: %v", err)
			return
		}
		defer l.Close()
		time.Sleep(2 * time.Second)
	}()

	err = waitForPort(port, 3*time.Second)
	require.NoError(t, err, "waitForPort should succeed")
}

func TestWaitForPortTimeout(t *testing.T) {
	err := waitForPort(54321, 1*time.Second)
	assert.Error(t, err, "waitForPort should timeout")
}

func TestWaitForHealthDryRun(t *testing.T) {
	proc := &Process{
		Name: "test-service",
		service: config.Service{
			Name:    "test-service",
			Command: "echo",
			Port:    8080,
		},
		pid: -1,
	}

	err := WaitForHealth(proc, 1*time.Second, true)
	require.NoError(t, err, "WaitForHealth in dry-run mode should not fail")

	proc.service.Port = 0
	err = WaitForHealth(proc, 1*time.Second, true)
	require.NoError(t, err, "WaitForHealth in dry-run mode without port should not fail")
}

func TestStartServiceDryRun(t *testing.T) {
	svc := config.Service{
		Name:    "test-service",
		Command: "echo",
		Args:    []string{"hello"},
		Port:    8080,
	}

	proc, err := startService(svc, true)
	require.NoError(t, err, "startService in dry-run mode should not fail")

	assert.True(t, proc.isDryRun(), "isDryRun should be true")
	assert.Equal(t, "test-service", proc.Name, "Name should match")
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
		},
	}

	processes, _, err := startAllServices(services, true)
	require.NoError(t, err, "startAllServices in dry-run mode should not fail")

	assert.Len(t, processes, 3, "Should have 3 processes")
	for i, proc := range processes {
		assert.True(t, proc.isDryRun(), "Process %d should be in dry-run mode", i)
	}
}

func TestGracefulShutdown(t *testing.T) {
	svc := config.Service{
		Name:    "sleep-service",
		Command: "sleep",
		Args:    []string{"30"},
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	require.NoError(t, err, "Failed to start service")

	assert.True(t, proc.IsRunning(), "Process should be running")

	err = proc.Stop()
	require.NoError(t, err, "Stop should not fail")

	time.Sleep(100 * time.Millisecond)
	assert.False(t, proc.IsRunning(), "Process should have stopped")
}

func TestForceKillAfterTimeout(t *testing.T) {
	svc := config.Service{
		Name:    "stubborn-service",
		Command: "sh",
		Args:    []string{"-c", "trap '' TERM; sleep 30"},
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	require.NoError(t, err, "Failed to start service")

	assert.True(t, proc.IsRunning(), "Process should be running")

	err = proc.StopWithTimeout(500 * time.Millisecond)
	require.NoError(t, err, "StopWithTimeout should not fail")

	time.Sleep(100 * time.Millisecond)
	assert.False(t, proc.IsRunning(), "Process should have been killed")
}

func TestStopAllServicesOrder(t *testing.T) {
	services := []config.Service{
		{Name: "service1", Command: "sleep", Args: []string{"30"}},
		{Name: "service2", Command: "sleep", Args: []string{"30"}},
		{Name: "service3", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	processes := []*Process{}
	for _, svc := range services {
		proc, err := startService(svc, false)
		require.NoError(t, err, "Failed to start service %s", svc.Name)
		processes = append(processes, proc)
	}

	for _, proc := range processes {
		assert.True(t, proc.IsRunning(), "Service %s should be running", proc.Name)
	}

	stopAllServices(processes)

	time.Sleep(200 * time.Millisecond)
	for _, proc := range processes {
		assert.False(t, proc.IsRunning(), "Service %s should have stopped", proc.Name)
	}
}

func TestStopAllServicesEmpty(t *testing.T) {
	assert.NotPanics(t, func() {
		stopAllServices([]*Process{})
	}, "stopAllServices should handle empty slice")
}

func TestManagerStartStop(t *testing.T) {
	mgr := NewManager()

	services := []config.Service{
		{Name: "service1", Command: "sleep", Args: []string{"30"}},
		{Name: "service2", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	_, err := mgr.Start(services, false)
	require.NoError(t, err, "Manager.Start should not fail")

	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	assert.Equal(t, 2, processCount, "Should have 2 processes")

	mgr.Stop()

	mgr.mu.Lock()
	processCount = len(mgr.processes)
	mgr.mu.Unlock()

	assert.Equal(t, 0, processCount, "Should have 0 processes after stop")
}

func TestManagerMultipleStops(t *testing.T) {
	mgr := NewManager()

	services := []config.Service{
		{Name: "service1", Command: "sleep", Args: []string{"30"}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	_, err := mgr.Start(services, false)
	require.NoError(t, err, "Manager.Start should not fail")

	mgr.Stop()
	mgr.Stop()
	mgr.Stop()

	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	assert.Equal(t, 0, processCount, "Should have 0 processes after multiple stops")
}

func TestStartServiceDryRunWithWorkDir(t *testing.T) {
	svc := config.Service{
		Name:    "test-service",
		Command: "echo",
		Args:    []string{"hello"},
		WorkDir: "/tmp",
	}

	proc, err := startService(svc, true)
	require.NoError(t, err, "startService with WorkDir in dry-run mode should not fail")

	assert.True(t, proc.isDryRun(), "isDryRun should be true")
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
	require.NoError(t, err, "startService with WorkDir should not fail")
	defer proc.Stop()

	assert.Equal(t, tmpDir, proc.cmd.Dir, "cmd.Dir should match WorkDir")
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

	err := RunBefore(cmds, true)
	require.NoError(t, err, "RunBefore with WorkDir in dry-run should not fail")
}

func TestManagerStopBeforeStart(t *testing.T) {
	mgr := NewManager()

	mgr.Stop()

	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	assert.Equal(t, 0, processCount, "Should have 0 processes")
}

func TestManagerDryRun(t *testing.T) {
	mgr := NewManager()

	services := []config.Service{
		{Name: "service1", Command: "echo", Args: []string{"test"}, Port: 8080},
	}

	_, err := mgr.Start(services, true)
	require.NoError(t, err, "Manager.Start in dry-run should not fail")

	mgr.mu.Lock()
	processCount := len(mgr.processes)
	mgr.mu.Unlock()

	assert.Equal(t, 1, processCount, "Should have 1 process in dry-run")

	mgr.Stop()

	mgr.mu.Lock()
	processCount = len(mgr.processes)
	mgr.mu.Unlock()

	assert.Equal(t, 0, processCount, "Should have 0 processes after stop")
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
	require.NoError(t, err, "RunBefore with failing optional command should return nil")
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
	assert.Error(t, err, "RunBefore with failing non-optional command should return error")
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
	require.NoError(t, err, "RunBefore with successful commands should return nil")
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
	require.NoError(t, err, "RunBefore with WorkDir should not fail")
}

func TestWaitForHealthProcessRunningNoPort(t *testing.T) {
	svc := config.Service{
		Name:    "health-test",
		Command: "sleep",
		Args:    []string{"30"},
		Port:    0,
	}
	t.Cleanup(func() { cleanupLogs(t, []config.Service{svc}) })

	proc, err := startService(svc, false)
	require.NoError(t, err, "Failed to start service")
	defer proc.Stop()

	err = WaitForHealth(proc, 5*time.Second, false)
	require.NoError(t, err, "WaitForHealth should return nil when process is running")
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
	require.NoError(t, err, "Failed to start service")

	proc.cmd.Wait()

	err = WaitForHealth(proc, 5*time.Second, false)
	assert.Error(t, err, "WaitForHealth should return error when process exits")
}

func TestStartAllServicesRollbackOnStartFailure(t *testing.T) {
	services := []config.Service{
		{Name: "rb-start-svc1", Command: "sleep", Args: []string{"30"}, Port: 17777},
		{Name: "rb-start-svc2", Command: "nonexistent-xyz", Args: []string{}},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	_, _, err := startAllServices(services, false)
	assert.Error(t, err, "startAllServices should fail")

	time.Sleep(600 * time.Millisecond)

	assert.False(t, checkPort(17777), "Port should be released after rollback")
}

func TestStartAllServicesRollbackOnHealthCheckFailure(t *testing.T) {
	services := []config.Service{
		{Name: "rb-health-svc1", Command: "sleep", Args: []string{"30"}, Port: 17788},
		{Name: "rb-health-svc2", Command: "sleep", Args: []string{"30"}, Port: 17787},
	}
	t.Cleanup(func() { cleanupLogs(t, services) })

	_, _, err := startAllServices(services, false)
	assert.Error(t, err, "startAllServices should fail health check")

	time.Sleep(600 * time.Millisecond)

	assert.False(t, checkPort(17788), "Port 17788 should be released after rollback")
	assert.False(t, checkPort(17787), "Port 17787 should be released after rollback")
}
