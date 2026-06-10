package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zon/ralph/internal/project"
)

func TestPassCmd_ConfirmationMessage(t *testing.T) {
	tests := []struct {
		name       string
		reqSlug    string
		searchSlug string
		initial    bool
		falseFlag  bool
		wantStatus bool
		wantOut    string
		wantErr    bool
	}{
		{
			name:       "mark passing",
			reqSlug:    "my-req",
			searchSlug: "my-req",
			initial:    false,
			falseFlag:  false,
			wantStatus: true,
			wantOut:    "passing",
			wantErr:    false,
		},
		{
			name:       "already passing",
			reqSlug:    "my-req",
			searchSlug: "my-req",
			initial:    true,
			falseFlag:  false,
			wantStatus: true,
			wantOut:    "passing",
			wantErr:    false,
		},
		{
			name:       "mark failing",
			reqSlug:    "my-req",
			searchSlug: "my-req",
			initial:    true,
			falseFlag:  true,
			wantStatus: false,
			wantOut:    "failing",
			wantErr:    false,
		},
		{
			name:       "file not found",
			reqSlug:    "my-req",
			searchSlug: "my-req",
			initial:    false,
			falseFlag:  false,
			wantStatus: false,
			wantOut:    "",
			wantErr:    true,
		},
		{
			name:       "slug not found",
			reqSlug:    "my-req",
			searchSlug: "unknown-slug",
			initial:    false,
			falseFlag:  false,
			wantStatus: false,
			wantOut:    "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.name == "file not found" {
				path = project.NonExistentFile(t)
			} else {
				path = project.FileWithRequirement(t, tt.reqSlug, tt.initial)
			}

			r, w, err := os.Pipe()
			require.NoError(t, err)
			old := os.Stdout
			os.Stdout = w
			defer func() { os.Stdout = old }()

			cmd := &PassCmd{ProjectFile: path, Slug: tt.searchSlug, False: tt.falseFlag}
			cmdErr := cmd.Run()

			w.Close()
			out, _ := io.ReadAll(r)

			if tt.wantErr {
				assert.Error(t, cmdErr)
				assert.Empty(t, string(out))
				return
			}

			require.NoError(t, cmdErr)
			assert.Equal(t, tt.wantStatus, project.RequirementStatus(t, path, tt.searchSlug))
			assert.Contains(t, string(out), "Requirement")
			assert.Contains(t, string(out), tt.searchSlug)
			assert.Contains(t, string(out), tt.wantOut)
		})
	}
}
