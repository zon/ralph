package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
