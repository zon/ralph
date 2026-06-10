package webhookconfig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

// sign returns a valid X-Hub-Signature-256 header value for body using secret.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidateSignature(t *testing.T) {
	secret := "mysecret"
	body := []byte(`{"test":"value"}`)
	validSig := sign(body, secret)

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{"valid signature", validSig, true},
		{"missing signature", "", false},
		{"wrong prefix", "sha1=" + validSig[7:], false},
		{"tampered body", "sha256=000000", false},
		{"wrong secret", sign(body, "wrong"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateSignature(body, secret, tc.signature)
			assert.Equal(t, tc.want, got)
		})
	}
}
