package run

// ResolveBaseBranch returns the base branch for a run or prompt: the explicit
// flag, then the current branch when it differs from the branch being created,
// then the configured default branch.
func ResolveBaseBranch(base, current, project, defaultBranch string) string {
	if base != "" {
		return base
	}
	if current != project {
		return current
	}
	return defaultBranch
}
