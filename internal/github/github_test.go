package github

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/testutil"
)

func TestIsGHInstalled(t *testing.T) {
	ctx := testutil.NewContext()
	_ = IsGHInstalled(ctx)
}

func TestIsAuthenticated(t *testing.T) {
	ctx := testutil.NewContext()
	_ = IsAuthenticated(ctx)
}

func TestCreatePR(t *testing.T) {
	ctx := testutil.NewContext()
	_, _ = CreatePR(ctx, "Test PR", "Test body", "main", "feature-branch")
}

func TestMakeRepo(t *testing.T) {
	repo := MakeRepo("zon", "ralph")
	assert.Equal(t, "zon", repo.Owner)
	assert.Equal(t, "ralph", repo.Name)
}

func TestCloneURL(t *testing.T) {
	assert.Equal(t, "https://github.com/zon/ralph.git", CloneURL("zon", "ralph"))
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "long string",
			input:  "hello world this is a long string",
			maxLen: 10,
			want:   "hello worl...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.want, got, "truncate should return expected value")
		})
	}
}
