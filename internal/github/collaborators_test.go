package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockListCollaborators_Multiple(t *testing.T) {
	mock := &MockGH{
		ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
			assert.Equal(t, "test-owner", owner)
			assert.Equal(t, "test-repo", repo)
			return []string{"alice", "bob", "charlie"}, nil
		},
	}

	ctx := context.Background()
	logins, err := mock.ListCollaborators(ctx, "test-owner", "test-repo")
	assert.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob", "charlie"}, logins)
}

func TestMockListCollaborators_Single(t *testing.T) {
	mock := &MockGH{
		ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
			return []string{"alice"}, nil
		},
	}

	ctx := context.Background()
	logins, err := mock.ListCollaborators(ctx, "test-owner", "test-repo")
	assert.NoError(t, err)
	assert.Equal(t, []string{"alice"}, logins)
}

func TestMockListCollaborators_Empty(t *testing.T) {
	mock := &MockGH{
		ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
			return nil, nil
		},
	}

	ctx := context.Background()
	logins, err := mock.ListCollaborators(ctx, "test-owner", "test-repo")
	assert.NoError(t, err)
	assert.Empty(t, logins)
}

func TestMockListCollaborators_Error(t *testing.T) {
	mock := &MockGH{
		ListCollaboratorsFn: func(_ context.Context, owner, repo string) ([]string, error) {
			return nil, assert.AnError
		},
	}

	ctx := context.Background()
	_, err := mock.ListCollaborators(ctx, "test-owner", "test-repo")
	assert.Error(t, err)
}

func TestMockListCollaborators_DefaultNil(t *testing.T) {
	mock := &MockGH{}

	ctx := context.Background()
	logins, err := mock.ListCollaborators(ctx, "test-owner", "test-repo")
	assert.NoError(t, err)
	assert.Nil(t, logins)
}
