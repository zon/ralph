package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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
	// This would make a GitHub API call to GET /repos/{owner}/{repo}/installation
	// For now, we'll return a placeholder implementation
	// TODO: Implement actual GitHub API call
	return 0, fmt.Errorf("GetInstallationID not implemented")
}

// GetInstallationToken retrieves an installation access token for a GitHub App
func GetInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	// This would make a GitHub API call to POST /app/installations/{id}/access_tokens
	// For now, we'll return a placeholder implementation
	// TODO: Implement actual GitHub API call
	return "", fmt.Errorf("GetInstallationToken not implemented")
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
