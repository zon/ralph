package workspace

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientChangeDirectory(t *testing.T) {
	safeDir := t.TempDir()
	require.NoError(t, os.Chdir(safeDir))

	t.Run("empty path returns nil and does not change cwd", func(t *testing.T) {
		defer func() { require.NoError(t, os.Chdir(safeDir)) }()

		cwdBefore, err := os.Getwd()
		require.NoError(t, err)

		err = (&Client{}).ChangeDirectory("")
		require.NoError(t, err)

		cwdAfter, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, cwdBefore, cwdAfter)
	})

	t.Run("valid directory changes cwd", func(t *testing.T) {
		defer func() { require.NoError(t, os.Chdir(safeDir)) }()

		dir := t.TempDir()

		err := (&Client{}).ChangeDirectory(dir)
		require.NoError(t, err)

		cwd, err := os.Getwd()
		require.NoError(t, err)
		require.Equal(t, dir, cwd)
	})
}
