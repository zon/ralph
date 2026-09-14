package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
)

// TestLoopRunSyncsBranchItWasCreatedFrom asserts the wired local loop resolves
// the branch it was created from through the git client and merges it into the
// loop branch before the first iteration.
func TestLoopRunSyncsBranchItWasCreatedFrom(t *testing.T) {
	writeLoopConfig(t, `loops:
  - slug: fmt
    steps:
      - run gofmt
`)

	git := &fakeGitClient{currentBranch: "main", needsMerge: true}
	cmd := &LoopCmd{
		Mode:         config.ModeLocal,
		Slug:         "fmt",
		slugProposer: &fakeSlugProposer{slug: "should-not-be-used"},
		aiClient:     &fakeAIClient{},
		reportReader: &fakeReportReader{content: "NOTHING_TO_DO"},
		gitClient:    git,
		prClient:     &fakePullRequestOpener{},
	}

	err := cmd.Run()

	require.NoError(t, err)
	assert.Equal(t, "main", git.lastFetchedBranch, "the loop fetches the branch the loop branch was created from")
	assert.Equal(t, "main", git.lastMergedBranch, "local mode merges the local base branch into the loop branch")
}
