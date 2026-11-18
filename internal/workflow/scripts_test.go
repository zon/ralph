package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
)

func TestBuildVolumeSetWorkspacePrefix(t *testing.T) {
	configMaps := []config.ConfigMapMount{
		{Name: "my-config", DestFile: "config/main.yaml"},
		{Name: "my-config-dir", DestDir: "config/extra"},
	}
	secrets := []config.SecretMount{
		{Name: "my-secret", DestFile: "config/secrets.yaml"},
		{Name: "my-secret-dir", DestDir: "config/auth"},
	}

	mounts, _ := buildVolumeSet(configMaps, secrets, "")

	expected := map[string]string{
		"my-config":     "/workspace/config/main.yaml",
		"my-config-dir": "/workspace/config/extra",
		"my-secret":     "/workspace/config/secrets.yaml",
		"my-secret-dir": "/workspace/config/auth",
	}

	for _, mount := range mounts {
		name, _ := mount["name"].(string)
		mountPath, _ := mount["mountPath"].(string)
		if want, ok := expected[name]; ok {
			assert.Equal(t, want, mountPath, "mount %q should have correct mountPath", name)
		}
	}
}

// Every mount needs a volume with the same name, and Kubernetes rejects a pod
// that repeats a volume name. A lossy sanitized name, a ConfigMap and a Secret
// with the same name, and the static credential volumes all share one name
// space.
func TestBuildVolumeSetUniqueNames(t *testing.T) {
	configMaps := []config.ConfigMapMount{
		{Name: "a.b", DestDir: "config/one"},
		{Name: "a_b", DestDir: "config/two"},
		{Name: "github-credentials", DestFile: "config/creds.yaml"},
	}
	secrets := []config.SecretMount{
		{Name: "a.b", DestFile: "secrets/one.yaml"},
		{Name: "github_credentials", DestDir: "secrets/creds"},
	}

	mounts, volumes := buildVolumeSet(configMaps, secrets, "github.credentials")

	mountNames := volumeNamesOf(mounts)
	volumeNames := volumeNamesOf(volumes)

	assert.Len(t, mountNames, len(mounts), "every mount should have a name")
	assert.Len(t, volumeNames, len(volumes), "every volume should have a name")
	seen := map[string]bool{}
	for _, name := range volumeNames {
		assert.NotEmpty(t, name, "volume name should not be empty")
		assert.False(t, seen[name], "volume name %q should be unique", name)
		seen[name] = true
	}
	assert.ElementsMatch(t, mountNames, volumeNames, "mounts and volumes should share names")
	assert.Contains(t, volumeNames, "a-b", "the first sanitized name is kept")
	assert.Contains(t, volumeNames, "a-b-2", "a colliding sanitized name gets a suffix")
	assert.Contains(t, volumeNames, "github-credentials", "the static credential volume keeps its name")
}

func volumeNamesOf(entries []map[string]interface{}) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name, _ := entry["name"].(string)
		names = append(names, name)
	}
	return names
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "hyphen", input: "my-config", expected: "my-config"},
		{name: "underscore", input: "my_config", expected: "my-config"},
		{name: "dot", input: "my.config", expected: "my-config"},
		{name: "camelcase", input: "MyConfig", expected: "myconfig"},
		{name: "mixed_separators", input: "my_config.map", expected: "my-config-map"},
		{name: "slash", input: "my/config", expected: "my-config"},
		{name: "slash_and_dot", input: "my/branch.name", expected: "my-branch-name"},
		{name: "uppercase_with_underscore", input: "MY_BRANCH", expected: "my-branch"},
		{name: "feature_branch_with_numbers", input: "feature/123-add-thing", expected: "feature-123-add-thing"},
		{name: "special_characters", input: "feature@special!", expected: "feature-special"},
		{name: "leading_digits", input: "123-branch", expected: "branch-123-branch"},
		{name: "leading_only_dashes", input: "---test---", expected: "test"},
		{name: "empty_string", input: "", expected: "default"},
		{name: "long_name", input: strings.Repeat("a", 100), expected: strings.Repeat("a", 58)},
		{name: "long_name_trailing_separator", input: strings.Repeat("a", 57) + "-" + strings.Repeat("b", 10), expected: strings.Repeat("a", 57)},
		{name: "long_leading_digits", input: "1" + strings.Repeat("a", 100), expected: "branch-1" + strings.Repeat("a", 50)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeName(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizeName should return expected value")
		})
	}
}

func TestAbsPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{name: "relative", path: "config/main.yaml", expected: "/workspace/config/main.yaml"},
		{name: "relative_dot", path: "./config/main.yaml", expected: "/workspace/config/main.yaml"},
		{name: "relative_parent", path: "config/../main.yaml", expected: "/workspace/main.yaml"},
		{name: "relative_escape", path: "../etc/cron.d/job", expected: "/workspace/etc/cron.d/job"},
		{name: "relative_escape_deep", path: "a/../../etc/passwd", expected: "/workspace/etc/passwd"},
		{name: "absolute", path: "/etc/id/backend/config.json", expected: "/etc/id/backend/config.json"},
		{name: "absolute_parent", path: "/workspace/../etc/passwd", expected: "/etc/passwd"},
		{name: "absolute_dot", path: "/workspace/config/./main.yaml", expected: "/workspace/config/main.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, absPath(tt.path), "absPath should return a clean path")
		})
	}
}

