package setskills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	skillsPath   = ".claude/skills"
	ralphRawBase = "https://raw.githubusercontent.com/zon/ralph/refs/heads"
	ralphAPIURL  = "https://api.github.com/repos/zon/ralph/contents/.claude/skills"
)

type githubEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func DiscoverSkills(branch string) ([]string, error) {
	url := fmt.Sprintf("%s?ref=%s", ralphAPIURL, branch)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var entries []githubEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	var skills []string
	for _, entry := range entries {
		if entry.Type == "dir" && strings.HasPrefix(entry.Name, "ralph-") {
			skills = append(skills, entry.Name)
		}
	}

	return skills, nil
}

func FetchSkillContents(skills []string, branch string) (map[string]string, error) {
	contents := make(map[string]string)

	for _, skill := range skills {
		url := fmt.Sprintf("%s/%s/.claude/skills/%s/SKILL.md", ralphRawBase, branch, skill)
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch skill %s: %w", skill, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch skill %s: status %d", skill, resp.StatusCode)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read skill %s content: %w", skill, err)
		}

		contents[skill] = string(data)
	}

	return contents, nil
}

func RewriteLinks(contents map[string]string, branch string) map[string]string {
	rewritten := make(map[string]string)
	baseURL := fmt.Sprintf("%s/%s/", ralphRawBase, branch)

	for skill, content := range contents {
		rewritten[skill] = rewriteContentLinks(content, baseURL)
	}

	return rewritten
}

var linkRegex = regexp.MustCompile(`\[([^\]]*)\]\(([^)]*)\)`)

func rewriteContentLinks(content, baseURL string) string {
	result := linkRegex.ReplaceAllStringFunc(content, func(match string) string {
		idx := strings.Index(match, "](")
		if idx == -1 {
			return match
		}
		link := match[idx+2 : len(match)-1]

		if !strings.Contains(link, "://") {
			link = resolveRelativeLink(link, baseURL)
		} else if strings.HasPrefix(link, "https://raw.githubusercontent.com/zon/ralph/refs/heads/") {
			for _, oldBranch := range []string{"main", "master"} {
				oldPrefix := "https://raw.githubusercontent.com/zon/ralph/refs/heads/" + oldBranch + "/"
				if strings.HasPrefix(link, oldPrefix) {
					link = baseURL + link[len(oldPrefix):]
					break
				}
			}
		}

		return fmt.Sprintf("[%s](%s)", match[1:idx], link)
	})
	return result
}

func resolveRelativeLink(link, baseURL string) string {
	cleaned := link
	for strings.HasPrefix(cleaned, "../") {
		cleaned = cleaned[3:]
	}
	return baseURL + cleaned
}

func rewriteLineLinks(line, baseURL string) string {
	return linkRegex.ReplaceAllStringFunc(line, func(match string) string {
		idx := strings.Index(match, "](")
		if idx == -1 {
			return match
		}
		link := match[idx+2 : len(match)-1]

		if !strings.Contains(link, "://") {
			link = resolveRelativeLink(link, baseURL)
		} else if strings.HasPrefix(link, "https://raw.githubusercontent.com/zon/ralph/refs/heads/") {
			for _, oldBranch := range []string{"main", "master"} {
				oldPrefix := "https://raw.githubusercontent.com/zon/ralph/refs/heads/" + oldBranch + "/"
				if strings.HasPrefix(link, oldPrefix) {
					link = baseURL + link[len(oldPrefix):]
					break
				}
			}
		}

		return fmt.Sprintf("[%s](%s)", match[1:idx], link)
	})
}

func RemoveStaleSkills(repoRoot string, skills []string) error {
	skillsDir := filepath.Join(repoRoot, skillsPath)

	existingEntries, err := os.ReadDir(skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read skills directory: %w", err)
	}

	skillSet := make(map[string]bool)
	for _, s := range skills {
		skillSet[s] = true
	}

	for _, entry := range existingEntries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, "ralph-") && !skillSet[name] {
			skillPath := filepath.Join(skillsDir, name)
			if err := os.RemoveAll(skillPath); err != nil {
				return fmt.Errorf("failed to remove stale skill %s: %w", name, err)
			}
		}
	}

	return nil
}

func WriteSkills(repoRoot string, contents map[string]string) error {
	for skill, content := range contents {
		skillDir := filepath.Join(repoRoot, skillsPath, skill)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill directory %s: %w", skill, err)
		}

		skillPath := filepath.Join(skillDir, "SKILL.md")
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write skill %s: %w", skill, err)
		}
	}

	return nil
}