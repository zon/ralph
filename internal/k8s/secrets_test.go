package k8s

import (
	"strings"
	"testing"
)

func TestGenerateSSHKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("GenerateSSHKeyPair() failed: %v", err)
	}

	// Verify private key is in PEM format
	if !strings.Contains(privateKey, "BEGIN") || !strings.Contains(privateKey, "PRIVATE KEY") {
		t.Errorf("Private key doesn't appear to be in PEM format")
	}

	// Verify public key is in OpenSSH format
	if !strings.HasPrefix(publicKey, "ssh-") {
		t.Errorf("Public key doesn't appear to be in OpenSSH format, got: %s", publicKey)
	}

	// Verify keys are non-empty
	if len(privateKey) == 0 {
		t.Error("Private key is empty")
	}

	if len(publicKey) == 0 {
		t.Error("Public key is empty")
	}

	// Verify keys are different each time
	privateKey2, publicKey2, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("Second GenerateSSHKeyPair() failed: %v", err)
	}

	if privateKey == privateKey2 {
		t.Error("Generated same private key twice - should be random")
	}

	if publicKey == publicKey2 {
		t.Error("Generated same public key twice - should be random")
	}
}

func TestSecretNames(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "github secret name",
			constant: GitHubSecretName,
			expected: "github-credentials",
		},
		{
			name:     "opencode secret name",
			constant: OpenCodeSecretName,
			expected: "opencode-credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.constant)
			}
		})
	}
}

func TestSSHKeyFormat(t *testing.T) {
	privateKey, publicKey, err := GenerateSSHKeyPair()
	if err != nil {
		t.Fatalf("GenerateSSHKeyPair() failed: %v", err)
	}

	// Test private key format more thoroughly
	if !strings.Contains(privateKey, "-----BEGIN") {
		t.Error("Private key missing BEGIN marker")
	}
	if !strings.Contains(privateKey, "-----END") {
		t.Error("Private key missing END marker")
	}

	// Test public key format
	parts := strings.Fields(publicKey)
	if len(parts) < 2 {
		t.Errorf("Public key should have at least 2 parts (type and key), got %d parts", len(parts))
	}

	keyType := parts[0]
	if !strings.HasPrefix(keyType, "ssh-") {
		t.Errorf("Public key type should start with 'ssh-', got %q", keyType)
	}

	// Verify the key part is base64 (should be alphanumeric + / + =)
	if len(parts) > 1 {
		keyPart := parts[1]
		if len(keyPart) == 0 {
			t.Error("Public key data part is empty")
		}
		// Base64 characters
		for _, ch := range keyPart {
			if !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
				(ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=') {
				t.Errorf("Public key contains invalid base64 character: %c", ch)
				break
			}
		}
	}
}
