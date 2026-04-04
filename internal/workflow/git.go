package workflow

import (
	"strings"
)

// toHTTPSURL converts a GitHub SSH remote URL to HTTPS.
// SSH format: git@github.com:owner/repo.git -> https://github.com/owner/repo.git
// HTTPS URLs are returned unchanged.
func toHTTPSURL(remoteURL string) string {
	if strings.HasPrefix(remoteURL, "git@github.com:") {
		return "https://github.com/" + strings.TrimPrefix(remoteURL, "git@github.com:")
	}
	return remoteURL
}
