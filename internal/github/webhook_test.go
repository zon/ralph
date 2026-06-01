package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockRegisterWebhook_Success(t *testing.T) {
	mock := &MockGH{
		RegisterWebhookFn: func(_ context.Context, owner, repo, webhookURL, secret string) error {
			assert.Equal(t, "test-owner", owner)
			assert.Equal(t, "test-repo", repo)
			assert.Equal(t, "https://example.com/hook", webhookURL)
			assert.Equal(t, "mysecret", secret)
			return nil
		},
	}

	ctx := context.Background()
	err := mock.RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	assert.NoError(t, err)
}

func TestMockRegisterWebhook_Error(t *testing.T) {
	mock := &MockGH{
		RegisterWebhookFn: func(_ context.Context, owner, repo, webhookURL, secret string) error {
			return assert.AnError
		},
	}

	ctx := context.Background()
	err := mock.RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	assert.Error(t, err)
}

func TestMockRegisterWebhook_DefaultNil(t *testing.T) {
	mock := &MockGH{}

	ctx := context.Background()
	err := mock.RegisterWebhook(ctx, "test-owner", "test-repo", "https://example.com/hook", "mysecret")
	assert.NoError(t, err)
}
