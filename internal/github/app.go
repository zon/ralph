package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var httpClient *http.Client

// GenerateAppJWT generates a JWT for GitHub App authentication using RS256 signing
func GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	if appID == "" {
		return "", fmt.Errorf("app ID cannot be empty")
	}
	if len(privateKeyPEM) == 0 {
		return "", fmt.Errorf("private key cannot be empty")
	}

	// Parse the private key
	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create the JWT claims
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)), // GitHub Apps JWTs expire after 10 minutes
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

// GetInstallationID retrieves the installation ID for a GitHub App in a specific repository
func GetInstallationID(ctx context.Context, jwtToken, owner, repo string) (int64, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/installation", owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ralph")

	client := httpClient
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID int64 `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.ID, nil
}

// GetInstallationToken retrieves an installation access token for a GitHub App
func GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "ralph")

	client := httpClient
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Token == "" {
		return "", fmt.Errorf("installation token is empty in response")
	}

	return result.Token, nil
}

// DefaultSecretsDir is the default directory for GitHub App credentials in containers.
const DefaultSecretsDir = "/secrets/github"

// cleanupStaleTokenRewrites removes any existing x-access-token insteadOf rewrites
// for github.com from the global git config. This prevents stale expired tokens
// from interfering when a fresh token is set: git picks an arbitrary match when
// multiple insteadOf rules rewrite the same URL, so old entries must be purged.
func cleanupStaleTokenRewrites(ctx context.Context) {
	// List all global git config entries and find token-bearing insteadOf keys.
	cmd := exec.CommandContext(ctx, "git", "config", "--global", "--list")
	out, err := cmd.Output()
	if err != nil {
		// Not fatal — if we can't list, we simply skip cleanup.
		return
	}

	for _, line := range strings.Split(string(out), "\n") {
		// Lines look like: url.https://x-access-token:TOKEN@github.com/.insteadof=https://github.com/
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "url.https://x-access-token:") || !strings.Contains(lower, "github.com") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		// The key is everything before the first '='.
		key := parts[0]
		// Unset the key; ignore errors (key may already be gone).
		exec.CommandContext(ctx, "git", "config", "--global", "--unset-all", key).Run() //nolint:errcheck
	}
}

// ConfigureGitAuth fetches a GitHub App installation token from secretsDir and
// configures git globally to authenticate HTTPS requests with it.
// owner and repo are used to look up the installation; if either is empty they
// are autodetected from the git remote.
//
// Any previously set x-access-token insteadOf rewrites for github.com are
// removed before the new entry is written. This prevents stale (expired) tokens
// from being chosen by git when multiple rewrites match the same URL.
func ConfigureGitAuth(ctx context.Context, owner, repo, secretsDir string) error {
	owner, repo, err := resolveRepoDetails(ctx, owner, repo)
	if err != nil {
		return err
	}

	appID, privateKeyBytes, err := readAppCredentials(secretsDir)
	if err != nil {
		return err
	}

	installationToken, err := obtainInstallationToken(ctx, owner, repo, appID, privateKeyBytes)
	if err != nil {
		return err
	}

	if err := configureGitAuth(ctx, installationToken); err != nil {
		return err
	}

	if err := authenticateGHCLI(ctx, installationToken); err != nil {
		return err
	}

	return nil
}

func resolveRepoDetails(ctx context.Context, owner, repo string) (string, string, error) {
	if owner == "" || repo == "" {
		detectedOwner, detectedRepo, err := GetRepo(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to autodetect repository from git remote: %w", err)
		}
		if owner == "" {
			owner = detectedOwner
		}
		if repo == "" {
			repo = detectedRepo
		}
	}

	if owner == "" {
		return "", "", fmt.Errorf("repository owner is required (use --owner flag or ensure git remote is configured)")
	}
	if repo == "" {
		return "", "", fmt.Errorf("repository name is required (use --repo flag or ensure git remote is configured)")
	}

	return owner, repo, nil
}

func readAppCredentials(secretsDir string) (string, []byte, error) {
	appIDPath := filepath.Join(secretsDir, "app-id")
	appIDBytes, err := os.ReadFile(appIDPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read app ID from %s: %w", appIDPath, err)
	}
	appID := strings.TrimSpace(string(appIDBytes))
	if appID == "" {
		return "", nil, fmt.Errorf("app ID is empty in %s", appIDPath)
	}

	privateKeyPath := filepath.Join(secretsDir, "private-key")
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read private key from %s: %w", privateKeyPath, err)
	}
	if len(privateKeyBytes) == 0 {
		return "", nil, fmt.Errorf("private key is empty in %s", privateKeyPath)
	}

	return appID, privateKeyBytes, nil
}

func obtainInstallationToken(ctx context.Context, owner, repo, appID string, privateKeyBytes []byte) (string, error) {
	jwtToken, err := GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	installationID, err := GetInstallationID(ctx, jwtToken, owner, repo)
	if err != nil {
		return "", fmt.Errorf("failed to get installation ID: %w", err)
	}

	installationToken, err := GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token: %w", err)
	}

	return installationToken, nil
}

func configureGitAuth(ctx context.Context, installationToken string) error {
	cleanupStaleTokenRewrites(ctx)

	insteadOfKey := "url.https://x-access-token:" + installationToken + "@github.com/.insteadOf"
	cmd := exec.CommandContext(ctx, "git", "config", "--global", insteadOfKey, "https://github.com/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to configure git HTTPS authentication: %w", err)
	}
	return nil
}

func authenticateGHCLI(ctx context.Context, installationToken string) error {
	loginCmd := exec.CommandContext(ctx, "gh", "auth", "login", "--with-token")
	loginCmd.Stdin = strings.NewReader(installationToken)
	loginCmd.Stdout = os.Stdout
	loginCmd.Stderr = os.Stderr
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to authenticate gh CLI: %w", err)
	}
	return nil
}

// parsePrivateKey parses a PEM-encoded RSA private key
func parsePrivateKey(privateKeyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not an RSA key")
		}
		return rsaKey, nil
	}

	return privateKey, nil
}
