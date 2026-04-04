package cleanup

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	m := NewManager()
	assert.NotNil(t, m, "NewManager should return non-nil manager")
	assert.NotNil(t, m.cleanupFn, "cleanupFn slice should be initialized")
	assert.NotNil(t, m.exitFn, "exitFn should be initialized")
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

func TestManager_InjectableExitFn(t *testing.T) {
	m := &Manager{
		cleanupFn: make([]func(), 0),
	}

	var exitCode int
	m.exitFn = func(code int) {
		exitCode = code
	}

	m.exitFn(42)
	assert.Equal(t, 42, exitCode)
}

func TestHandleSignal(t *testing.T) {
	m := NewManager()

	var exitCode int
	var cleanupCalled bool
	m.exitFn = func(code int) {
		exitCode = code
	}
	m.RegisterCleanup(func() {
		cleanupCalled = true
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go m.handleSignal(sigChan)

	sigChan <- syscall.SIGTERM

	assert.Eventually(t, func() bool { return cleanupCalled }, time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return exitCode == 0 }, time.Second, 10*time.Millisecond)
}

func TestHandleSignal_SIGINT(t *testing.T) {
	m := NewManager()

	var exitCode int
	m.exitFn = func(code int) {
		exitCode = code
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go m.handleSignal(sigChan)

	sigChan <- os.Interrupt

	assert.Eventually(t, func() bool { return exitCode == 0 }, time.Second, 10*time.Millisecond)
}

func TestCleanupIsCalledOnce(t *testing.T) {
	m := NewManager()

	m.exitFn = func(code int) {}

	callCount := 0
	m.RegisterCleanup(func() {
		callCount++
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.handleSignal(sigChan)
	}()

	sigChan <- syscall.SIGTERM
	wg.Wait()

	assert.Equal(t, 1, callCount)
}
