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
	allParams := []string{"project-path", "instructions-md", "base-branch"}
	parameters := make([]map[string]interface{}, 0, len(allParams))
	for _, name := range allParams {
		parameters = append(parameters, map[string]interface{}{"name": name, "value": params[name]})
	}
	return parameters
}

// buildVolumeSet builds the pod volumes and the container mounts that
// reference them. Names must be unique across every mount and volume, so both
// lists are built together from one allocator: building them separately lets a
// ConfigMap, a Secret, and the static credential volumes collide on a name
// after sanitizing, which makes the pod invalid.
func buildVolumeSet(configMaps []config.ConfigMapMount, secrets []config.SecretMount, openCodeSecret string) ([]map[string]interface{}, []map[string]interface{}) {
	names := newVolumeNameAllocator()
	openCodeSecret = resolveOpenCodeSecretName(openCodeSecret)

	githubName := names.allocate("github-credentials")
	openCodeName := names.allocate(sanitizeName(openCodeSecret))

	mounts := []map[string]interface{}{
		{"name": githubName, "mountPath": "/secrets/github", "readOnly": true},
		{"name": openCodeName, "mountPath": "/secrets/opencode", "readOnly": true},
	}
	volumes := []map[string]interface{}{
		{"name": githubName, "secret": map[string]interface{}{"secretName": k8s.GitHubSecretName}},
		{"name": openCodeName, "secret": map[string]interface{}{"secretName": openCodeSecret}},
	}

	for _, cm := range configMaps {
		name := names.allocate(sanitizeName(cm.Name))
		mounts = append(mounts, buildVolumeMount(name, cm.DestFile, cm.DestDir, "/configmaps/"+cm.Name))
		volumes = append(volumes, buildConfigMapVolume(name, cm.Name, cm.DestFile))
	}

	for _, secret := range secrets {
		name := names.allocate(sanitizeName(secret.Name))
		mounts = append(mounts, buildVolumeMount(name, secret.DestFile, secret.DestDir, "/secrets/"+secret.Name))
		volumes = append(volumes, buildSecretVolume(name, secret.Name, secret.DestFile))
	}

	return mounts, volumes
}

// maxVolumeNameLength is the Kubernetes DNS label limit for a volume name.
const maxVolumeNameLength = 63

// volumeNameAllocator hands out unique volume names. A sanitized name can
// collide with another entry or with a static credential volume, so a numeric
// suffix is appended until the name is free.
type volumeNameAllocator struct {
	used map[string]bool
}

func newVolumeNameAllocator() *volumeNameAllocator {
	return &volumeNameAllocator{used: map[string]bool{}}
}

func (a *volumeNameAllocator) allocate(base string) string {
	base = truncateVolumeName(base, maxVolumeNameLength)
	candidate := base
	for suffix := 2; a.used[candidate]; suffix++ {
		s := fmt.Sprintf("-%d", suffix)
		candidate = truncateVolumeName(base, maxVolumeNameLength-len(s)) + s
	}
	a.used[candidate] = true
	return candidate
}

// truncateVolumeName keeps a name within limit characters without leaving a
// trailing hyphen, which Kubernetes rejects.
func truncateVolumeName(name string, limit int) string {
	if len(name) > limit {
		name = name[:limit]
	}
	name = strings.TrimRight(name, "-")
	if name == "" {
		return "volume"
	}
	return name
}

func buildVolumeMount(name, destFile, destDir, fallbackMountPath string) map[string]interface{} {
	mount := map[string]interface{}{
		"name":     name,
		"readOnly": true,
	}
	switch {
	case destFile != "":
		mount["mountPath"] = absPath(destFile)
		mount["subPath"] = filepath.Base(destFile)
	case destDir != "":
		mount["mountPath"] = absPath(destDir)
	default:
		mount["mountPath"] = fallbackMountPath
	}
	return mount
}

// absPath resolves a mount destination to a clean absolute path. A relative
// destination is resolved inside the workspace and cannot escape it through
// ".." segments. An absolute destination is accepted as written because the
// config documents mounts outside the workspace, such as /tools, but it is
// still cleaned.
func absPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join("/workspace", filepath.Clean("/"+path))
}

func buildConfigMapVolume(volumeName, name, destFile string) map[string]interface{} {
	configMap := map[string]interface{}{"name": name}
	if destFile != "" {
		configMap["items"] = []map[string]interface{}{
			{"key": filepath.Base(destFile), "path": filepath.Base(destFile)},
		}
	}
	return map[string]interface{}{
		"name":      volumeName,
		"configMap": configMap,
	}
}

func buildSecretVolume(volumeName, name, destFile string) map[string]interface{} {
	secret := map[string]interface{}{"secretName": name}
	if destFile != "" {
		secret["items"] = []map[string]interface{}{
			{"key": filepath.Base(destFile), "path": filepath.Base(destFile)},
		}
	}
	return map[string]interface{}{
		"name":   volumeName,
		"secret": secret,
	}
}

// resolveOpenCodeSecretName returns the Secret the workflow mounts for
// OpenCode. An unset name takes the default, the Secret ralph setup writes.
func resolveOpenCodeSecretName(openCodeSecret string) string {
	if openCodeSecret == "" {
		return k8s.OpenCodeSecretName
	}
	return openCodeSecret
}

// buildResources returns the container resources block from the configured
// requests and limits, omitting entries that are unset. An empty result means
// no resources were configured.
func buildResources(r config.WorkflowResources) map[string]interface{} {
	resources := map[string]interface{}{}
	if req := resourceListMap(r.Requests); len(req) > 0 {
		resources["requests"] = req
	}
	if lim := resourceListMap(r.Limits); len(lim) > 0 {
		resources["limits"] = lim
	}
	return resources
}

// resourceListMap returns a resources entry mapping only the set quantities.
func resourceListMap(l config.ResourceList) map[string]interface{} {
	entry := map[string]interface{}{}
	if l.Memory != "" {
		entry["memory"] = l.Memory
	}
	if l.CPU != "" {
		entry["cpu"] = l.CPU
	}
	return entry
}

// maxSanitizedNameLength bounds a sanitized name so callers that use it as a
// Kubernetes volume name stay within the 63 character DNS label limit. Mount
// builders append a "-<index>" suffix, so five characters are kept in reserve.
const maxSanitizedNameLength = 58

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

	if len(sanitized) > maxSanitizedNameLength {
		sanitized = strings.Trim(sanitized[:maxSanitizedNameLength], "-")
	}

	return sanitized
}
