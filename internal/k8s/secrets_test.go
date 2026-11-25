package k8s

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSecretManifest(t *testing.T) {
	tests := []struct {
		name              string
		secretName        string
		namespace         string
		data              map[string]string
		expectedNamespace string
	}{
		{
			name:              "basic secret with single data entry",
			secretName:        "my-secret",
			namespace:         "my-namespace",
			data:              map[string]string{"key1": "value1"},
			expectedNamespace: "my-namespace",
		},
		{
			name:              "secret with multiple data entries",
			secretName:        "multi-secret",
			namespace:         "default",
			data:              map[string]string{"user": "admin", "pass": "secret123"},
			expectedNamespace: "default",
		},
		{
			name:              "empty namespace uses default",
			secretName:        "default-ns-secret",
			namespace:         "",
			data:              map[string]string{"key": "val"},
			expectedNamespace: "default",
		},
		{
			name:              "empty data",
			secretName:        "empty-secret",
			namespace:         "argo",
			data:              map[string]string{},
			expectedNamespace: "argo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := buildSecretManifest(tt.secretName, tt.namespace, tt.data)
			require.NoError(t, err)

			var got struct {
				APIVersion string `json:"apiVersion"`
				Kind       string `json:"kind"`
				Metadata   struct {
					Name      string `json:"name"`
					Namespace string `json:"namespace"`
				} `json:"metadata"`
				Type string            `json:"type"`
				Data map[string]string `json:"data"`
			}
			require.NoError(t, json.Unmarshal(manifest, &got))

			assert.Equal(t, "v1", got.APIVersion)
			assert.Equal(t, "Secret", got.Kind)
			assert.Equal(t, "Opaque", got.Type)
			assert.Equal(t, tt.secretName, got.Metadata.Name)
			assert.Equal(t, tt.expectedNamespace, got.Metadata.Namespace)
			assert.Len(t, got.Data, len(tt.data))

			for key, value := range tt.data {
				encoded, ok := got.Data[key]
				require.True(t, ok, "manifest should contain key %q", key)
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				require.NoError(t, err)
				assert.Equal(t, value, string(decoded))
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

			assert.Len(t, args, len(tt.expectedArgs), "Args length should match")
			for i, expected := range tt.expectedArgs {
				assert.Equal(t, expected, args[i], "Arg %d should match", i)
			}

			if tt.expectContext {
				assert.True(t, slices.Contains(args, "--context"), "Should contain --context flag")
			}
		})
	}
}

func TestBuildGetSecretDataArgs(t *testing.T) {
	tests := []struct {
		name         string
		secretName   string
		namespace    string
		kubeContext  string
		expectedArgs []string
	}{
		{
			name:         "basic secret read",
			secretName:   "github-credentials",
			namespace:    "argo",
			kubeContext:  "",
			expectedArgs: []string{"get", "secret", "github-credentials", "-n", "argo", "-o", "json"},
		},
		{
			name:         "secret read with context",
			secretName:   "github-credentials",
			namespace:    "argo",
			kubeContext:  "staging",
			expectedArgs: []string{"get", "secret", "github-credentials", "-n", "argo", "-o", "json", "--context", "staging"},
		},
		{
			name:         "empty namespace uses default",
			secretName:   "github-credentials",
			namespace:    "",
			kubeContext:  "",
			expectedArgs: []string{"get", "secret", "github-credentials", "-n", "default", "-o", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildGetSecretDataArgs(tt.secretName, tt.namespace, tt.kubeContext)

			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestBuildSecretExistsArgs(t *testing.T) {
	tests := []struct {
		name         string
		secretName   string
		namespace    string
		kubeContext  string
		expectedArgs []string
	}{
		{
			name:         "basic secret lookup",
			secretName:   "github-credentials",
			namespace:    "argo",
			kubeContext:  "",
			expectedArgs: []string{"get", "secret", "github-credentials", "-n", "argo"},
		},
		{
			name:         "secret lookup with context",
			secretName:   "github-credentials",
			namespace:    "argo",
			kubeContext:  "staging",
			expectedArgs: []string{"get", "secret", "github-credentials", "-n", "argo", "--context", "staging"},
		},
		{
			name:         "empty namespace uses default",
			secretName:   "github-credentials",
			namespace:    "",
			kubeContext:  "",
			expectedArgs: []string{"get", "secret", "github-credentials", "-n", "default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildSecretExistsArgs(tt.secretName, tt.namespace, tt.kubeContext)

			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}

func TestParseSecretData(t *testing.T) {
	encode := func(value string) string {
		return base64.StdEncoding.EncodeToString([]byte(value))
	}

	tests := []struct {
		name    string
		raw     string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "decodes values",
			raw:  `{"data":{"token":"` + encode("gho_token") + `","other":"` + encode("value") + `"}}`,
			want: map[string]string{"token": "gho_token", "other": "value"},
		},
		{
			name: "empty data",
			raw:  `{"data":{}}`,
			want: map[string]string{},
		},
		{
			name:    "invalid json",
			raw:     `not json`,
			wantErr: true,
		},
		{
			name:    "invalid base64",
			raw:     `{"data":{"token":"!"}}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSecretData([]byte(tt.raw))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
