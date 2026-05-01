package skills

import "errors"

var (
	ErrNotInGitRepo   = errors.New("not in a git repository")
	ErrDiscoveryFailed = errors.New("failed to discover skills from GitHub")
	ErrFetchFailed     = errors.New("failed to fetch skill content")
	ErrInstallFailed   = errors.New("failed to install skills")
)