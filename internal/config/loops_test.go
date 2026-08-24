package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_LoopsSectionParsed(t *testing.T) {
	// GIVEN a `.ralph/config.yaml` with a `loops:` section of multiple entries
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
  - slug: test
    steps:
      - run the test suite
      - fix failures
`)

	// THEN each entry's slug and steps list are read correctly
	require.Len(t, cfg.Loops, 2)

	assert.Equal(t, "fmt", cfg.Loops[0].Slug)
	assert.Equal(t, []string{"run gofmt", "run go vet"}, cfg.Loops[0].Steps)

	assert.Equal(t, "test", cfg.Loops[1].Slug)
	assert.Equal(t, []string{"run the test suite", "fix failures"}, cfg.Loops[1].Steps)
}

func TestLoopSteps_MatchingSlugReturnsSteps(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
      - run go vet
`)

	steps, err := cfg.LoopSteps("fmt")
	require.NoError(t, err)
	assert.Equal(t, []string{"run gofmt", "run go vet"}, steps)
}

func TestLoopSteps_PicksRightEntryAmongMultiple(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
  - slug: test
    steps:
      - run the test suite
      - fix failures
  - slug: lint
    steps:
      - run the linter
`)

	steps, err := cfg.LoopSteps("test")
	require.NoError(t, err)
	assert.Equal(t, []string{"run the test suite", "fix failures"}, steps)
}

func TestLoopSteps_NotFoundReturnsError(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	_, err := cfg.LoopSteps("missing")
	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: missing")
}

func TestLoopSteps_NoLoopsSection(t *testing.T) {
	cfg := loadConfigWithContent(t, "defaultBranch: main\n")

	assert.Nil(t, cfg.Loops, "Loops must stay nil when the config has no loops section")

	_, err := cfg.LoopSteps("fmt")
	require.Error(t, err)
	assert.EqualError(t, err, "loop config not found: fmt")
}
