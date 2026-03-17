package cleanup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m, "NewManager should return non-nil manager")
	assert.NotNil(t, m.cleanupFn, "cleanupFn slice should be initialized")
}

func TestRegisterCleanup(t *testing.T) {
	m := NewManager()

	cleanupCalled := false
	m.RegisterCleanup(func() {
		cleanupCalled = true
	})

	m.Cleanup()

	assert.True(t, cleanupCalled, "Cleanup function should have been called")
}

func TestMultipleCleanupFunctions(t *testing.T) {
	m := NewManager()

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

	assert.Len(t, callOrder, 3, "Expected 3 cleanup calls")
	assert.Equal(t, []int{3, 2, 1}, callOrder, "Cleanup functions should be called in reverse order")
}

func TestCleanupIdempotent(t *testing.T) {
	m := NewManager()

	cleanupCount := 0
	m.RegisterCleanup(func() {
		cleanupCount++
	})

	m.Cleanup()
	m.Cleanup()
	m.Cleanup()

	assert.Equal(t, 1, cleanupCount, "Cleanup should only be called once")
}

func TestCleanupEmptyManager(t *testing.T) {
	m := NewManager()

	m.Cleanup()
}

func TestManagerImplementsRegistrar(t *testing.T) {
	m := NewManager()
	var _ Registrar = m
}
