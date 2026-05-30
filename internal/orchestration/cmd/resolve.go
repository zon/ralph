package cmd

func resolveBaseBranch(base, current, projectBranch, defaultBranch string) string {
	if base != "" {
		return base
	}
	if current != projectBranch {
		return current
	}
	return defaultBranch
}

func resolveMaxIterations(cfgMax, flagMax int) int {
	if flagMax != 0 {
		return flagMax
	}
	return cfgMax
}
