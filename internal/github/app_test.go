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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAppJWT(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate test RSA key")

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	appID := "12345"

	token, err := GenerateAppJWT(appID, privateKeyPEM)
	require.NoError(t, err, "GenerateAppJWT should not fail")
	assert.NotEmpty(t, token, "GenerateAppJWT should return non-empty token")

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &privateKey.PublicKey, nil
	})

	require.NoError(t, err, "failed to parse generated JWT")
	assert.True(t, parsedToken.Valid, "generated JWT should be valid")

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	require.True(t, ok, "failed to parse JWT claims")

	issuer, ok := claims["iss"].(string)
	require.True(t, ok, "JWT should have iss claim")
	assert.Equal(t, appID, issuer, "issuer should match appID")

	iat, ok := claims["iat"].(float64)
	require.True(t, ok, "JWT should have iat claim")
	iatTime := time.Unix(int64(iat), 0)
	assert.WithinDuration(t, time.Now(), iatTime, time.Minute, "JWT issued at time should be recent")

	exp, ok := claims["exp"].(float64)
	require.True(t, ok, "JWT should have exp claim")
	expTime := time.Unix(int64(exp), 0)
	iatTime = time.Unix(int64(iat), 0)
	expectedExp := iatTime.Add(10 * time.Minute)
	assert.WithinDuration(t, expectedExp, expTime, time.Second, "JWT expiration should be 10 minutes from iat")
}

func TestGenerateAppJWT_InvalidKey(t *testing.T) {
	invalidPEM := []byte("not a valid PEM")
	_, err := GenerateAppJWT("12345", invalidPEM)
	assert.Error(t, err, "should return error for invalid PEM")

	_, err = GenerateAppJWT("12345", []byte{})
	assert.Error(t, err, "should return error for empty key")

	_, err = GenerateAppJWT("", []byte("test"))
	assert.Error(t, err, "should return error for empty app ID")
}

func TestParsePrivateKey(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "failed to generate test RSA key")

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	parsedKey, err := parsePrivateKey(privateKeyPEM)
	require.NoError(t, err, "parsePrivateKey should not fail for PKCS1")
	assert.Zero(t, parsedKey.D.Cmp(privateKey.D), "parsed PKCS1 key should match original")

	privateKeyPKCS8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err, "failed to marshal PKCS8 key")

	privateKeyPEM8 := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyPKCS8,
	})

	parsedKey8, err := parsePrivateKey(privateKeyPEM8)
	require.NoError(t, err, "parsePrivateKey should not fail for PKCS8")
	assert.Zero(t, parsedKey8.D.Cmp(privateKey.D), "parsed PKCS8 key should match original")

	_, err = parsePrivateKey([]byte("invalid"))
	assert.Error(t, err, "should return error for invalid PEM")

	nonRSAPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: []byte("not a valid private key"),
	})
	_, err = parsePrivateKey(nonRSAPEM)
	assert.Error(t, err, "should return error for non-RSA key")
}

func withIsolatedGitHome(t *testing.T, f func()) {
	t.Helper()

	fakeHome := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", fakeHome)

	f()

	os.Setenv("HOME", origHome)
}

func gitConfigGlobal(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"config", "--global"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git config --global %v failed: %v\n%s", args, err, out)
	}
}

