package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	orchestrationRun "github.com/zon/ralph/internal/orchestration/run"
	"github.com/zon/ralph/internal/project"
)

func withItems(n int) *project.Project {
	return &project.Project{Items: project.NewItems(make([]any, n))}
}

func TestExtraIterationsDefaultThirtyPercent(t *testing.T) {
	cfg := &config.RalphConfig{}
	c := &project.Client{}
	assert.Equal(t, 3, c.ExtraIterations(withItems(10), cfg))
}

func TestExtraIterationsRoundsUp(t *testing.T) {
	cfg := &config.RalphConfig{}
	c := &project.Client{}
	assert.Equal(t, 1, c.ExtraIterations(withItems(3), cfg))
}

func TestExtraIterationsUsesConfigValue(t *testing.T) {
	v := 5
	cfg := &config.RalphConfig{ExtraIterations: &v}
	c := &project.Client{}
	assert.Equal(t, 5, c.ExtraIterations(withItems(10), cfg))
}

func TestValidateFile(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "project.yaml")
	require.NoError(t, os.WriteFile(regular, []byte("items: []\n"), 0o644))
	dangling := filepath.Join(dir, "dangling.yaml")
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing-target"), dangling))
	linkToFile := filepath.Join(dir, "linked.yaml")
	require.NoError(t, os.Symlink(regular, linkToFile))

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "regular file", path: regular},
		{name: "symlink to regular file", path: linkToFile},
		{name: "missing file", path: filepath.Join(dir, "absent.yaml"), wantErr: "project file not found"},
		{name: "dangling symlink", path: dangling, wantErr: "project file not found"},
		{name: "directory", path: dir, wantErr: "project file is not a regular file"},
		{name: "empty path", path: "", wantErr: "project file required"},
	}

	c := &project.Client{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateFile(tt.path)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestProjectAdapterImplementsInterfaces(t *testing.T) {
	var _ orchestrationRun.ProjectClient = &project.Client{}
	var _ orchestrationRun.ProjectRepo = &project.Client{}
}
