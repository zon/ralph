package k8s

import (
	"slices"
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

func TestBuildSecretArgs(t *testing.T) {
	tests := []struct {
		name          string
		secretName    string
		namespace     string
		kubeContext   string
		data          map[string]string
		expectedArgs  []string
		expectContext bool
	}{
		{
			name:        "basic secret with single data entry",
			secretName:  "my-secret",
			namespace:   "my-namespace",
			kubeContext: "",
			data:        map[string]string{"key1": "value1"},
			expectedArgs: []string{
				"create", "secret", "generic", "my-secret",
				"--from-literal=key1=value1",
				"-n", "my-namespace",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: false,
		},
		{
			name:        "secret with multiple data entries",
			secretName:  "multi-secret",
			namespace:   "default",
			kubeContext: "",
			data:        map[string]string{"user": "admin", "pass": "secret123"},
			expectedArgs: []string{
				"create", "secret", "generic", "multi-secret",
				"-n", "default",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: false,
		},
		{
			name:        "secret with context",
			secretName:  "context-secret",
			namespace:   "prod",
			kubeContext: "my-cluster",
			data:        map[string]string{"token": "abc123"},
			expectedArgs: []string{
				"create", "secret", "generic", "context-secret",
				"-n", "prod",
				"--context", "my-cluster",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: true,
		},
		{
			name:        "empty namespace uses default",
			secretName:  "default-ns-secret",
			namespace:   "",
			kubeContext: "",
			data:        map[string]string{"key": "val"},
			expectedArgs: []string{
				"create", "secret", "generic", "default-ns-secret",
				"-n", "default",
				"--dry-run=client", "-o", "yaml",
			},
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildSecretArgs(tt.secretName, tt.namespace, tt.kubeContext, tt.data)

			if args[0] != "create" || args[1] != "secret" || args[2] != "generic" || args[3] != tt.secretName {
				t.Errorf("expected command starting with create secret generic %s, got: %v", tt.secretName, args[:4])
			}

			if args[len(args)-3] != "--dry-run=client" || args[len(args)-2] != "-o" || args[len(args)-1] != "yaml" {
				t.Errorf("expected --dry-run=client -o yaml at end, got: %v", args[len(args)-3:])
			}

			nsIdx := slices.Index(args, "-n")
			if nsIdx == -1 || args[nsIdx+1] != tt.namespace {
				if tt.namespace == "" && args[nsIdx+1] != "default" {
					t.Errorf("expected namespace 'default', got: %v", args)
				} else if tt.namespace != "" && args[nsIdx+1] != tt.namespace {
					t.Errorf("expected namespace %q, got: %v", tt.namespace, args)
				}
			}

			for key, value := range tt.data {
				expectedFlag := "--from-literal=" + key + "=" + value
				if !slices.Contains(args, expectedFlag) {
					t.Errorf("expected --from-literal=%s=%s in args, got: %v", key, value, args)
				}
			}

			if tt.expectContext {
				if !slices.Contains(args, "--context") {
					t.Error("expected --context flag in args")
				}
				ctxIdx := slices.Index(args, "--context")
				if args[ctxIdx+1] != tt.kubeContext {
					t.Errorf("expected context %q, got %q", tt.kubeContext, args[ctxIdx+1])
				}
			} else {
				if slices.Contains(args, "--context") {
					t.Error("did not expect --context flag in args")
				}
			}
		})
	}
}

func TestBuildSecretApplyArgs(t *testing.T) {
	tests := []struct {
		name          string
		kubeContext   string
		expectContext bool
		expectedArgs  []string
	}{
		{
			name:          "apply without context",
			kubeContext:   "",
			expectContext: false,
			expectedArgs:  []string{"apply", "-f", "-"},
		},
		{
			name:          "apply with context",
			kubeContext:   "my-cluster",
			expectContext: true,
			expectedArgs:  []string{"apply", "-f", "-", "--context", "my-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildSecretApplyArgs(tt.kubeContext)

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("expected %d args, got %d: %v", len(tt.expectedArgs), len(args), args)
			}

			for i, expected := range tt.expectedArgs {
				if args[i] != expected {
					t.Errorf("arg %d: expected %q, got %q", i, expected, args[i])
				}
			}

			if tt.expectContext {
				if !slices.Contains(args, "--context") {
					t.Error("expected --context flag in args")
				}
			}
		})
	}
}

func TestBuildConfigMapArgs(t *testing.T) {
	tests := []struct {
		name          string
		configMapName string
		namespace     string
		kubeContext   string
		data          map[string]string
		expectContext bool
	}{
		{
			name:          "basic configmap with single data entry",
			configMapName: "my-configmap",
			namespace:     "my-namespace",
			kubeContext:   "",
			data:          map[string]string{"key1": "value1"},
			expectContext: false,
		},
		{
			name:          "configmap with multiple data entries",
			configMapName: "multi-configmap",
			namespace:     "default",
			kubeContext:   "",
			data:          map[string]string{"config1": "val1", "config2": "val2"},
			expectContext: false,
		},
		{
			name:          "configmap with context",
			configMapName: "context-configmap",
			namespace:     "prod",
			kubeContext:   "my-cluster",
			data:          map[string]string{"setting": "value"},
			expectContext: true,
		},
		{
			name:          "empty namespace uses default",
			configMapName: "default-ns-configmap",
			namespace:     "",
			kubeContext:   "",
			data:          map[string]string{"key": "val"},
			expectContext: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildConfigMapArgs(tt.configMapName, tt.namespace, tt.kubeContext, tt.data)

			if args[0] != "create" || args[1] != "configmap" || args[2] != tt.configMapName {
				t.Errorf("expected command starting with create configmap %s, got: %v", tt.configMapName, args[:3])
			}

			if args[len(args)-3] != "--dry-run=client" || args[len(args)-2] != "-o" || args[len(args)-1] != "yaml" {
				t.Errorf("expected --dry-run=client -o yaml at end, got: %v", args[len(args)-3:])
			}

			nsIdx := slices.Index(args, "-n")
			if nsIdx == -1 {
				t.Error("expected -n flag in args")
			} else {
				expectedNS := tt.namespace
				if expectedNS == "" {
					expectedNS = "default"
				}
				if args[nsIdx+1] != expectedNS {
					t.Errorf("expected namespace %q, got %q", expectedNS, args[nsIdx+1])
				}
			}

			for key, value := range tt.data {
				expectedFlag := "--from-literal=" + key + "=" + value
				if !slices.Contains(args, expectedFlag) {
					t.Errorf("expected --from-literal=%s=%s in args, got: %v", key, value, args)
				}
			}

			if tt.expectContext {
				if !slices.Contains(args, "--context") {
					t.Error("expected --context flag in args")
				}
				ctxIdx := slices.Index(args, "--context")
				if args[ctxIdx+1] != tt.kubeContext {
					t.Errorf("expected context %q, got %q", tt.kubeContext, args[ctxIdx+1])
				}
			} else {
				if slices.Contains(args, "--context") {
					t.Error("did not expect --context flag in args")
				}
			}
		})
	}
}

func TestBuildConfigMapApplyArgs(t *testing.T) {
	tests := []struct {
		name          string
		kubeContext   string
		expectContext bool
		expectedArgs  []string
	}{
		{
			name:          "apply without context",
			kubeContext:   "",
			expectContext: false,
			expectedArgs:  []string{"apply", "-f", "-"},
		},
		{
			name:          "apply with context",
			kubeContext:   "my-cluster",
			expectContext: true,
			expectedArgs:  []string{"apply", "-f", "-", "--context", "my-cluster"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildConfigMapApplyArgs(tt.kubeContext)

			if len(args) != len(tt.expectedArgs) {
				t.Errorf("expected %d args, got %d: %v", len(tt.expectedArgs), len(args), args)
			}

			for i, expected := range tt.expectedArgs {
				if args[i] != expected {
					t.Errorf("arg %d: expected %q, got %q", i, expected, args[i])
				}
			}

			if tt.expectContext {
				if !slices.Contains(args, "--context") {
					t.Error("expected --context flag in args")
				}
			}
		})
	}
}
