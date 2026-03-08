package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSSHKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateSSHKeyPair()
	require.NoError(t, err, "GenerateSSHKeyPair() should not fail")

	assert.True(t, strings.Contains(privateKey, "BEGIN"), "Private key should contain 'BEGIN'")
	assert.True(t, strings.Contains(privateKey, "PRIVATE KEY"), "Private key should contain 'PRIVATE KEY'")
	assert.True(t, strings.HasPrefix(publicKey, "ssh-"), "Public key should start with 'ssh-'")
	assert.NotEmpty(t, privateKey, "Private key should not be empty")
	assert.NotEmpty(t, publicKey, "Public key should not be empty")

	privateKey2, publicKey2, err := GenerateSSHKeyPair()
	require.NoError(t, err, "Second GenerateSSHKeyPair() should not fail")

	assert.NotEqual(t, privateKey, privateKey2, "Private keys should be unique")
	assert.NotEqual(t, publicKey, publicKey2, "Public keys should be unique")
}

func TestSSHKeyFormat(t *testing.T) {
	privateKey, publicKey, err := GenerateSSHKeyPair()
	require.NoError(t, err, "GenerateSSHKeyPair() should not fail")

	assert.Contains(t, privateKey, "-----BEGIN", "Private key should contain BEGIN marker")
	assert.Contains(t, privateKey, "-----END", "Private key should contain END marker")

	parts := strings.Fields(publicKey)
	assert.GreaterOrEqual(t, len(parts), 2, "Public key should have at least 2 parts")

	keyType := parts[0]
	assert.True(t, strings.HasPrefix(keyType, "ssh-"), "Public key type should start with 'ssh-'")

	if len(parts) > 1 {
		keyPart := parts[1]
		assert.NotEmpty(t, keyPart, "Public key data part should not be empty")
		for _, ch := range keyPart {
			assert.True(t, (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
				(ch >= '0' && ch <= '9') || ch == '+' || ch == '/' || ch == '=',
				"Public key should contain valid base64 characters")
		}
	}
}
