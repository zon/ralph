package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func GenerateSSHKeyPair() (string, string, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	privKeyBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal private key: %w", err)
	}

	privKeyPEM := pem.EncodeToMemory(privKeyBytes)
	if privKeyPEM == nil {
		return "", "", fmt.Errorf("failed to encode private key to PEM")
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	pubKeyOpenSSH := string(ssh.MarshalAuthorizedKey(sshPubKey))
	pubKeyOpenSSH = strings.TrimSpace(pubKeyOpenSSH)

	return string(privKeyPEM), pubKeyOpenSSH, nil
}
