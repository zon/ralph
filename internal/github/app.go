package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/zon/ralph/internal/git"
)

func init() {
	git.SetAuthConfigurator(&gitAuthConfigurator{})
}

type gitAuthConfigurator struct{}

func (g *gitAuthConfigurator) ConfigureGitAuth(ctx context.Context, owner, repo, secretsDir string) error {
	return ConfigureGitAuth(ctx, owner, repo, secretsDir)
}

func (g *gitAuthConfigurator) DefaultSecretsDir() string {
	return DefaultSecretsDir
}

// defaultHTTPTimeout bounds each GitHub API request made with the fallback
// client, so a stalled connection cannot block token acquisition when the
// caller's context carries no deadline.
const defaultHTTPTimeout = 30 * time.Second

const (
	// appJWTClockSkew backdates a JWT's issue time so a local clock that runs
	// slightly ahead of GitHub's does not produce an "Issued at time is in the
	// future" rejection.
	appJWTClockSkew = 60 * time.Second
	// appJWTLifetime is the maximum lifetime GitHub allows for an app JWT.
	appJWTLifetime = 10 * time.Minute
)

var httpClient *http.Client

// githubHTTPClient returns the configured client, or a default client with a
// bounded timeout when none is configured.
func githubHTTPClient() *http.Client {
	if httpClient != nil {
		return httpClient
	}
	return &http.Client{Timeout: defaultHTTPTimeout}
}

// GenerateAppJWT generates a JWT for GitHub App authentication using RS256 signing
func GenerateAppJWT(appID string, privateKeyPEM []byte) (string, error) {
	if appID == "" {
		return "", fmt.Errorf("app ID cannot be empty")
	}
	if len(privateKeyPEM) == 0 {
		return "", fmt.Errorf("private key cannot be empty")
	}

	privateKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	now := time.Now()
	// Backdate the issue time to tolerate clock drift, then expire one allowed
	// lifetime after it so the token still stays within GitHub's limit.
	issuedAt := now.Add(-appJWTClockSkew)
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(issuedAt),
		ExpiresAt: jwt.NewNumericDate(issuedAt.Add(appJWTLifetime)),
	}

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

	client := githubHTTPClient()
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

	client := githubHTTPClient()
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
func cleanupStaleTokenRewrites(ctx context.Context) error {
	out, err := git.ConfigList(true)
	if err != nil {
		return nil
	}

	for _, line := range strings.Split(out, "\n") {
		lower := strings.ToLower(line)
		if !strings.HasPrefix(lower, "url.https://x-access-token:") || !strings.Contains(lower, "github.com") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		if err := git.ConfigUnset(true, key); err != nil {
			return fmt.Errorf("failed to remove stale x-access-token rewrite: %w", err)
		}
	}
	return nil
}

// ConfigureGitAuth configures git HTTPS authentication and the gh CLI from
// credentials in secretsDir. GitHub App credentials are preferred: when both
// app-id and private-key are mounted, they are exchanged for a short-lived
// installation token. Otherwise the token stored in the token file is used, so
// a token-only secret is sufficient for remote runs.
//
// Any previously set x-access-token insteadOf rewrites for github.com are
// removed before the new entry is written. This prevents stale (expired) tokens
// from being chosen by git when multiple rewrites match the same URL.
func ConfigureGitAuth(ctx context.Context, owner, repo, secretsDir string) error {
	token, err := ResolveAuthToken(ctx, owner, repo, secretsDir)
	if err != nil {
		return err
	}

	if err := configureGitAuth(ctx, token); err != nil {
		return err
	}

	if err := authenticateGHCLI(ctx, token); err != nil {
		return err
	}

	return nil
}

// ResolveAuthToken returns the token to use for git HTTPS authentication and
// gh CLI authentication from credentials mounted in secretsDir. When both
// app-id and private-key are mounted, the GitHub App credentials are preferred
// and exchanged for a short-lived installation token. Otherwise the token
// stored in the token file is returned, so a token-only secret is sufficient.
func ResolveAuthToken(ctx context.Context, owner, repo, secretsDir string) (string, error) {
	if !appCredentialsPresent(secretsDir) {
		return readStoredToken(secretsDir)
	}

	owner, repo, err := resolveRepoDetails(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	appID, privateKeyBytes, err := readAppCredentials(secretsDir)
	if err != nil {
		return "", err
	}

	return obtainInstallationToken(ctx, owner, repo, appID, privateKeyBytes)
}

// appCredentialsPresent reports whether both the app-id and private-key files
// are mounted at the secrets directory.
func appCredentialsPresent(secretsDir string) bool {
	for _, name := range []string{"app-id", "private-key"} {
		if _, err := os.Stat(filepath.Join(secretsDir, name)); err != nil {
			return false
		}
	}
	return true
}

// readStoredToken reads the token stored at the token file in secretsDir.
func readStoredToken(secretsDir string) (string, error) {
	tokenPath := filepath.Join(secretsDir, "token")
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token from %s: %w", tokenPath, err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return "", fmt.Errorf("token is empty in %s", tokenPath)
	}
	return token, nil
}

func resolveRepoDetails(ctx context.Context, owner, repo string) (string, string, error) {
	if owner == "" || repo == "" {
		detected, err := GetRepo(ctx)
		if err != nil {
			return "", "", fmt.Errorf("failed to autodetect repository from git remote: %w", err)
		}
		if owner == "" {
			owner = detected.Owner
		}
		if repo == "" {
			repo = detected.Name
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

// GenerateInstallationToken returns the token to use for git authentication
// from secretsDir, preferring GitHub App credentials when both app-id and
// private-key are mounted and falling back to the stored token file.
func GenerateInstallationToken(ctx context.Context, owner, repo, secretsDir string) (string, error) {
	return ResolveAuthToken(ctx, owner, repo, secretsDir)
}

// ConfigureTokenAuth configures git to authenticate HTTPS requests with the
// given token and authenticates the gh CLI (gh auth login --with-token).
func ConfigureTokenAuth(ctx context.Context, token string) error {
	if err := configureGitAuth(ctx, token); err != nil {
		return err
	}
	return authenticateGHCLI(ctx, token)
}

func ValidateAppCredentials(ctx context.Context, keyPath, appID string) error {
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}
	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key file is empty")
	}

	repo, err := GetRepo(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect GitHub repository: %w", err)
	}

	jwtToken, err := GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT for validation: %w", err)
	}

	installationID, err := GetInstallationID(ctx, jwtToken, repo.Owner, repo.Name)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	_, err = GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	return nil
}

func ReadGitHubAppCredentials(keyPath string) ([]byte, error) {
	privateKeyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}
	if len(privateKeyBytes) == 0 {
		return nil, fmt.Errorf("private key file is empty")
	}
	return privateKeyBytes, nil
}

func configureGitAuth(ctx context.Context, installationToken string) error {
	if err := cleanupStaleTokenRewrites(ctx); err != nil {
		return err
	}

	insteadOfKey := "url.https://x-access-token:" + installationToken + "@github.com/.insteadOf"
	if err := git.Config(true, insteadOfKey, "https://github.com/"); err != nil {
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
		// The gh CLI is optional: token-only flows authenticate git over HTTPS
		// without it, so a missing binary is not a failure.
		if errors.Is(err, exec.ErrNotFound) {
			return nil
		}
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