func TestBuildVolumeMount(t *testing.T) {
	tests := []struct {
		name       string
		destFile   string
		destDir    string
		volumeName string
		fallback   string
		wantPath   string
		wantSub    string
	}{
		{
			name:       "dest file",
			destFile:   "config/main.yaml",
			volumeName: "my-config",
			fallback:   "/configmaps/my-config",
			wantPath:   "/workspace/config/main.yaml",
			wantSub:    "main.yaml",
		},
		{
			name:       "absolute dest file",
			destFile:   "/etc/id/backend/config.json",
			volumeName: "my-config",
			fallback:   "/configmaps/my-config",
			wantPath:   "/etc/id/backend/config.json",
			wantSub:    "config.json",
		},
		{
			name:       "dest dir",
			destDir:    "config/extra",
			volumeName: "my-config",
			fallback:   "/configmaps/my-config",
			wantPath:   "/workspace/config/extra",
		},
		{
			name:       "absolute dest dir",
			destDir:    "/etc/id/backend",
			volumeName: "my-config",
			fallback:   "/configmaps/my-config",
			wantPath:   "/etc/id/backend",
		},
		{
			name:       "fallback",
			volumeName: "my-config",
			fallback:   "/configmaps/my-config",
			wantPath:   "/configmaps/my-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := buildVolumeMount(tt.volumeName, tt.destFile, tt.destDir, tt.fallback)
			assert.Equal(t, tt.volumeName, mount["name"], "name should match")
			assert.Equal(t, tt.wantPath, mount["mountPath"], "mountPath should match")
			assert.Equal(t, true, mount["readOnly"], "mount should be read-only")
			if tt.wantSub != "" {
				assert.Equal(t, tt.wantSub, mount["subPath"], "subPath should match")
			} else {
				assert.NotContains(t, mount, "subPath", "mount should not have a subPath")
			}
		})
	}
}

func TestBuildVolumeSetCredentials(t *testing.T) {
	tests := []struct {
		name           string
		openCodeSecret string
		wantName       string
		wantSecretName string
	}{
		{
			name:           "unset mounts the default Secret",
			wantName:       "opencode-credentials",
			wantSecretName: "opencode-credentials",
		},
		{
			name:           "set mounts the named Secret",
			openCodeSecret: "opencode-ai-gateway",
			wantName:       "opencode-ai-gateway",
			wantSecretName: "opencode-ai-gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mounts, volumes := buildVolumeSet(nil, nil, tt.openCodeSecret)
			require.Len(t, mounts, 2, "should have 2 credential mounts")
			require.Len(t, volumes, 2, "should have 2 credential volumes")

			assert.Equal(t, "github-credentials", mounts[0]["name"], "github mount name should be unaffected")
			assert.Equal(t, "/secrets/github", mounts[0]["mountPath"], "github mount path should be unaffected")
			assert.Equal(t, tt.wantName, mounts[1]["name"], "opencode mount name should match the mounted Secret")
			assert.Equal(t, "/secrets/opencode", mounts[1]["mountPath"], "opencode mount path should be fixed")
			assert.Equal(t, true, mounts[1]["readOnly"], "opencode mount should be read-only")

			assert.Equal(t, "github-credentials", volumes[0]["name"], "github volume name should be unaffected")
			githubSecret := volumes[0]["secret"].(map[string]interface{})
			assert.Equal(t, k8s.GitHubSecretName, githubSecret["secretName"], "github Secret should be unaffected")
			assert.Equal(t, tt.wantName, volumes[1]["name"], "opencode volume name should match the mounted Secret")
			openCodeSecret := volumes[1]["secret"].(map[string]interface{})
			assert.Equal(t, tt.wantSecretName, openCodeSecret["secretName"], "opencode volume should mount the named Secret")
		})
	}
}

func TestBuildConfigMapVolume(t *testing.T) {
	vol := buildConfigMapVolume("my-config-0", "my-config", "config/main.yaml")
	assert.Equal(t, "my-config-0", vol["name"], "name should match")
	configMap := vol["configMap"].(map[string]interface{})
	assert.Equal(t, "my-config", configMap["name"], "ConfigMap should match")
	items := configMap["items"].([]map[string]interface{})
	require.Len(t, items, 1, "a file mount should select one item")
	assert.Equal(t, "main.yaml", items[0]["key"], "item key should match")
	assert.Equal(t, "main.yaml", items[0]["path"], "item path should match")

	vol = buildConfigMapVolume("my-config", "my-config", "")
	configMap = vol["configMap"].(map[string]interface{})
	assert.NotContains(t, configMap, "items", "a directory mount should not select items")
}

func TestBuildSecretVolume(t *testing.T) {
	vol := buildSecretVolume("my-secret-0", "my-secret", "secrets.yaml")
	assert.Equal(t, "my-secret-0", vol["name"], "name should match")
	secret := vol["secret"].(map[string]interface{})
	assert.Equal(t, "my-secret", secret["secretName"], "Secret should match")
	items := secret["items"].([]map[string]interface{})
	require.Len(t, items, 1, "a file mount should select one item")
	assert.Equal(t, "secrets.yaml", items[0]["key"], "item key should match")
	assert.Equal(t, "secrets.yaml", items[0]["path"], "item path should match")

	vol = buildSecretVolume("my-secret", "my-secret", "")
	secret = vol["secret"].(map[string]interface{})
	assert.NotContains(t, secret, "items", "a directory mount should not select items")
}
