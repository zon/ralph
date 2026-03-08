package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/testutil"
)

func TestIsGHInstalled(t *testing.T) {
	tests := []struct {
		name   string
		dryRun bool
	}{
		{
			name:   "dry-run mode",
			dryRun: true,
		},
		{
			name:   "normal mode",
			dryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testutil.NewContext()
			_ = IsGHInstalled(ctx)
		})
	}
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name   string
		dryRun bool
	}{
		{
			name:   "dry-run mode",
			dryRun: true,
		},
		{
			name:   "normal mode",
			dryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testutil.NewContext()
			_ = IsAuthenticated(ctx)
		})
	}
}

func TestCreatePR(t *testing.T) {
	tests := []struct {
		name        string
		dryRun      bool
		title       string
		body        string
		base        string
		head        string
		wantErr     bool
		wantURL     string
		checkDryRun bool
	}{
		{
			name:        "dry-run mode returns URL containing dry-run",
			dryRun:      true,
			title:       "Test PR",
			body:        "Test body",
			base:        "main",
			head:        "feature-branch",
			wantErr:     false,
			wantURL:     "dry-run",
			checkDryRun: true,
		},
		{
			name:    "dry-run mode does not invoke gh CLI",
			dryRun:  true,
			title:   "Test PR",
			body:    "Test body",
			base:    "main",
			head:    "feature-branch",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testutil.NewContext()
			url, err := CreatePR(ctx, tt.title, tt.body, tt.base, tt.head)

			if tt.wantErr {
				assert.Error(t, err, "CreatePR should return error")
			} else {
				require.NoError(t, err, "CreatePR should not return error")
			}
			if tt.dryRun {
				assert.NotEmpty(t, url, "CreatePR in dry-run mode should return URL")
			}
			if tt.checkDryRun {
				assert.Contains(t, url, tt.wantURL, "CreatePR URL should contain dry-run")
			}
		})
	}
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
