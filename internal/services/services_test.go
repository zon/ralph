package services

import (
	"testing"
	"time"

	"github.com/zon/ralph/internal/config"
)

func TestStartService_DryRun(t *testing.T) {
	svc := config.Service{
		Name:    "test-service",
		Command: "sleep",
		Args:    []string{"10"},
	}

	process, err := StartService(svc, true)
	if err != nil {
		t.Fatalf("Expected no error in dry-run mode, got: %v", err)
	}

	if process == nil {
		t.Fatal("Expected process to be returned")
	}

	if process.PID != -1 {
		t.Errorf("Expected PID to be -1 in dry-run mode, got: %d", process.PID)
	}

	if process.Name != "test-service" {
		t.Errorf("Expected service name 'test-service', got: %s", process.Name)
	}
}

func TestStartService_Real(t *testing.T) {
	svc := config.Service{
		Name:    "echo-service",
		Command: "sleep",
		Args:    []string{"2"},
	}

	process, err := StartService(svc, false)
	if err != nil {
		t.Fatalf("Expected no error starting service, got: %v", err)
	}
	defer process.Stop()

	if process == nil {
		t.Fatal("Expected process to be returned")
	}

	if process.PID <= 0 {
		t.Errorf("Expected valid PID, got: %d", process.PID)
	}

	// Process should still be running
	if !process.IsRunning() {
		t.Error("Expected process to be running")
	}
}

func TestStopService_DryRun(t *testing.T) {
	process := &Process{
		Name: "test-service",
		PID:  -1, // Dry-run sentinel
	}

	err := process.Stop()
	if err != nil {
		t.Errorf("Expected no error stopping dry-run process, got: %v", err)
	}
}

func TestStopAllServices(t *testing.T) {
	// Create some dry-run processes
	processes := []*Process{
		{Name: "service1", PID: -1},
		{Name: "service2", PID: -1},
		{Name: "service3", PID: -1},
	}

	// Should not panic or error
	StopAllServices(processes)
}

func TestJoinArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: "",
		},
		{
			name:     "single arg",
			args:     []string{"test"},
			expected: "test",
		},
		{
			name:     "multiple args",
			args:     []string{"docker", "compose", "up"},
			expected: "docker compose up",
		},
		{
			name:     "args with spaces",
			args:     []string{"echo", "hello world"},
			expected: "echo 'hello world'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinArgs(tt.args)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsRunning(t *testing.T) {
	// Test dry-run process
	dryRunProcess := &Process{
		Name: "dry-run",
		PID:  -1,
	}

	if dryRunProcess.IsRunning() {
		t.Error("Expected dry-run process to report not running")
	}

	// Test real process
	svc := config.Service{
		Name:    "sleep-service",
		Command: "sleep",
		Args:    []string{"2"},
	}

	process, err := StartService(svc, false)
	if err != nil {
		t.Fatalf("Failed to start test process: %v", err)
	}
	defer process.Stop()

	// Check immediately - should be running
	if !process.IsRunning() {
		t.Error("Expected process to be running immediately after start")
	}

	// Stop the process
	err = process.Stop()
	if err != nil {
		t.Errorf("Failed to stop process: %v", err)
	}

	// After stopping, should not be running
	time.Sleep(100 * time.Millisecond)
	if process.IsRunning() {
		t.Error("Expected process to have stopped after Stop()")
	}
}
