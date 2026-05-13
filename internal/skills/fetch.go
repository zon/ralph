package skills

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Skill struct {
	Name  string
	Body  string
	Link  string
	Paths []string
}

func FetchAll(branch string, names []string) ([]Skill, error) {
	return FetchAllWithClient(branch, names, httpClient)
}

func FetchAllWithClient(branch string, names []string, client *http.Client) ([]Skill, error) {
	var skills []Skill
	for _, name := range names {
		skill, err := fetchOneWithClient(branch, name, client)
		if err != nil {
			return nil, err
		}
		skills = append(skills, *skill)
	}
	return skills, nil
}

func fetchOne(branch, name string) (*Skill, error) {
	return fetchOneWithClient(branch, name, httpClient)
}

func fetchOneWithClient(branch, name string, client *http.Client) (*Skill, error) {
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/zon/ralph/refs/heads/%s/.claude/skills/%s/SKILL.md", branch, name)

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skill %q: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch skill %q: status %d", name, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	rewritten, paths := rewriteLinks(string(body), branch)

	link := rawURL
	if len(paths) > 0 {
		link = paths[0]
	}

	return &Skill{
		Name:  name,
		Body:  rewritten,
		Link:  link,
		Paths: paths,
	}, nil
}

const rawGitHubPrefix = "https://raw.githubusercontent.com/zon/ralph/refs/heads/"

func rewriteLinks(content, branch string) (string, []string) {
	var result []string
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		rewrittenLine, path := rewriteLine(line, branch)
		result = append(result, rewrittenLine)
		if path != "" {
			paths = append(paths, path)
		}
	}
	return strings.Join(result, "\n"), paths
}

func rewriteLine(line, branch string) (string, string) {
	if strings.Contains(line, "http") && strings.Contains(line, rawGitHubPrefix) {
		start := 0
		for {
			idx := strings.Index(line[start:], "http")
			if idx == -1 {
				break
			}
			absStart := start + idx
			remaining := line[absStart:]

			end := len(remaining)
			for i, ch := range remaining {
				if ch == ' ' || ch == ')' || ch == '"' || ch == '\'' || ch == '\n' {
					end = i
					break
				}
			}

			urlStr := remaining[:end]
			if strings.HasPrefix(urlStr, rawGitHubPrefix) {
				path := strings.TrimPrefix(urlStr, rawGitHubPrefix)
				parts := strings.SplitN(path, "/", 2)
				if len(parts) == 2 {
					newURL := rawGitHubPrefix + branch + "/" + parts[1]
					line = line[:absStart] + newURL + remaining[end:]
					start = absStart + len(newURL)
					continue
				}
			}

			start = absStart + 1
		}
		return line, extractURLFromLine(line)
	}
	return rewriteRelativeLink(line, branch)
}

func extractURLFromLine(line string) string {
	idx := strings.Index(line, "](")
	if idx == -1 {
		return ""
	}
	after := line[idx+2:]
	endIdx := len(after)
	for i, ch := range after {
		if ch == ')' || ch == '"' || ch == ' ' || ch == '\n' {
			endIdx = i
			break
		}
	}
	url := after[:endIdx]
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		if strings.HasPrefix(url, rawGitHubPrefix) {
			return url
		}
		return ""
	}
	if strings.Contains(url, ".") && !strings.Contains(url, "http") {
		return rawGitHubPrefix + url
	}
	return ""
}

func rewriteRelativeLink(line, branch string) (string, string) {
	for _, suffix := range []string{".md", ".yaml", ".yml"} {
		pattern := suffix + ")"
		idx := strings.Index(line, pattern)
		if idx == -1 {
			continue
		}
		before := line[:idx+1]
		openIdx := strings.LastIndex(before, "](")
		if openIdx == -1 {
			continue
		}
		link := before[openIdx+2:]
		if !strings.Contains(link, "http") && strings.Contains(link, ".") {
			newLink := rawGitHubPrefix + branch + "/" + link
			newLine := line[:openIdx+2] + newLink + line[idx+len(suffix):]
			path := rawGitHubPrefix + branch + "/" + link + suffix[1:]
			return newLine, path
		}
	}
	return line, ""
}