package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
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

// withIsolatedGitHome runs f with HOME pointing to a fresh temp directory so
// that git config --global operates on an isolated ~/.gitconfig and does not
// touch or read the developer's real git configuration.
func withIsolatedGitHome(t *testing.T, f func()) {
	t.Helper()

	fakeHome := t.TempDir()
	origHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", fakeHome); err != nil {
		t.Fatalf("failed to set HOME: %v", err)
	}
	t.Cleanup(func() { os.Setenv("HOME", origHome) })

	f()
}

// gitConfigGlobal runs "git config --global <args>" with the current HOME.
func gitConfigGlobal(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"config", "--global"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config --global %v failed: %v\n%s", args, err, out)
	}
}

// gitListGlobal returns all "key=value" lines from git config --global --list.
func gitListGlobal(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("git", "config", "--global", "--list").Output()
	if err != nil {
		// An empty config returns exit code 1; treat as empty.
		return nil
	}
	var lines []string
	for _, l := range strings.Split(string(out), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// TestCleanupStaleTokenRewrites verifies that cleanupStaleTokenRewrites removes
// all x-access-token insteadOf entries for github.com from the global git config
// while leaving unrelated entries untouched.
func TestCleanupStaleTokenRewrites(t *testing.T) {
	withIsolatedGitHome(t, func() {
		ctx := context.Background()

		// Seed the isolated global config with two stale token entries and one
		// unrelated insteadOf rule that must survive cleanup.
		gitConfigGlobal(t,
			"url.https://x-access-token:old-token-1@github.com/.insteadOf",
			"https://github.com/",
		)
		gitConfigGlobal(t,
			"url.https://x-access-token:old-token-2@github.com/.insteadOf",
			"https://github.com/",
		)
		// Unrelated entry — should not be removed.
		gitConfigGlobal(t, "user.email", "test@example.com")

		before := gitListGlobal(t)
		tokenEntries := 0
		for _, l := range before {
			if strings.HasPrefix(strings.ToLower(l), "url.https://x-access-token:") {
				tokenEntries++
			}
		}
		if tokenEntries != 2 {
			t.Fatalf("expected 2 stale token entries before cleanup, got %d: %v", tokenEntries, before)
		}

		cleanupStaleTokenRewrites(ctx)

		after := gitListGlobal(t)
		for _, l := range after {
			if strings.HasPrefix(strings.ToLower(l), "url.https://x-access-token:") &&
				strings.Contains(strings.ToLower(l), "github.com") {
				t.Errorf("stale token entry not removed: %s", l)
			}
		}

		// Unrelated entry must still be present.
		found := false
		for _, l := range after {
			if l == "user.email=test@example.com" {
				found = true
			}
		}
		if !found {
			t.Error("unrelated git config entry was incorrectly removed by cleanupStaleTokenRewrites")
		}
	})
}

// TestCleanupStaleTokenRewrites_Empty verifies that cleanupStaleTokenRewrites is
// a no-op when the global git config contains no token rewrites.
func TestCleanupStaleTokenRewrites_Empty(t *testing.T) {
	withIsolatedGitHome(t, func() {
		ctx := context.Background()

		gitConfigGlobal(t, "user.name", "Test User")

		// Should not panic or error.
		cleanupStaleTokenRewrites(ctx)

		after := gitListGlobal(t)
		found := false
		for _, l := range after {
			if l == "user.name=Test User" {
				found = true
			}
		}
		if !found {
			t.Error("user.name was unexpectedly removed")
		}
	})
}

func TestGetInstallationID_ReturnsIDOn200(t *testing.T) {
	expectedID := int64(12345678)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]int64{"id": expectedID})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	id, err := GetInstallationID(ctx, "test-jwt", "owner", "repo")

	if err != nil {
		t.Fatalf("GetInstallationID returned error: %v", err)
	}
	if id != expectedID {
		t.Errorf("expected ID %d, got %d", expectedID, id)
	}
}

func TestGetInstallationID_ReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden access"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	_, err := GetInstallationID(ctx, "test-jwt", "owner", "repo")

	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "403") {
		t.Errorf("error should contain status code 403, got: %s", errStr)
	}
	if !strings.Contains(errStr, "Forbidden access") {
		t.Errorf("error should contain response body, got: %s", errStr)
	}
}

func TestGetInstallationID_ReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	_, err := GetInstallationID(ctx, "test-jwt", "owner", "repo")

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGetInstallationID_SendsCorrectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		accept := r.Header.Get("Accept")
		userAgent := r.Header.Get("User-Agent")

		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization header should start with 'Bearer ', got: %s", auth)
		}
		if accept != "application/vnd.github+json" {
			t.Errorf("Accept header should be 'application/vnd.github+json', got: %s", accept)
		}
		if userAgent != "ralph" {
			t.Errorf("User-Agent header should be 'ralph', got: %s", userAgent)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]int64{"id": 12345})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	_, err := GetInstallationID(ctx, "test-jwt-token", "owner", "repo")

	if err != nil {
		t.Fatalf("GetInstallationID returned error: %v", err)
	}
}

func TestGetInstallationToken_ReturnsTokenOn201(t *testing.T) {
	expectedToken := "gho_xxxxxxxxxxxx"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"token": expectedToken})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	token, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	if err != nil {
		t.Fatalf("GetInstallationToken returned error: %v", err)
	}
	if token != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, token)
	}
}

func TestGetInstallationToken_ReturnsErrorOnNon201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	_, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	if err == nil {
		t.Fatal("expected error for non-201 status, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "400") {
		t.Errorf("error should contain status code 400, got: %s", errStr)
	}
}

func TestGetInstallationToken_ReturnsErrorOnEmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"token": ""})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	_, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "empty") {
		t.Errorf("error should mention empty token, got: %s", errStr)
	}
}

func TestGetInstallationToken_ReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{old: oldTransport, serverURL: serverURL}
	defer func() { http.DefaultTransport = oldTransport }()

	ctx := context.Background()
	_, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

type rewriteTransport struct {
	old       http.RoundTripper
	serverURL string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host == "api.github.com" {
		req = cloneRequest(req)
		req.URL.Scheme = "http"
		req.URL.Host = rt.serverURL
		req.URL.Path = strings.Replace(req.URL.Path, "api.github.com", "", 1)
		req.URL.RawPath = ""
	}
	return rt.old.RoundTrip(req)
}

func cloneRequest(req *http.Request) *http.Request {
	newReq := *req
	newReq.Header = make(http.Header, len(req.Header))
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	return &newReq
}
