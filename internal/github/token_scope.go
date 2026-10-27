package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// TokenWorkflowScope reports whether GitHub lists the workflow scope for the
// token. Known is false when GitHub does not report OAuth scopes for the token
// type, such as fine-grained personal access tokens and GitHub App tokens.
func TokenWorkflowScope(ctx context.Context, token string) (present, known bool, err error) {
	if token == "" {
		return false, false, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return false, false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ralph")

	client := httpClient
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return false, false, fmt.Errorf("failed to check token scopes: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, false, fmt.Errorf("failed to check token scopes: GitHub API returned status %d", resp.StatusCode)
	}

	scopes := resp.Header.Values("X-OAuth-Scopes")
	if len(scopes) == 0 {
		return false, false, nil
	}

	return hasScope(scopes, "workflow"), true, nil
}

func hasScope(values []string, want string) bool {
	for _, value := range values {
		for _, scope := range strings.Split(value, ",") {
			if strings.TrimSpace(scope) == want {
				return true
			}
		}
	}
	return false
}
