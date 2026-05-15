package skills

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type Skill struct {
	Name    string
	Content string
}

func FetchAll(branch string, names []string) ([]Skill, error) {
	var skills []Skill
	for _, name := range names {
		content, err := fetchSkill(branch, name)
		if err != nil {
			return nil, err
		}
		skills = append(skills, Skill{Name: name, Content: content})
	}
	return skills, nil
}

func fetchSkill(branch, name string) (string, error) {
	url := fmt.Sprintf(
		"https://raw.githubusercontent.com/%s/%s/refs/heads/%s/.claude/skills/%s/SKILL.md",
		repoOwner, repoName, branch, name,
	)

	resp, err := httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch skill %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch skill %s: status %d", name, resp.StatusCode)
	}

	buf := new(strings.Builder)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return "", fmt.Errorf("failed to read skill %s content: %w", name, err)
	}

	return rewriteLinks(buf.String(), branch), nil
}

var rawGitHubURLRegex = regexp.MustCompile(`https://raw\.githubusercontent\.com/zon/ralph/refs/heads/([^/]+)(/[^)]+)`)
var relativeLinkRegex = regexp.MustCompile(`\]\(([^)]+)\)`)

func rewriteLinks(content, branch string) string {
	rewritten := rawGitHubURLRegex.ReplaceAllStringFunc(content, func(match string) string {
		if strings.HasPrefix(match, "https://raw.githubusercontent.com/zon/ralph/") {
			return rawGitHubURLRegex.ReplaceAllString(match, "https://raw.githubusercontent.com/zon/ralph/refs/heads/"+branch+"$2")
		}
		return match
	})
	return relativeLinkRegex.ReplaceAllStringFunc(rewritten, func(match string) string {
		before := match[:strings.Index(match, "(")]
		linkStart := strings.Index(match, "(") + 1
		linkEnd := strings.LastIndex(match, ")")
		link := match[linkStart:linkEnd]
		if !strings.HasPrefix(link, "http://") && !strings.HasPrefix(link, "https://") && !strings.HasPrefix(link, "//") {
			return before + "(https://raw.githubusercontent.com/zon/ralph/refs/heads/" + branch + "/" + link + ")"
		}
		return match
	})
}