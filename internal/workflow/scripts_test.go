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
		input    string
		expected string
	}{
		{"my-config", "my-config"},
		{"my_config", "my-config"},
		{"my.config", "my-config"},
		{"MyConfig", "myconfig"},
		{"my_config.map", "my-config-map"},
		{"my/config", "my-config"},
		{"my/branch.name", "my-branch-name"},
		{"MY_BRANCH", "my-branch"},
		{"feature/123-add-thing", "feature-123-add-thing"},
		{"feature@special!", "feature-special"},
		{"123-branch", "branch-123-branch"},
		{"---test---", "test"},
		{"", "default"},
	}

	for _, tt := range tests {
		result := sanitizeName(tt.input)
		assert.Equal(t, tt.expected, result, "sanitizeName should return expected value")
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
}

func TestBuildCredentialMounts(t *testing.T) {
	mounts := buildCredentialMounts()
	assert.Len(t, mounts, 3, "should have 3 credential mounts")
	assert.Equal(t, "github-credentials", mounts[0]["name"], "first mount name should match")
	assert.Equal(t, "opencode-credentials", mounts[1]["name"], "second mount name should match")
	assert.Equal(t, "pulumi-credentials", mounts[2]["name"], "third mount name should match")
}

func TestBuildCredentialVolumes(t *testing.T) {
	volumes := buildCredentialVolumes()
	assert.Len(t, volumes, 3, "should have 3 credential volumes")
	assert.Equal(t, "github-credentials", volumes[0]["name"], "first volume name should match")
	assert.Equal(t, "opencode-credentials", volumes[1]["name"], "second volume name should match")
	assert.Equal(t, "pulumi-credentials", volumes[2]["name"], "third volume name should match")

	// Verify Pulumi secret is optional
	pulumiVol, ok := volumes[2]["secret"].(map[string]interface{})
	assert.True(t, ok, "pulumi volume should have secret map")
	assert.Equal(t, true, pulumiVol["optional"], "pulumi secret should be optional")
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
