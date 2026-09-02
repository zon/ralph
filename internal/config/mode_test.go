package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_ModeField(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "local",
			content: "mode: local\n",
			want:    "local",
		},
		{
			name:    "worktree",
			content: "mode: worktree\n",
			want:    "worktree",
		},
		{
			name:    "remote",
			content: "mode: remote\n",
			want:    "remote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			ralphDir := filepath.Join(tmpDir, ".ralph")
			require.NoError(t, os.Mkdir(ralphDir, 0755))

			configPath := filepath.Join(ralphDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.content), 0644))

			t.Chdir(tmpDir)

			config, err := LoadConfig()
			require.NoError(t, err, "LoadConfig() unexpected error")
			assert.Equal(t, tt.want, config.Mode)
		})
	}
}

func TestLoadConfig_ModeDefaultsToLocal(t *testing.T) {
	tmpDir := t.TempDir()

	ralphDir := filepath.Join(tmpDir, ".ralph")
	require.NoError(t, os.Mkdir(ralphDir, 0755))

	configPath := filepath.Join(ralphDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("defaultBranch: main\n"), 0644))

	t.Chdir(tmpDir)

	config, err := LoadConfig()
	require.NoError(t, err, "LoadConfig() unexpected error")
	assert.Equal(t, "local", config.Mode)
}

func TestLoadConfig_InvalidModeRejected(t *testing.T) {
	tests := []struct {
		name    string
		content string
		errMsg  string
	}{
		{
			name:    "unrecognized value",
			content: "mode: sandbox\n",
			errMsg:  "invalid mode: sandbox (expected local, worktree, or remote)",
		},
		{
			name:    "uppercase value",
			content: "mode: Remote\n",
			errMsg:  "invalid mode: Remote (expected local, worktree, or remote)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			ralphDir := filepath.Join(tmpDir, ".ralph")
			require.NoError(t, os.Mkdir(ralphDir, 0755))

			configPath := filepath.Join(ralphDir, "config.yaml")
			require.NoError(t, os.WriteFile(configPath, []byte(tt.content), 0644))

			t.Chdir(tmpDir)

			_, err := LoadConfig()
			require.Error(t, err, "LoadConfig() expected error for invalid mode")
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestResolveMode(t *testing.T) {
	t.Run("flag overrides configured mode", func(t *testing.T) {
		cfg := &RalphConfig{Mode: "remote"}
		mode, err := cfg.ResolveMode("local")
		require.NoError(t, err)
		assert.Equal(t, "local", mode)
	})

	t.Run("configured mode used when no flag", func(t *testing.T) {
		cfg := &RalphConfig{Mode: "remote"}
		mode, err := cfg.ResolveMode("")
		require.NoError(t, err)
		assert.Equal(t, "remote", mode)
	})

	t.Run("local default when flag and config unset", func(t *testing.T) {
		cfg := &RalphConfig{}
		mode, err := cfg.ResolveMode("")
		require.NoError(t, err)
		assert.Equal(t, "local", mode)
	})

	t.Run("invalid flag value rejected", func(t *testing.T) {
		cfg := &RalphConfig{Mode: "worktree"}
		_, err := cfg.ResolveMode("sandbox")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode: sandbox (expected local, worktree, or remote)")
	})

	t.Run("invalid configured value rejected", func(t *testing.T) {
		cfg := &RalphConfig{Mode: "sandbox"}
		_, err := cfg.ResolveMode("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mode: sandbox (expected local, worktree, or remote)")
	})
}

func TestValidateMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{name: "local is valid", mode: "local"},
		{name: "worktree is valid", mode: "worktree"},
		{name: "remote is valid", mode: "remote"},
		{name: "empty is invalid", mode: "", wantErr: true},
		{name: "unrecognized is invalid", mode: "sandbox", wantErr: true},
		{name: "mixed case is invalid", mode: "Local", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMode(tt.mode)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "expected local, worktree, or remote")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
