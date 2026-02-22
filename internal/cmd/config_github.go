package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/github"
	"github.com/zon/ralph/internal/k8s"
	"golang.org/x/term"
)

// ConfigGithubCmd configures GitHub credentials for Argo Workflows
type ConfigGithubCmd struct {
	Context   string `help:"Kubernetes context to use (defaults to current context)"`
	Namespace string `help:"Kubernetes namespace to use (defaults to context default or 'default')"`
}

// Run executes the config github command
func (c *ConfigGithubCmd) Run() error {
	ctx := context.Background()

	fmt.Println("Configuring GitHub App credentials for Ralph remote execution...")
	fmt.Println()

	// Load context and namespace with priority: flags > .ralph/config.yaml > kubectl
	kubeContext, namespace, err := loadContextAndNamespace(ctx, c.Context, c.Namespace)
	if err != nil {
		return err
	}

	fmt.Println()

	// Get the current repository name from git remote
	repoName, repoOwner, err := github.GetRepo(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect GitHub repository: %w", err)
	}

	appID := config.DefaultAppID

	// Check if the app is installed on the current repo
	fmt.Println("Checking if GitHub App is installed on the repository...")

	// First, we need to prompt for a temporary token to check installation
	// We'll ask for a personal access token with repo scope
	fmt.Println("To check installation status, we need a GitHub personal access token with 'repo' scope.")
	fmt.Println("This token is only used temporarily and won't be stored.")
	fmt.Println()
	fmt.Print("Enter a GitHub personal access token (with repo scope): ")

	tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	fmt.Println() // Print newline after hidden input

	tempToken := strings.TrimSpace(string(tokenBytes))
	if tempToken == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Check installation using the GitHub API
	installed, err := checkAppInstallation(ctx, tempToken, repoOwner, repoName, appID)
	if err != nil {
		return fmt.Errorf("failed to check app installation: %w", err)
	}

	if !installed {
		fmt.Println("GitHub App is not installed on this repository.")
		fmt.Println()
		fmt.Printf("Installation URL: https://github.com/apps/%s/installations/new\n", "ralph-bot") // TODO: Get app slug from API
		fmt.Println()
		fmt.Print("Press Enter after you have installed the app...")
		fmt.Scanln()

		// Recheck installation
		fmt.Println("Rechecking installation status...")
		installed, err = checkAppInstallation(ctx, tempToken, repoOwner, repoName, appID)
		if err != nil {
			return fmt.Errorf("failed to recheck app installation: %w", err)
		}
		if !installed {
			return fmt.Errorf("GitHub App is still not installed. Please install it and try again.")
		}
	}

	fmt.Println("✓ GitHub App is installed on the repository")
	fmt.Println()

	// Prompt for private key file path
	fmt.Print("Enter path to GitHub App private key (.pem file): ")
	var privateKeyPath string
	fmt.Scanln(&privateKeyPath)
	privateKeyPath = strings.TrimSpace(privateKeyPath)
	if privateKeyPath == "" {
		return fmt.Errorf("private key path cannot be empty")
	}

	// Read private key
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}
	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key file is empty")
	}

	// Validate credentials by generating a test installation token
	fmt.Println("Validating credentials...")
	jwtToken, err := github.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT for validation: %w", err)
	}

	// Get installation ID
	installationID, err := github.GetInstallationID(ctx, jwtToken, repoOwner, repoName)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	// Get installation token (validate it works)
	_, err = github.GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	fmt.Println("✓ Credentials validated successfully")
	fmt.Println()

	// Create or update the Kubernetes secret
	fmt.Printf("Creating/updating Kubernetes secret '%s'...\n", k8s.GitHubSecretName)

	secretData := map[string]string{
		"app-id":      appID,
		"private-key": string(privateKeyBytes),
	}

	if err := k8s.CreateOrUpdateSecret(ctx, k8s.GitHubSecretName, namespace, kubeContext, secretData); err != nil {
		return fmt.Errorf("failed to create/update secret: %w", err)
	}

	fmt.Printf("✓ Secret '%s' created/updated successfully\n", k8s.GitHubSecretName)
	fmt.Println()

	fmt.Printf("Configuration complete! The secret '%s' is ready for use in namespace '%s'.\n", k8s.GitHubSecretName, namespace)
	fmt.Println()
	fmt.Println("Note: GitHub App credentials are not tied to any user account.")

	return nil
}

// checkAppInstallation checks if a GitHub App is installed on a repository
func checkAppInstallation(ctx context.Context, token, owner, repo, appID string) (bool, error) {
	// Use the GitHub API to check if the app is installed
	// We'll make a request to the repository's installation endpoint
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/installation", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ralph")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// App is not installed
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to check app ID
	var result struct {
		AppID int64 `json:"app_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Compare app ID (note: appID from user is string, result.AppID is int64)
	// We need to convert appID string to int64 for comparison
	expectedAppID, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid app ID format: %w", err)
	}

	return result.AppID == expectedAppID, nil
}
