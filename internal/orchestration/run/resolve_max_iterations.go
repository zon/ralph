package run

func resolveMaxIterations(cfgMax, flagMax int) int {
	if flagMax != 0 {
		return flagMax
	}
	return cfgMax
}
