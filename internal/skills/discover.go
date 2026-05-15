package skills

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	repoOwner = "zon"
	repoName  = "ralph"
)

type apiEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

var httpClient = http.DefaultClient

func Discover(branch string) ([]string, error) {
	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/contents/.claude/skills?ref=%s",
		repoOwner, repoName, branch,
	)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub Contents API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub Contents API returned status %d", resp.StatusCode)
	}

	var entries []apiEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub Contents API response: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.Type == "dir" && strings.HasPrefix(entry.Name, "ralph-") {
			names = append(names, entry.Name)
		}
	}

	return names, nil
}