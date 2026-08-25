package workflow

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zon/ralph/internal/config"
	"github.com/zon/ralph/internal/k8s"
)

// buildParameters builds workflow parameters from the params map
func buildParameters(params map[string]string) []map[string]interface{} {
	allParams := []string{"project-path", "instructions-md", "comment-body", "pr-number", "base-branch"}
	parameters := make([]map[string]interface{}, 0, len(allParams))
	for _, name := range allParams {
		parameters = append(parameters, map[string]interface{}{"name": name, "value": params[name]})
	}
	return parameters
}

func buildConfigMapVolumeMount(name string, destFile, destDir string, index int) map[string]interface{} {
	return buildVolumeMount(name, destFile, destDir, index, "/configmaps/"+name)
}

func buildSecretVolumeMount(name string, destFile, destDir string, index int) map[string]interface{} {
	return buildVolumeMount(name, destFile, destDir, index, "/secrets/"+name)
}

func buildVolumeMount(name string, destFile, destDir string, index int, fallbackMountPath string) map[string]interface{} {
	mount := map[string]interface{}{
		"name":     sanitizeName(name),
		"readOnly": true,
	}
	switch {
	case destFile != "":
		mount["mountPath"] = absPath(destFile)
		mount["subPath"] = filepath.Base(destFile)
		mount["name"] = fmt.Sprintf("%s-%d", sanitizeName(name), index)
	case destDir != "":
		mount["mountPath"] = absPath(destDir)
	default:
		mount["mountPath"] = fallbackMountPath
	}
	return mount
}

func absPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return "/workspace/" + path
}

func buildConfigMapVolume(name string, destFile string, index int) map[string]interface{} {
	volumeName := sanitizeName(name)
	if destFile != "" {
		volumeName = fmt.Sprintf("%s-%d", sanitizeName(name), index)
		return map[string]interface{}{
			"name": volumeName,
			"configMap": map[string]interface{}{
				"name": name,
				"items": []map[string]interface{}{
					{"key": filepath.Base(destFile), "path": filepath.Base(destFile)},
				},
			},
		}
	}
	return map[string]interface{}{
		"name":      volumeName,
		"configMap": map[string]interface{}{"name": name},
	}
}

func buildSecretVolume(name string, destFile string, index int) map[string]interface{} {
	volumeName := sanitizeName(name)
	if destFile != "" {
		volumeName = fmt.Sprintf("%s-%d", sanitizeName(name), index)
		return map[string]interface{}{
			"name": volumeName,
			"secret": map[string]interface{}{
				"secretName": name,
				"items": []map[string]interface{}{
					{"key": filepath.Base(destFile), "path": filepath.Base(destFile)},
				},
			},
		}
	}
	return map[string]interface{}{
		"name":   volumeName,
		"secret": map[string]interface{}{"secretName": name},
	}
}

func buildCredentialVolumes() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name": "github-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.GitHubSecretName,
			},
		},
		{
			"name": "opencode-credentials",
			"secret": map[string]interface{}{
				"secretName": k8s.OpenCodeSecretName,
			},
		},
	}
}

func buildCredentialMounts() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "github-credentials", "mountPath": "/secrets/github", "readOnly": true},
		{"name": "opencode-credentials", "mountPath": "/secrets/opencode", "readOnly": true},
	}
}

func buildVolumeMounts(configMaps []config.ConfigMapMount, secrets []config.SecretMount) []map[string]interface{} {
	mounts := buildCredentialMounts()

	for i, cm := range configMaps {
		mounts = append(mounts, buildConfigMapVolumeMount(cm.Name, cm.DestFile, cm.DestDir, i))
	}

	for i, secret := range secrets {
		mounts = append(mounts, buildSecretVolumeMount(secret.Name, secret.DestFile, secret.DestDir, i))
	}

	return mounts
}

func buildVolumes(configMaps []config.ConfigMapMount, secrets []config.SecretMount) []map[string]interface{} {
	volumes := buildCredentialVolumes()

	for i, cm := range configMaps {
		volumes = append(volumes, buildConfigMapVolume(cm.Name, cm.DestFile, i))
	}

	for i, secret := range secrets {
		volumes = append(volumes, buildSecretVolume(secret.Name, secret.DestFile, i))
	}

	return volumes
}

// sanitizeName sanitizes a name for use as a Kubernetes volume/resource name.
func sanitizeName(name string) string {
	sanitized := strings.ReplaceAll(name, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ToLower(sanitized)

	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('-')
		}
	}

	sanitized = result.String()
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return "default"
	}

	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "branch-" + sanitized
	}

	return sanitized
}
