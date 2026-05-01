package skills

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

const rawGitHubBase = "https://raw.githubusercontent.com/zon/ralph/refs/heads/"

var ralphRawURLReplacer = regexp.MustCompile(`https://raw\.githubusercontent\.com/zon/ralph/refs/heads/[^/]+/`)

func RewriteLinks(content string, branch string) string {
	branchBase := rawGitHubBase + branch + "/"

	content = ralphRawURLReplacer.ReplaceAllString(content, branchBase)

	content = expandRelativeLinks(content, branchBase)

	return content
}

func expandRelativeLinks(content, branchBase string) string {
	var result strings.Builder
	result.Grow(len(content))

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if i > 0 {
			result.WriteByte('\n')
		}
		result.WriteString(expandLineRelative(line, branchBase))
	}

	return result.String()
}

func expandLineRelative(line, branchBase string) string {
	var result strings.Builder

	for len(line) > 0 {
		spaceIdx := strings.IndexAny(line, " \t")
		if spaceIdx == -1 {
			result.WriteString(expandPath(line, branchBase))
			break
		}

		result.WriteString(expandPath(line[:spaceIdx], branchBase))
		result.WriteByte(line[spaceIdx])
		line = line[spaceIdx+1:]
	}

	return result.String()
}

func expandPath(token, branchBase string) string {
	if isFilePath(token) && !isURL(token) {
		return branchBase + token
	}
	return token
}

func isFilePath(s string) bool {
	if strings.HasPrefix(s, "/") {
		return false
	}

	dotIdx := strings.LastIndex(s, ".")
	if dotIdx == -1 || dotIdx == len(s)-1 {
		return false
	}

	ext := strings.ToLower(s[dotIdx+1:])
	knownExts := map[string]bool{
		"md": true, "txt": true, "yaml": true, "yml": true,
		"json": true, "go": true, "sh": true, "py": true,
		"ts": true, "js": true, "html": true, "css": true,
	}

	return knownExts[ext]
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func Fetch(skillName string, branch string) (string, error) {
	url := fmt.Sprintf("%s%s/.claude/skills/%s/SKILL.md", rawGitHubBase, branch, skillName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	req.Header.Set("User-Agent", "ralph")

	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w: status %d for skill %s, body: %s", ErrFetchFailed, resp.StatusCode, skillName, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: failed to read response body: %v", ErrFetchFailed, err)
	}

	content := string(body)
	content = RewriteLinks(content, branch)

	return content, nil
}

func FetchAll(skillNames []string, branch string) (map[string]string, error) {
	results := make(map[string]string)

	for _, name := range skillNames {
		content, err := Fetch(name, branch)
		if err != nil {
			return nil, err
		}
		results[name] = content
	}

	return results, nil
}