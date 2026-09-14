// Package basesync keeps a working branch current with its base branch. It is
// the shared behavior `ralph run` and `ralph loop` use in every execution mode:
// fetch the base branch and merge it into the working branch before the first
// iteration and again before the pull request is opened.
package basesync

// GitClient is the git operations synchronization performs.
type GitClient interface {
	FetchBranch(branch string) error
	NeedsMerge(branch string) (bool, error)
	Merge(branch string) error
	AbortMerge() error
}

// AIClient resolves a conflicting merge by running the configured agent to
// resolve the conflicts, run the tests, and stage the result.
type AIClient interface {
	ResolveMergeConflicts(baseBranch, projectBranch string) error
}

// OutputClient logs the warning emitted when the base branch cannot be fetched.
type OutputClient interface {
	Warnf(format string, a ...any)
}

// Sync fetches the base branch and merges it into the working branch when the
// base branch is not already contained, before the first iteration. A fetch
// failure is warned about and skipped. A conflicting merge is aborted and
// handed to the configured AI agent to resolve, run tests, and stage; a failed
// resolution is returned so execution stops, and the merge the agent left
// behind is aborted so the repository is not left mid-merge. It reports whether
// a merge was performed.
func Sync(git GitClient, ai AIClient, output OutputClient, base, projectBranch string, inWorktree bool) (bool, error) {
	if base == "" {
		return false, nil
	}
	if err := git.FetchBranch(base); err != nil {
		output.Warnf("Failed to fetch base branch %q: %v", base, err)
		return false, nil
	}
	baseRef := mergeRef(base, inWorktree)
	needsMerge, err := git.NeedsMerge(baseRef)
	if err != nil {
		return false, err
	}
	if !needsMerge {
		return false, nil
	}
	if err := git.Merge(baseRef); err != nil {
		_ = git.AbortMerge()
		if err := ai.ResolveMergeConflicts(baseRef, projectBranch); err != nil {
			_ = git.AbortMerge()
			return false, err
		}
		return true, nil
	}
	return true, nil
}

// mergeRef is the ref synchronization merges. In a worktree the base branch is
// normally checked out in the main checkout, so it cannot be moved without
// changing that checkout; the fetched remote-tracking ref is merged instead.
func mergeRef(base string, inWorktree bool) string {
	if inWorktree {
		return "origin/" + base
	}
	return base
}