func gitListGlobal(t *testing.T) []string {
	t.Helper()
	out, err := exec.Command("git", "config", "--global", "--list").Output()
	if err != nil {
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

func TestCleanupStaleTokenRewrites(t *testing.T) {
	withIsolatedGitHome(t, func() {
		ctx := context.Background()

		gitConfigGlobal(t,
			"url.https://x-access-token:old-token-1@github.com/.insteadOf",
			"https://github.com/",
		)
		gitConfigGlobal(t,
			"url.https://x-access-token:old-token-2@github.com/.insteadOf",
			"https://github.com/",
		)
		gitConfigGlobal(t, "user.email", "test@example.com")

		before := gitListGlobal(t)
		tokenEntries := 0
		for _, l := range before {
			if strings.HasPrefix(strings.ToLower(l), "url.https://x-access-token:") {
				tokenEntries++
			}
		}
		assert.Equal(t, 2, tokenEntries, "should have 2 stale token entries before cleanup")

		cleanupStaleTokenRewrites(ctx)

		after := gitListGlobal(t)
		for _, l := range after {
			if strings.HasPrefix(strings.ToLower(l), "url.https://x-access-token:") &&
				strings.Contains(strings.ToLower(l), "github.com") {
				t.Errorf("stale token entry not removed: %s", l)
			}
		}

		found := false
		for _, l := range after {
			if l == "user.email=test@example.com" {
				found = true
			}
		}
		assert.True(t, found, "unrelated git config entry should remain")
	})
}

func TestCleanupStaleTokenRewrites_Empty(t *testing.T) {
	withIsolatedGitHome(t, func() {
		ctx := context.Background()

		gitConfigGlobal(t, "user.name", "Test User")

		cleanupStaleTokenRewrites(ctx)

		after := gitListGlobal(t)
		found := false
		for _, l := range after {
			if l == "user.name=Test User" {
				found = true
			}
		}
		assert.True(t, found, "user.name should remain")
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

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	id, err := GetInstallationID(ctx, "test-jwt", "owner", "repo")

	require.NoError(t, err, "GetInstallationID should not return error")
	assert.Equal(t, expectedID, id, "returned ID should match expected")
}

func TestGetInstallationID_ReturnsErrorOnNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden access"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	_, err := GetInstallationID(ctx, "test-jwt", "owner", "repo")

	assert.Error(t, err, "should return error for non-200 status")
	errStr := err.Error()
	assert.Contains(t, errStr, "403", "error should contain 403 status code")
	assert.Contains(t, errStr, "Forbidden access", "error should contain response body")
}

func TestGetInstallationID_ReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	_, err := GetInstallationID(ctx, "test-jwt", "owner", "repo")

	assert.Error(t, err, "should return error for invalid JSON")
}

func TestGetInstallationID_SendsCorrectHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		accept := r.Header.Get("Accept")
		userAgent := r.Header.Get("User-Agent")

		assert.True(t, strings.HasPrefix(auth, "Bearer "), "Authorization should start with 'Bearer '")
		assert.Equal(t, "application/vnd.github+json", accept, "Accept header should match")
		assert.Equal(t, "ralph", userAgent, "User-Agent header should match")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]int64{"id": 12345})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	_, err := GetInstallationID(ctx, "test-jwt-token", "owner", "repo")

	require.NoError(t, err, "GetInstallationID should not return error")
}

func TestGetInstallationToken_ReturnsTokenOn201(t *testing.T) {
	expectedToken := "gho_xxxxxxxxxxxx"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"token": expectedToken})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	token, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	require.NoError(t, err, "GetInstallationToken should not return error")
	assert.Equal(t, expectedToken, token, "returned token should match expected")
}

func TestGetInstallationToken_ReturnsErrorOnNon201(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	_, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	assert.Error(t, err, "should return error for non-201 status")
	errStr := err.Error()
	assert.Contains(t, errStr, "400", "error should contain 400 status code")
}

func TestGetInstallationToken_ReturnsErrorOnEmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"token": ""})
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	_, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	assert.Error(t, err, "should return error for empty token")
	errStr := err.Error()
	assert.Contains(t, errStr, "empty", "error should mention empty token")
}

func TestGetInstallationToken_ReturnsErrorOnInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	serverURL := strings.TrimPrefix(server.URL, "http://")

	oldClient := httpClient
	httpClient = &http.Client{Transport: &rewriteTransport{old: http.DefaultTransport, serverURL: serverURL}}
	defer func() { httpClient = oldClient }()

	ctx := context.Background()
	_, err := GetInstallationToken(ctx, "test-jwt", 12345678)

	assert.Error(t, err, "should return error for invalid JSON")
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
