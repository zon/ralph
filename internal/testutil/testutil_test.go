package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContext(t *testing.T) {
	ctx := NewContext()

	assert.False(t, ctx.IsVerbose(), "IsVerbose should be false by default")
	assert.True(t, ctx.NoNotify(), "NoNotify should be true by default")
	assert.False(t, ctx.NoServices(), "NoServices should be false when NoServices=false")
	assert.Empty(t, ctx.ProjectFile(), "ProjectFile should be empty by default")
}

func TestNewContext_WithOptions(t *testing.T) {
	ctx := NewContext(
		WithProjectFile("/path/to/project.yaml"),
		WithVerbose(true),
	)

	assert.Equal(t, "/path/to/project.yaml", ctx.ProjectFile(), "ProjectFile should match")
	assert.True(t, ctx.IsVerbose(), "IsVerbose should be true")
	assert.True(t, ctx.NoNotify(), "NoNotify should be true (default)")
}

func TestNewContext_MultipleOptions(t *testing.T) {
	ctx := NewContext(
		WithNoServices(true),
	)

	assert.True(t, ctx.NoServices(), "NoServices should be true when NoServices=true")
}

// TestNewContext_OutputDebugFollowsVerbose asserts that WithVerbose reaches the
// output client rather than being overwritten by a hardcoded debugging flag.
func TestNewContext_OutputDebugFollowsVerbose(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		want    bool
	}{
		{name: "verbose enables debug output", verbose: true, want: true},
		{name: "quiet suppresses debug output", verbose: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			require.NoError(t, err)
			old := os.Stdout
			os.Stdout = w
			defer func() { os.Stdout = old }()

			ctx := NewContext(WithVerbose(tt.verbose))
			ctx.Output().Debug("debug message")

			w.Close()

			var buf bytes.Buffer
			_, err = io.Copy(&buf, r)
			require.NoError(t, err)
			r.Close()

			assert.Equal(t, tt.want, buf.Len() > 0, "debug output presence")
		})
	}
}
