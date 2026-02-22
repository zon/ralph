package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAppJWT(t *testing.T) {
	// Generate a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test RSA key: %v", err)
	}

	// Encode private key to PEM format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	appID := "12345"

	// Test JWT generation
	token, err := GenerateAppJWT(appID, privateKeyPEM)
	if err != nil {
		t.Fatalf("GenerateAppJWT failed: %v", err)
	}

	if token == "" {
		t.Fatal("GenerateAppJWT returned empty token")
	}

	// Parse and verify the JWT
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &privateKey.PublicKey, nil
	})

	if err != nil {
		t.Fatalf("failed to parse generated JWT: %v", err)
	}

	if !parsedToken.Valid {
		t.Fatal("generated JWT is not valid")
	}

	// Verify claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("failed to parse JWT claims")
	}

	// Check issuer
	issuer, ok := claims["iss"].(string)
	if !ok || issuer != appID {
		t.Errorf("expected issuer %q, got %q", appID, issuer)
	}

	// Check issued at time is recent
	if iat, ok := claims["iat"].(float64); ok {
		iatTime := time.Unix(int64(iat), 0)
		if time.Since(iatTime) > time.Minute {
			t.Errorf("JWT issued at time is too old: %v", iatTime)
		}
	} else {
		t.Error("JWT missing iat claim")
	}

	// Check expiration time (should be 10 minutes from issued at)
	if exp, ok := claims["exp"].(float64); ok {
		expTime := time.Unix(int64(exp), 0)
		if iat, ok := claims["iat"].(float64); ok {
			iatTime := time.Unix(int64(iat), 0)
			expectedExp := iatTime.Add(10 * time.Minute)
			if expTime.Sub(expectedExp).Abs() > time.Second {
				t.Errorf("JWT expiration time mismatch: expected ~%v, got %v", expectedExp, expTime)
			}
		}
	} else {
		t.Error("JWT missing exp claim")
	}
}

func TestGenerateAppJWT_InvalidKey(t *testing.T) {
	// Test with invalid PEM data
	invalidPEM := []byte("not a valid PEM")
	_, err := GenerateAppJWT("12345", invalidPEM)
	if err == nil {
		t.Error("expected error for invalid PEM, got nil")
	}

	// Test with empty key
	_, err = GenerateAppJWT("12345", []byte{})
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}

	// Test with empty app ID
	_, err = GenerateAppJWT("", []byte("test"))
	if err == nil {
		t.Error("expected error for empty app ID, got nil")
	}
}

func TestParsePrivateKey(t *testing.T) {
	// Generate a test RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test RSA key: %v", err)
	}

	// Test PKCS1 format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	parsedKey, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		t.Fatalf("parsePrivateKey failed for PKCS1: %v", err)
	}

	if parsedKey.D.Cmp(privateKey.D) != 0 {
		t.Error("parsed PKCS1 key does not match original")
	}

	// Test PKCS8 format (GenerateAppJWT should handle this through parsePrivateKey)
	privateKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal PKCS8 key: %v", err)
	}

	privateKeyPEM8 := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyPKCS8,
	})

	parsedKey8, err := parsePrivateKey(privateKeyPEM8)
	if err != nil {
		t.Fatalf("parsePrivateKey failed for PKCS8: %v", err)
	}

	if parsedKey8.D.Cmp(privateKey.D) != 0 {
		t.Error("parsed PKCS8 key does not match original")
	}

	// Test invalid PEM
	_, err = parsePrivateKey([]byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid PEM, got nil")
	}

	// Test non-RSA key (would need to generate an ECDSA key, but we'll test error path)
	nonRSAPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: []byte("not a valid private key"),
	})
	_, err = parsePrivateKey(nonRSAPEM)
	if err == nil {
		t.Error("expected error for non-RSA key, got nil")
	}
}
