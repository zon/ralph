package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zon/ralph/internal/github"
)

// Run executes the github-token command (implements kong.Run interface)
func (g *GithubTokenCmd) Run() error {
	ctx := context.Background()

	// Autodetect owner and repo from git remote if not provided
	owner := g.Owner
	repo := g.Repo
	if owner == "" || repo == "" {
		detectedOwner, detectedRepo, err := github.GetRepo(ctx)
		if err != nil {
			return fmt.Errorf("failed to autodetect repository from git remote: %w", err)
		}
		if owner == "" {
			owner = detectedOwner
		}
		if repo == "" {
			repo = detectedRepo
		}
	}

	if owner == "" {
		return fmt.Errorf("repository owner is required (use --owner flag or ensure git remote is configured)")
	}
	if repo == "" {
		return fmt.Errorf("repository name is required (use --repo flag or ensure git remote is configured)")
	}

	// Read app ID from secrets directory
	appIDPath := filepath.Join(g.SecretsDir, "app-id")
	appIDBytes, err := os.ReadFile(appIDPath)
	if err != nil {
		return fmt.Errorf("failed to read app ID from %s: %w", appIDPath, err)
	}
	appID := string(appIDBytes)
	if appID == "" {
		return fmt.Errorf("app ID is empty in %s", appIDPath)
	}

	// Read private key from secrets directory
	privateKeyPath := filepath.Join(g.SecretsDir, "private-key")
	privateKeyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key from %s: %w", privateKeyPath, err)
	}
	if len(privateKeyBytes) == 0 {
		return fmt.Errorf("private key is empty in %s", privateKeyPath)
	}

	// Generate JWT
	jwtToken, err := github.GenerateAppJWT(appID, privateKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %w", err)
	}

	// Get installation ID
	installationID, err := github.GetInstallationID(ctx, jwtToken, owner, repo)
	if err != nil {
		return fmt.Errorf("failed to get installation ID: %w", err)
	}

	// Get installation token
	installationToken, err := github.GetInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return fmt.Errorf("failed to get installation token: %w", err)
	}

	// Print token to stdout
	fmt.Print(installationToken)
	return nil
}
