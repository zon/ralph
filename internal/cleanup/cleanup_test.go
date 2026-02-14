package cleanup

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.cleanupFn == nil {
		t.Error("cleanupFn slice should be initialized")
	}
}

func TestRegisterCleanup(t *testing.T) {
	m := NewManager()

	cleanupCalled := false
	m.RegisterCleanup(func() {
		cleanupCalled = true
	})

	m.Cleanup()

	if !cleanupCalled {
		t.Error("Cleanup function should have been called")
	}
}

func TestMultipleCleanupFunctions(t *testing.T) {
	m := NewManager()

	// Track call order
	var callOrder []int
	m.RegisterCleanup(func() {
		callOrder = append(callOrder, 1)
	})
	m.RegisterCleanup(func() {
		callOrder = append(callOrder, 2)
	})
	m.RegisterCleanup(func() {
		callOrder = append(callOrder, 3)
	})

	m.Cleanup()

	// Cleanup functions should be called in reverse order
	if len(callOrder) != 3 {
		t.Errorf("Expected 3 cleanup calls, got %d", len(callOrder))
	}
	if callOrder[0] != 3 || callOrder[1] != 2 || callOrder[2] != 1 {
		t.Errorf("Expected cleanup order [3, 2, 1], got %v", callOrder)
	}
}

func TestCleanupIdempotent(t *testing.T) {
	m := NewManager()

	cleanupCount := 0
	m.RegisterCleanup(func() {
		cleanupCount++
	})

	// Call cleanup multiple times
	m.Cleanup()
	m.Cleanup()
	m.Cleanup()

	// Should only be called once (cleanup clears the functions)
	if cleanupCount != 1 {
		t.Errorf("Expected cleanup to be called once, got %d", cleanupCount)
	}
}

func TestCleanupEmptyManager(t *testing.T) {
	m := NewManager()

	// Should not panic when cleaning up empty manager
	m.Cleanup()
}
