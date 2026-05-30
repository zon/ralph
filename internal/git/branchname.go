package git

func BranchName(slug string) string {
	return SanitizeBranchName(slug)
}
