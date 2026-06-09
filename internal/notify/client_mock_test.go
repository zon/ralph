package notify

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockNotifierImplementsNotifier(t *testing.T) {
	var _ Notifier = (*MockNotifier)(nil)
}

func TestMockNotifierNotify_CallsNotifyFn(t *testing.T) {
	var title, message, appIcon string
	m := &MockNotifier{
		NotifyFn: func(t, msg, icon string) error {
			title, message, appIcon = t, msg, icon
			return nil
		},
	}
	err := m.Notify("test-title", "test-message", "test-icon")
	assert.NoError(t, err)
	assert.Equal(t, "test-title", title)
	assert.Equal(t, "test-message", message)
	assert.Equal(t, "test-icon", appIcon)
}

func TestMockNotifierNotify_NilNotifyFn(t *testing.T) {
	m := &MockNotifier{}
	err := m.Notify("title", "msg", "icon")
	assert.NoError(t, err)
}

func TestMockNotifierNotify_ReturnsError(t *testing.T) {
	wantErr := errors.New("notify failed")
	m := &MockNotifier{
		NotifyFn: func(_, _, _ string) error { return wantErr },
	}
	err := m.Notify("title", "msg", "icon")
	assert.ErrorIs(t, err, wantErr)
}

func TestMockClientError_AppendsToSlice(t *testing.T) {
	m := &MockClient{}
	m.Error("test-slug")
	assert.Equal(t, []string{"test-slug"}, m.ErrorsSlice)
}

func TestMockClientError_CallsErrorFunc(t *testing.T) {
	var called string
	m := &MockClient{
		ErrorFunc: func(slug string) { called = slug },
	}
	m.Error("test-slug")
	assert.Equal(t, "test-slug", called)
}

func TestMockClientError_NilErrorFunc(t *testing.T) {
	m := &MockClient{}
	assert.NotPanics(t, func() { m.Error("test-slug") })
}

func TestMockClientSuccess_AppendsToSlice(t *testing.T) {
	m := &MockClient{}
	m.Success("test-slug")
	assert.Equal(t, []string{"test-slug"}, m.SuccessesSlice)
}

func TestMockClientSuccess_CallsSuccessFunc(t *testing.T) {
	var called string
	m := &MockClient{
		SuccessFunc: func(slug string) { called = slug },
	}
	m.Success("test-slug")
	assert.Equal(t, "test-slug", called)
}

func TestMockClientSuccess_NilSuccessFunc(t *testing.T) {
	m := &MockClient{}
	assert.NotPanics(t, func() { m.Success("test-slug") })
}
