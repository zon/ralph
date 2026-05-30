package run

func resolveBaseBranch(base, current, project, defaultBranch string) string {
	if base != "" {
		return base
	}
	if current != project {
		return current
	}
	return defaultBranch
}
