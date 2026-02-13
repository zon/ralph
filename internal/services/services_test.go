package services

import (
	"net"
	"testing"
	"time"

	"github.com/zon/ralph/internal/config"
)

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
		Service: config.Service{
			Name:    "test-service",
			Command: "echo",
			Port:    8080,
		},
		PID: -1, // Dry-run sentinel
	}

	// Should succeed without actually checking port
	err := WaitForHealth(proc, 1*time.Second, true)
	if err != nil {
		t.Errorf("WaitForHealth in dry-run mode failed: %v", err)
	}

	// Test without port
	proc.Service.Port = 0
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

	proc, err := StartService(svc, true)
	if err != nil {
		t.Fatalf("StartService in dry-run mode failed: %v", err)
	}

	if proc.PID != -1 {
		t.Errorf("Expected PID = -1 for dry-run, got %d", proc.PID)
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

	processes, err := StartAllServices(services, true)
	if err != nil {
		t.Fatalf("StartAllServices in dry-run mode failed: %v", err)
	}

	if len(processes) != 3 {
		t.Errorf("Expected 3 processes, got %d", len(processes))
	}

	for i, proc := range processes {
		if proc.PID != -1 {
			t.Errorf("Process %d: Expected PID = -1 for dry-run, got %d", i, proc.PID)
		}
	}
}
