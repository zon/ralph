package skills

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

var httpClient *http.Client

func HTTPClient() *http.Client {
	return httpClient
}

func SetHTTPClient(client *http.Client) {
	httpClient = client
}

type contentsResponse []struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func Discover(branch string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/zon/ralph/contents/.claude/skills?ref=%s", branch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents contentsResponse
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var names []string
	for _, item := range contents {
		if item.Type == "dir" && strings.HasPrefix(item.Name, "ralph-") {
			names = append(names, item.Name)
		}
	}

	return names, nil
}