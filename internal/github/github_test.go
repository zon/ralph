package github

import (
	"testing"

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
			// Just ensure it doesn't panic
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
			// Just ensure it doesn't panic
			_ = IsAuthenticated(ctx)
		})
	}
}

func TestCreatePR(t *testing.T) {
	tests := []struct {
		name    string
		dryRun  bool
		title   string
		body    string
		base    string
		head    string
		wantErr bool
	}{
		{
			name:    "dry-run mode returns URL",
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

			if tt.wantErr && err == nil {
				t.Error("CreatePR() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("CreatePR() unexpected error: %v", err)
			}
			if tt.dryRun && url == "" {
				t.Error("CreatePR() in dry-run mode should return URL")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}
