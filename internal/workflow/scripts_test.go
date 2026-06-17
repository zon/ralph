package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zon/ralph/internal/config"
)

func TestBuildVolumeMounts_WorkspacePrefix(t *testing.T) {
	configMaps := []config.ConfigMapMount{
		{Name: "my-config", DestFile: "config/main.yaml"},
		{Name: "my-config-dir", DestDir: "config/extra"},
	}
	secrets := []config.SecretMount{
		{Name: "my-secret", DestFile: "config/secrets.yaml"},
		{Name: "my-secret-dir", DestDir: "config/auth"},
	}

	mounts := buildVolumeMounts(configMaps, secrets)

	expected := map[string]string{
		"my-config-0":   "/workspace/config/main.yaml",
		"my-config-dir": "/workspace/config/extra",
		"my-secret-1":   "/workspace/config/secrets.yaml",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeName(tt.input)
			assert.Equal(t, tt.expected, result, "sanitizeName should return expected value")
		})
	}
}

func TestBuildConfigMapVolumeMount(t *testing.T) {
	tests := []struct {
		name     string
		destFile string
		destDir  string
		index    int
		check    func(t *testing.T, mount map[string]interface{})
	}{
		{
			name:     "my-config",
			destFile: "config/main.yaml",
			destDir:  "",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "my-config-0", mount["name"], "name should match")
				assert.Equal(t, "/workspace/config/main.yaml", mount["mountPath"], "mountPath should match")
				assert.Equal(t, "main.yaml", mount["subPath"], "subPath should match")
			},
		},
		{
			name:     "my-config",
			destFile: "/etc/id/backend/config.json",
			destDir:  "",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "my-config-0", mount["name"], "name should match")
				assert.Equal(t, "/etc/id/backend/config.json", mount["mountPath"], "mountPath should use absolute path")
				assert.Equal(t, "config.json", mount["subPath"], "subPath should match")
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "config/extra",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "my-config", mount["name"], "name should match")
				assert.Equal(t, "/workspace/config/extra", mount["mountPath"], "mountPath should match")
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "/etc/id/backend",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "my-config", mount["name"], "name should match")
				assert.Equal(t, "/etc/id/backend", mount["mountPath"], "mountPath should use absolute path")
			},
		},
		{
			name:     "my-config",
			destFile: "",
			destDir:  "",
			index:    0,
			check: func(t *testing.T, mount map[string]interface{}) {
				assert.Equal(t, "/configmaps/my-config", mount["mountPath"], "mountPath should match")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount := buildConfigMapVolumeMount(tt.name, tt.destFile, tt.destDir, tt.index)
			tt.check(t, mount)
		})
	}
}

func TestBuildSecretVolumeMount(t *testing.T) {
	mount := buildSecretVolumeMount("my-secret", "secrets.yaml", "", 0)
	assert.Equal(t, "my-secret-0", mount["name"], "name should match")
	assert.Equal(t, "/workspace/secrets.yaml", mount["mountPath"], "mountPath should match")

	mount = buildSecretVolumeMount("my-secret", "/etc/id/backend/secrets.json", "", 0)
	assert.Equal(t, "my-secret-0", mount["name"], "name should match")
	assert.Equal(t, "/etc/id/backend/secrets.json", mount["mountPath"], "mountPath should use absolute path")
}

func TestBuildCredentialMounts(t *testing.T) {
	mounts := buildCredentialMounts()
	assert.Len(t, mounts, 2, "should have 2 credential mounts")
	assert.Equal(t, "github-credentials", mounts[0]["name"], "first mount name should match")
	assert.Equal(t, "opencode-credentials", mounts[1]["name"], "second mount name should match")
}

func TestBuildCredentialVolumes(t *testing.T) {
	volumes := buildCredentialVolumes()
	assert.Len(t, volumes, 2, "should have 2 credential volumes")
	assert.Equal(t, "github-credentials", volumes[0]["name"], "first volume name should match")
	assert.Equal(t, "opencode-credentials", volumes[1]["name"], "second volume name should match")
}

func TestBuildConfigMapVolume(t *testing.T) {
	vol := buildConfigMapVolume("my-config", "config/main.yaml", 0)
	assert.Equal(t, "my-config-0", vol["name"], "name should match")

	vol = buildConfigMapVolume("my-config", "", 0)
	assert.Equal(t, "my-config", vol["name"], "name should match")
}

func TestBuildSecretVolume(t *testing.T) {
	vol := buildSecretVolume("my-secret", "secrets.yaml", 0)
	assert.Equal(t, "my-secret-0", vol["name"], "name should match")

	vol = buildSecretVolume("my-secret", "", 0)
	assert.Equal(t, "my-secret", vol["name"], "name should match")
}
