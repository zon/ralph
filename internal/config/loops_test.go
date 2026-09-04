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
	require.Nil(t, cfg.Loops[0].Max, "an entry without max leaves the cap unset")

	assert.Equal(t, "test", cfg.Loops[1].Slug)
	assert.Equal(t, []string{"run the test suite", "fix failures"}, cfg.Loops[1].Steps)
	require.Nil(t, cfg.Loops[1].Max, "an entry without max leaves the cap unset")
}

func TestLoopConfig_OptionalMaxFieldParsed(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
    max: 30
  - slug: lint
    steps:
      - run the linter
`)

	require.Len(t, cfg.Loops, 2)
	require.NotNil(t, cfg.Loops[0].Max, "the entry's max field is parsed")
	assert.Equal(t, 30, *cfg.Loops[0].Max)
	require.Nil(t, cfg.Loops[1].Max, "an entry without max leaves the cap unset")
}

func TestLoopMax_MatchingSlugReturnsConfiguredMax(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
    max: 30
  - slug: lint
    steps:
      - run the linter
`)

	assert.NotNil(t, cfg.LoopMax("fmt"))
	assert.Equal(t, 30, *cfg.LoopMax("fmt"))
}

func TestLoopMax_EntryWithoutMaxReturnsNil(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	assert.Nil(t, cfg.LoopMax("fmt"))
}

func TestLoopMax_NoMatchingSlugReturnsNil(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	assert.Nil(t, cfg.LoopMax("missing"))
}

func TestLoopMax_NoLoopsSectionReturnsNil(t *testing.T) {
	cfg := loadConfigWithContent(t, "defaultBranch: main\n")

	assert.Nil(t, cfg.LoopMax("fmt"))
}

func TestLoadConfig_NonPositiveLoopMaxRejected(t *testing.T) {
	tests := []struct {
		name    string
		maxYAML string
	}{
		{name: "zero max rejected", maxYAML: "max: 0"},
		{name: "negative max rejected", maxYAML: "max: -1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := "loops:\n  - slug: fmt\n    steps:\n      - run gofmt\n    " + tt.maxYAML + "\n"

			_, err := loadConfigErrorWithContent(t, content)
			require.Error(t, err, "LoadConfig() expected error for a loop entry whose max is zero or negative")
			assert.Contains(t, err.Error(), "fmt", "the error must name the loop slug")
		})
	}
}

func TestLoadConfig_NonPositiveLoopMaxNamesOffendingSlug(t *testing.T) {
	content := `loops:
  - slug: fmt
    steps:
      - run gofmt
    max: 30
  - slug: lint
    steps:
      - run the linter
    max: 0
`

	_, err := loadConfigErrorWithContent(t, content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lint", "the error must name the offending loop slug")
	assert.NotContains(t, err.Error(), "fmt", "the error must not name a valid loop entry")
}

func TestLoadConfig_PositiveLoopMaxAccepted(t *testing.T) {
	cfg := loadConfigWithContent(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
    max: 1
`)

	require.Len(t, cfg.Loops, 1)
	require.NotNil(t, cfg.Loops[0].Max)
	assert.Equal(t, 1, *cfg.Loops[0].Max)
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
