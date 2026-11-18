package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCurrentContext(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		viewExit    int
		wantContext Context
		wantErr     string
	}{
		{
			name:        "resolves the context namespace",
			namespace:   "'staging'",
			wantContext: Context{Name: "test-context", Namespace: "staging"},
		},
		{
			name:     "reports a namespace lookup failure",
			viewExit: 1,
			wantErr:  `failed to get namespace for context "test-context"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			script := fmt.Sprintf(`#!/bin/bash
if [ "$1" = "config" ] && [ "$2" = "current-context" ]; then
  echo "test-context"
  exit 0
fi
if [ "$1" = "config" ] && [ "$2" = "view" ]; then
  echo "%s"
  echo "view failed" >&2
  exit %d
fi
exit 1
`, tt.namespace, tt.viewExit)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0755))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

			got, err := NewClient().GetCurrentContext(context.Background())
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantContext, got)
		})
	}
}
