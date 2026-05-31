package cmd

import (
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewSeedFlag(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--seed", "42"})
	require.NoError(t, err)
	assert.Equal(t, int64(42), cmd.Review.Run.Seed)
}

func TestReviewFollowFlag(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--follow"})
	require.NoError(t, err)
	assert.True(t, cmd.Review.Run.Follow)
}

func TestReviewFollowFlagShort(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "-f"})
	require.NoError(t, err)
	assert.True(t, cmd.Review.Run.Follow)
}

func TestReviewFilterFlag(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--filter", "myfilter"})
	require.NoError(t, err)
	assert.Equal(t, "myfilter", cmd.Review.Run.Filter)
}

func TestReviewFilterFlagShort(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--filter", "substring"})
	require.NoError(t, err)
	assert.Equal(t, "substring", cmd.Review.Run.Filter)
}

func TestReviewOneFlag(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--one"})
	require.NoError(t, err)
	assert.True(t, cmd.Review.Run.One)
}

func TestReviewRunSubcommandExplicit(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "run", "--seed", "42"})
	require.NoError(t, err)
	assert.Equal(t, int64(42), cmd.Review.Run.Seed)
}

func TestReviewRunSubcommandExplicitAllFlags(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "run", "--model", "gpt-4", "--base", "develop", "--local", "--verbose", "--seed", "123", "--filter", "test", "--one"})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", cmd.Review.Run.Model)
	assert.Equal(t, "develop", cmd.Review.Run.Base)
	assert.True(t, cmd.Review.Run.Local)
	assert.True(t, cmd.Review.Run.Verbose)
	assert.Equal(t, int64(123), cmd.Review.Run.Seed)
	assert.Equal(t, "test", cmd.Review.Run.Filter)
	assert.True(t, cmd.Review.Run.One)
}

func TestReviewDefaultSubcommandWithArgs(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--seed", "99"})
	require.NoError(t, err)
	assert.Equal(t, int64(99), cmd.Review.Run.Seed)
}

func TestReviewDefaultSubcommandWithArgsAllFlags(t *testing.T) {
	cmd := &Cmd{}
	parser, err := kong.New(cmd,
		kong.Name("ralph"),
		kong.Exit(func(int) {}),
	)
	require.NoError(t, err)
	_, err = parser.Parse([]string{"review", "--model", "claude", "--base", "main", "--local", "--verbose", "--seed", "456", "--filter", "filtertext", "--one"})
	require.NoError(t, err)
	assert.Equal(t, "claude", cmd.Review.Run.Model)
	assert.Equal(t, "main", cmd.Review.Run.Base)
	assert.True(t, cmd.Review.Run.Local)
	assert.True(t, cmd.Review.Run.Verbose)
	assert.Equal(t, int64(456), cmd.Review.Run.Seed)
	assert.Equal(t, "filtertext", cmd.Review.Run.Filter)
	assert.True(t, cmd.Review.Run.One)
}
