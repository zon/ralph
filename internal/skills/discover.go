package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const githubAPI = "https://api.github.com/repos/zon/ralph/contents/.claude/skills"

var httpClient *http.Client

func HTTPClient() *http.Client {
	return httpClient
}

func SetHTTPClient(c *http.Client) {
	httpClient = c
}

type GitHubContent struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url,omitempty"`
}

func Discover(branch string) ([]string, error) {
	url := fmt.Sprintf("%s?ref=%s", githubAPI, branch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ralph")

	client := httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiscoveryFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrDiscoveryFailed, resp.StatusCode, string(body))
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %v", ErrDiscoveryFailed, err)
	}

	var skills []string
	for _, c := range contents {
		if c.Type == "dir" && strings.HasPrefix(c.Name, "ralph-") {
			skills = append(skills, c.Name)
		}
	}

	return skills, nil
}