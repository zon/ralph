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

func TestCleanup(t *testing.T) {
	t.Run("single function called", func(t *testing.T) {
		m := NewManager(nil)

		called := false
		m.RegisterCleanup(func() { called = true })
		m.Cleanup()

		assert.True(t, called, "cleanup function should have been called")
	})

	t.Run("multiple functions called in reverse order", func(t *testing.T) {
		m := NewManager(nil)

		var callOrder []int
		m.RegisterCleanup(func() { callOrder = append(callOrder, 1) })
		m.RegisterCleanup(func() { callOrder = append(callOrder, 2) })
		m.RegisterCleanup(func() { callOrder = append(callOrder, 3) })

		m.Cleanup()

		assert.Equal(t, []int{3, 2, 1}, callOrder)
	})

	t.Run("idempotent", func(t *testing.T) {
		m := NewManager(nil)

		count := 0
		m.RegisterCleanup(func() { count++ })
		m.Cleanup()
		m.Cleanup()

		assert.Equal(t, 1, count, "cleanup should only execute once")
	})

	t.Run("empty manager", func(t *testing.T) {
		m := NewManager(nil)

		assert.NotPanics(t, func() { m.Cleanup() })
	})
}

func TestHandleSignal(t *testing.T) {
	tests := []struct {
		name string
		sig  os.Signal
	}{
		{"SIGTERM", syscall.SIGTERM},
		{"os.Interrupt", os.Interrupt},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			sigChan <- tt.sig

			assert.Eventually(t, func() bool { return cleanupCalled }, time.Second, 10*time.Millisecond)
			assert.Eventually(t, func() bool { return exitCode == 0 }, time.Second, 10*time.Millisecond)
		})
	}
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
