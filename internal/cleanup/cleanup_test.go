package cleanup

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/output"
)

func TestNewManager(t *testing.T) {
	m := NewManager(nil)
	assert.NotNil(t, m, "NewManager should return non-nil manager")

	assert.NotPanics(t, func() { m.Cleanup() })

	called := false
	m.RegisterCleanup(func() { called = true })
	m.Cleanup()
	assert.True(t, called, "cleanup function should have been called")
}

func TestRegisterCleanup(t *testing.T) {
	m := NewManager(nil)

	cleanupCalled := false
	m.RegisterCleanup(func() {
		cleanupCalled = true
	})

	m.Cleanup()

	assert.True(t, cleanupCalled, "Cleanup function should have been called")
}

func TestMultipleCleanupFunctions(t *testing.T) {
	m := NewManager(nil)

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
	m := NewManager(nil)

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
	m := NewManager(nil)

	m.Cleanup()
}

func TestHandleSignal(t *testing.T) {
	m := NewManager(output.NewClient(os.Stdout, os.Stderr, false))

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
	m := NewManager(output.NewClient(os.Stdout, os.Stderr, false))

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
	m := NewManager(output.NewClient(os.Stdout, os.Stderr, false))

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
