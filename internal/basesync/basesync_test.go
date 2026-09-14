package basesync

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGit records the synchronization git operations and returns injected
// errors, so tests never touch a real repository.
type fakeGit struct {
	fetchErr      error
	needsMerge    bool
	needsMergeErr error
	mergeErr      error

	fetchBranchCalled bool
	needsMergeCalled  bool
	mergeCalled       bool
	abortMergeCalled  bool
	lastFetchedBranch string
	lastMergedBranch  string
}

func (f *fakeGit) FetchBranch(branch string) error {
	f.fetchBranchCalled = true
	f.lastFetchedBranch = branch
	return f.fetchErr
}

func (f *fakeGit) NeedsMerge(branch string) (bool, error) {
	f.needsMergeCalled = true
	return f.needsMerge, f.needsMergeErr
}

func (f *fakeGit) Merge(branch string) error {
	f.mergeCalled = true
	f.lastMergedBranch = branch
	return f.mergeErr
}

func (f *fakeGit) AbortMerge() error {
	f.abortMergeCalled = true
	return nil
}

// fakeAI records the conflict resolutions and returns an injected error.
type fakeAI struct {
	resolveErr    error
	resolveCalled bool
	lastBase      string
	lastProject   string
}

func (f *fakeAI) ResolveMergeConflicts(baseBranch, projectBranch string) error {
	f.resolveCalled = true
	f.lastBase = baseBranch
	f.lastProject = projectBranch
	return f.resolveErr
}

// fakeOutput records the warnings synchronization logs.
type fakeOutput struct {
	warnings []string
}

func (f *fakeOutput) Warnf(format string, a ...any) {
	f.warnings = append(f.warnings, fmt.Sprintf(format, a...))
}

func TestSync(t *testing.T) {
	tests := []struct {
		name           string
		base           string
		inWorktree     bool
		git            *fakeGit
		ai             *fakeAI
		wantMerged     bool
		wantErr        bool
		wantFetch      bool
		wantNeedsMerge bool
		wantMerge      bool
		wantAbort      bool
		wantResolve    bool
		wantWarnings   int
		wantMergedRef  string
	}{
		{
			name: "no base branch skips synchronization",
			base: "",
		},
		{
			name:           "up-to-date base branch fetches without merging",
			base:           "main",
			git:            &fakeGit{needsMerge: false},
			wantFetch:      true,
			wantNeedsMerge: true,
			wantMerge:      false,
		},
		{
			name:           "clean merge reports a merge",
			base:           "main",
			git:            &fakeGit{needsMerge: true},
			wantFetch:      true,
			wantNeedsMerge: true,
			wantMerge:      true,
			wantMerged:     true,
			wantMergedRef:  "main",
		},
		{
			name:         "fetch failure warns and continues without merging",
			base:         "main",
			git:          &fakeGit{fetchErr: errors.New("fetch boom")},
			wantFetch:    true,
			wantWarnings: 1,
		},
		{
			name:           "needs merge error is returned",
			base:           "main",
			git:            &fakeGit{needsMergeErr: errors.New("needs merge boom")},
			wantFetch:      true,
			wantNeedsMerge: true,
			wantErr:        true,
		},
		{
			name:           "conflict is aborted and resolved by the agent",
			base:           "main",
			git:            &fakeGit{needsMerge: true, mergeErr: errors.New("conflict")},
			ai:             &fakeAI{},
			wantFetch:      true,
			wantNeedsMerge: true,
			wantMerge:      true,
			wantAbort:      true,
			wantResolve:    true,
			wantMerged:     true,
			wantMergedRef:  "main",
		},
		{
			name:           "failed conflict resolution returns the error",
			base:           "main",
			git:            &fakeGit{needsMerge: true, mergeErr: errors.New("conflict")},
			ai:             &fakeAI{resolveErr: errors.New("resolve boom")},
			wantFetch:      true,
			wantNeedsMerge: true,
			wantMerge:      true,
			wantAbort:      true,
			wantResolve:    true,
			wantErr:        true,
		},
		{
			name:           "worktree merges the fetched remote base",
			base:           "main",
			inWorktree:     true,
			git:            &fakeGit{needsMerge: true},
			wantFetch:      true,
			wantNeedsMerge: true,
			wantMerge:      true,
			wantMerged:     true,
			wantMergedRef:  "origin/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := tt.git
			if git == nil {
				git = &fakeGit{}
			}
			ai := tt.ai
			if ai == nil {
				ai = &fakeAI{}
			}
			out := &fakeOutput{}

			merged, err := Sync(git, ai, out, tt.base, "loop-fmt", tt.inWorktree)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantMerged, merged, "the merge report")
			assert.Equal(t, tt.wantFetch, git.fetchBranchCalled, "whether the base branch is fetched")
			assert.Equal(t, tt.wantNeedsMerge, git.needsMergeCalled, "whether the base branch containment is checked")
			assert.Equal(t, tt.wantMerge, git.mergeCalled, "whether a merge is attempted")
			assert.Equal(t, tt.wantAbort, git.abortMergeCalled, "whether the conflicting merge is aborted")
			assert.Equal(t, tt.wantResolve, ai.resolveCalled, "whether the agent resolves conflicts")
			assert.Len(t, out.warnings, tt.wantWarnings, "the number of warnings logged")
			if tt.wantFetch {
				assert.Equal(t, tt.base, git.lastFetchedBranch, "the fetched branch")
			}
			if tt.wantMergedRef != "" {
				assert.Equal(t, tt.wantMergedRef, git.lastMergedBranch, "the merged ref")
			}
			if tt.wantWarnings > 0 {
				assert.Contains(t, out.warnings[0], tt.base, "the warning names the base branch")
			}
		})
	}
}

// TestSyncConflictResolutionReceivesRefs asserts the agent is handed the same
// base ref that conflicted and the loop branch being synchronized.
func TestSyncConflictResolutionReceivesRefs(t *testing.T) {
	git := &fakeGit{needsMerge: true, mergeErr: errors.New("conflict")}
	ai := &fakeAI{}

	_, err := Sync(git, ai, &fakeOutput{}, "main", "loop-fmt", false)

	require.NoError(t, err)
	assert.Equal(t, "main", ai.lastBase)
	assert.Equal(t, "loop-fmt", ai.lastProject)
}
